package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/hb9fxq/flexlib-go/vita"
	"github.com/kc2g-flex-tools/flexclient"
	"github.com/kc2g-flex-tools/minstrel/audio"
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
	FlexClient      *flexclient.FlexClient
	UI              *ui.UI
	Audio           *audio.Audio
	ClientID        string
	WaterfallStream uint32
	AudioStream     uint32
	WFState         WFState
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

func (rs *RadioState) Run() {
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
					log.Println("my waterfall is", streamStr)
					rs.WaterfallStream = uint32(streamId)
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
	slices := map[string]ui.SliceData{}
	for _, slice := range rs.FlexClient.FindObjects("slice ") {
		if slice["client_handle"] != rs.ClientID {
			continue
		}
		letter := slice["index_letter"]
		out := ui.SliceData{Present: slice["in_use"] != "0"}
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
		slices[letter] = out
	}
	rs.UI.Widgets.WaterfallPage.SetSlices(slices)
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
		rs.WFState.dataLow = float64(data.FrameLowFreq) / 1e6
		// The +1 is very confusing and probably wrong,
		// and yet it seems to produce a correct result.
		rs.WFState.dataHigh = float64(data.FrameLowFreq+uint64(data.TotalBinsInFrame)*(data.BinBandwidth+1)) / 1e6
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

func (rs *RadioState) ToggleAudio() {
	if rs.AudioStream == 0 {
		rs.FlexClient.SendCmd("stream create type=remote_audio_rx compression=opus")
		rs.Audio.Player.Play()
	} else {
		rs.FlexClient.SendCmd(fmt.Sprintf("stream remove 0x%08x", rs.AudioStream))
		rs.AudioStream = 0
		rs.Audio.Player.Pause()
	}
}
