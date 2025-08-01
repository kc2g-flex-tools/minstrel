package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/hb9fxq/flexlib-go/vita"
	"github.com/kc2g-flex-tools/flexclient"
	"github.com/kc2g-flex-tools/minstrel/audio"
	"github.com/kc2g-flex-tools/minstrel/radioshim"
	"github.com/kc2g-flex-tools/minstrel/ui"
)

func getClientID() (string, bool) {
	fn, err := xdg.DataFile("minstrel/client_id")
	if err != nil {
		log.Println(err)
		return "", false
	}
	file, err := os.Open(fn)
	if err != nil {
		log.Println(err)
		return "", false
	}
	defer file.Close()
	contents, err := io.ReadAll(file)
	if err != nil {
		log.Println(err)
		return "", false
	}
	return strings.TrimSuffix(string(contents), "\n"), true
}

func setClientID(clientID string) error {
	fn, _ := xdg.DataFile("minstrel/client_id")
	file, err := os.Create(fn)
	if err != nil {
		return err
	}
	defer file.Close()
	fmt.Fprintf(file, "%s\n", clientID)
	return nil
}

type WFState struct {
	width      uint16
	bins       []uint16
	timecode   uint32
	binsFilled uint16
	dataLow    float64
	dataHigh   float64
}

type RadioState struct {
	mu              sync.RWMutex
	FlexClient      *flexclient.FlexClient
	UI              *ui.UI
	Audio           *audio.Audio
	ClientID        string
	WaterfallStream uint32
	AudioStream     uint32
	WFState         WFState
	Slices          map[string]radioshim.SliceData
}

func NewRadioState(fc *flexclient.FlexClient, u *ui.UI, audioCtx *audio.Audio) *RadioState {
	rs := &RadioState{
		FlexClient: fc,
		UI:         u,
		Audio:      audioCtx,
	}
	u.RadioShim = rs
	return rs
}

func (rs *RadioState) Run(ctx context.Context) {
	fc := rs.FlexClient
	waterfalls := fc.Subscribe(flexclient.Subscription{
		Prefix:  "display waterfall ",
		Updates: make(chan flexclient.StateUpdate, 100),
	})
	streams := fc.Subscribe(flexclient.Subscription{
		Prefix:  "stream ",
		Updates: make(chan flexclient.StateUpdate, 100),
	})

	ClientUUID, uuidOK := getClientID()
	if uuidOK {
		fc.SendAndWait("client gui " + ClientUUID)
		fmt.Println("connected with client ID " + ClientUUID)
	} else {
		res := fc.SendAndWait("client gui")
		if res.Error != 0 {
			log.Fatal(res)
		}
		ClientUUID := res.Message
		err := setClientID(ClientUUID)
		log.Println("got new client ID " + ClientUUID)
		if err != nil {
			log.Println(err)
		}
	}
	rs.ClientID = "0x" + fc.ClientID()

	fc.SendAndWait("client program Minstrel")
	fc.SendAndWait("client station " + strings.ReplaceAll(config.Station, " ", "\x7f"))

	if config.Profile != "" {
		fc.SendAndWait("profile global load " + config.Profile)
	}
	fc.SendAndWait("sub radio all")
	fc.SendAndWait("sub slice all")

	err := fc.InitUDP()
	if err != nil {
		log.Fatal(err)
	}
	go fc.RunUDP()

	notif := make(chan struct{}, 1)
	fc.SetStateNotify(notif)
	vita := make(chan flexclient.VitaPacket, 10)
	fc.SetVitaChan(vita)
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-notif:
			rs.updateGUI()
		case <-ticker.C:
			rs.updateGUI()
		case st := <-waterfalls.Updates:
			if st.CurrentState["client_handle"] == rs.ClientID {
				streamStr := strings.TrimPrefix(st.Object, "display waterfall 0x")
				streamId, err := strconv.ParseUint(streamStr, 16, 32)
				if err != nil {
					log.Println(err)
				} else {
					if rs.WaterfallStream == 0 {
						log.Println("my waterfall is", streamStr)
						rs.WaterfallStream = uint32(streamId)
						wf, _ := rs.getWaterfallAndPan()
						fc.PanSet(wf["panadapter"], flexclient.Object{"xpixels": "1000"})
					}
					center, _ := strconv.ParseFloat(st.CurrentState["center"], 64)
					span, _ := strconv.ParseFloat(st.CurrentState["bandwidth"], 64)
					rs.UI.Widgets.WaterfallPage.Waterfall.DispLow = center - span/2
					rs.UI.Widgets.WaterfallPage.Waterfall.DispHigh = center + span/2
				}
			}
		case st := <-streams.Updates:
			if st.CurrentState["client_handle"] == rs.ClientID && st.CurrentState["type"] == "remote_audio_rx" && st.CurrentState["compression"] == "OPUS" {
				streamStr := strings.TrimPrefix(st.Object, "stream 0x")
				streamId, err := strconv.ParseUint(streamStr, 16, 32)
				if err != nil {
					log.Println(err)
				} else {
					log.Println("got opus stream", streamStr)
					rs.AudioStream = uint32(streamId)
				}
			}
		case pkt := <-vita:
			if pkt.Preamble.Stream_id == rs.WaterfallStream {
				rs.updateWaterfall(pkt)
			}
			if pkt.Preamble.Stream_id == rs.AudioStream {
				rs.playOpus(pkt)
			}
		}
	}
}

func formatFreq(fFloat float64, err error) string {
	if err != nil {
		return "<error>"
	}
	freq := int(math.Round(fFloat * 1e6))
	fStr := ""
	for freq > 0 {
		mod1000 := freq % 1000
		freq = freq / 1000
		var chunk string
		if freq > 0 {
			chunk = fmt.Sprintf("%03d", mod1000)
		} else {
			chunk = fmt.Sprintf("%d", mod1000)
		}

		if fStr == "" {
			fStr = chunk
		} else {
			fStr = chunk + "." + fStr
		}
	}
	return fStr
}

func (rs *RadioState) updateGUI() {
	slices := map[string]radioshim.SliceData{}
	for objName, slice := range rs.FlexClient.FindObjects("slice ") {
		if slice["client_handle"] != rs.ClientID {
			continue
		}
		letter := slice["index_letter"]
		out := radioshim.SliceData{Present: slice["in_use"] != "0"}
		var err error
		out.Freq, err = strconv.ParseFloat(slice["RF_frequency"], 64)
		out.FreqFormatted = formatFreq(out.Freq, err)
		out.Mode = slice["mode"]
		out.Modes = strings.Split(slice["mode_list"], ",")
		out.RXAnt = slice["rxant"]
		out.TXAnt = slice["txant"]
		out.Active = slice["active"] != "0"
		out.FiltLow, _ = strconv.ParseFloat(slice["filter_lo"], 64)
		out.FiltHigh, _ = strconv.ParseFloat(slice["filter_hi"], 64)
		out.Index, _ = strconv.Atoi(strings.TrimPrefix(objName, "slice "))
		out.TuneStep, _ = strconv.ParseFloat(slice["step"], 64)
		out.TuneStep /= 1e6
		slices[letter] = out
	}
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.Slices = slices
}

func (rs *RadioState) updateWaterfall(pkt flexclient.VitaPacket) {
	data := vita.ParseVitaWaterfall(pkt.Payload, pkt.Preamble)
	if data.TotalBinsInFrame != rs.WFState.width {
		rs.UI.Widgets.WaterfallPage.Waterfall.SetBins(data.TotalBinsInFrame)
		rs.WFState.width = data.TotalBinsInFrame
		rs.WFState.bins = make([]uint16, data.TotalBinsInFrame)
	}

	// TODO: FlexLib does a fancy thing here where it keeps several "in-progress" rows
	// keyed by timecode, and flushes them out as they fill, which I guess can be useful
	// in case of packet reordering.
	if data.Timecode != rs.WFState.timecode {
		rs.WFState.timecode = data.Timecode
		low := data.FrameLowFreq
		// The +1 is very confusing and probably wrong,
		// and yet it seems to produce a correct result.
		high := low + uint64(data.TotalBinsInFrame-1)*(data.BinBandwidth+1)
		rs.WFState.dataLow = float64(low) / 1e6
		rs.WFState.dataHigh = float64(high) / 1e6
		rs.WFState.binsFilled = 0
	}

	copy(rs.WFState.bins[int(data.FirstBinIndex):int(data.FirstBinIndex)+int(data.Width)], data.Data)
	rs.WFState.binsFilled += data.Width

	if rs.WFState.binsFilled == rs.WFState.width {
		rs.UI.Widgets.WaterfallPage.Waterfall.DataLow = rs.WFState.dataLow
		rs.UI.Widgets.WaterfallPage.Waterfall.DataHigh = rs.WFState.dataHigh
		rs.UI.Widgets.WaterfallPage.Waterfall.AddRow(rs.WFState.bins, data.AutoBlackLevel)
	}
}

func (rs *RadioState) playOpus(pkt flexclient.VitaPacket) {
	data := vita.ParseVitaOpus(pkt.Payload, pkt.Preamble)
	rs.Audio.Decode(data)
}

func (rs *RadioState) ToggleAudio(enable bool) {
	if enable {
		rs.FlexClient.SendCmd("stream create type=remote_audio_rx compression=opus")
		rs.Audio.Start()
	} else {
		rs.FlexClient.SendCmd(fmt.Sprintf("stream remove 0x%08x", rs.AudioStream))
		rs.AudioStream = 0
		rs.Audio.Pause()
	}
}

func (rs *RadioState) GetSlices() map[string]radioshim.SliceData {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.Slices
}

func (rs *RadioState) getWaterfallAndPan() (flexclient.Object, flexclient.Object) {
	if rs.WaterfallStream == 0 {
		return nil, nil
	}
	fc := rs.FlexClient
	wf, ok := fc.GetObject(fmt.Sprintf("display waterfall 0x%08X", rs.WaterfallStream))
	if !ok {
		return nil, nil
	}
	panId := wf["panadapter"]
	pan, _ := fc.GetObject(fmt.Sprintf("display pan %s", panId))
	return wf, pan
}

func (rs *RadioState) ZoomIn() {
	wf, pan := rs.getWaterfallAndPan()
	if wf == nil || pan == nil {
		return
	}
	bw, _ := strconv.ParseFloat(pan["bandwidth"], 64)
	minBw, _ := strconv.ParseFloat(pan["min_bw"], 64)
	bw = max(bw/2, minBw)
	rs.FlexClient.PanSet(wf["panadapter"], flexclient.Object{"bandwidth": fmt.Sprintf("%f", bw)})
}

func (rs *RadioState) ZoomOut() {
	wf, pan := rs.getWaterfallAndPan()
	if wf == nil || pan == nil {
		return
	}
	bw, _ := strconv.ParseFloat(pan["bandwidth"], 64)
	maxBw, _ := strconv.ParseFloat(pan["max_bw"], 64)
	bw = min(bw*2, maxBw)
	rs.FlexClient.PanSet(wf["panadapter"], flexclient.Object{"bandwidth": fmt.Sprintf("%f", bw)})
}

func (rs *RadioState) FindActiveSlice() {
	for objName, slice := range rs.FlexClient.FindObjects("slice ") {
		if slice["active"] == "0" {
			continue
		}
		index := strings.TrimPrefix(objName, "slice ")
		freq := slice["RF_frequency"]
		rs.FlexClient.SendCmd(fmt.Sprintf("slice tune %s %s autopan=1", index, freq))
	}
}

func (rs *RadioState) ActivateSlice(index int) {
	rs.FlexClient.SliceSet(fmt.Sprintf("%d", index), flexclient.Object{"active": "1"})
}

func (rs *RadioState) TuneSlice(data radioshim.SliceData, freq float64, snap bool) {
	if snap {
		freq = math.Round(freq/data.TuneStep) * data.TuneStep
	}
	rs.FlexClient.SliceTune(fmt.Sprintf("%d", data.Index), freq)
}

func (rs *RadioState) TuneSliceStep(data radioshim.SliceData, steps int) {
	newFreq := data.Freq + float64(steps)*data.TuneStep
	rs.FlexClient.SliceTune(fmt.Sprintf("%d", data.Index), newFreq)
}

func (rs *RadioState) SetSliceMode(index int, mode string) {
	rs.FlexClient.SliceSet(fmt.Sprintf("%d", index), flexclient.Object{"mode": mode})
}

func (rs *RadioState) CenterWaterfallAt(freq float64) {
	wf, pan := rs.getWaterfallAndPan()
	if wf == nil || pan == nil {
		return
	}
	rs.FlexClient.PanSet(wf["panadapter"], flexclient.Object{"center": fmt.Sprintf("%f", freq)})
}
