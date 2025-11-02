// RadioState lifecycle management and main event loop
// Extracted from flexclient.go on 2025-11-02

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/kc2g-flex-tools/flexclient"

	"github.com/kc2g-flex-tools/minstrel/audio"
	"github.com/kc2g-flex-tools/minstrel/events"
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
	Audio           *audio.Audio
	EventBus        *events.Bus
	ClientID        string
	WaterfallStream uint32
	RXAudioStream   uint32
	TXAudioStream   uint32
	WFState         WFState
	Slices          map[string]*radioshim.SliceData
}

func NewRadioState(fc *flexclient.FlexClient, u *ui.UI, audioCtx *audio.Audio, eventBus *events.Bus) *RadioState {
	rs := &RadioState{
		FlexClient: fc,
		Audio:      audioCtx,
		EventBus:   eventBus,
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
	interlock := fc.Subscribe(flexclient.Subscription{
		Prefix:  "interlock",
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
	fc.SendAndWait("sub tx all")

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
						_, err := fc.PanSet(context.Background(), wf["panadapter"], flexclient.Object{"xpixels": "1000"})
						if err != nil {
							log.Println("PanSet error:", err)
						}
					}
					center, _ := strconv.ParseFloat(st.CurrentState["center"], 64)
					span, _ := strconv.ParseFloat(st.CurrentState["bandwidth"], 64)
					rs.EventBus.Publish(events.WaterfallDisplayRangeChanged{
						Low:  center - span/2,
						High: center + span/2,
					})
				}
			}
		case st := <-streams.Updates:
			if st.CurrentState["client_handle"] == rs.ClientID && st.CurrentState["type"] == "remote_audio_rx" && st.CurrentState["compression"] == "OPUS" {
				streamStr := strings.TrimPrefix(st.Object, "stream 0x")
				streamId, err := strconv.ParseUint(streamStr, 16, 32)
				if err != nil {
					log.Println(err)
				} else {
					log.Println("got opus RX stream", streamStr)
					rs.RXAudioStream = uint32(streamId)
				}
			}
			if st.CurrentState["client_handle"] == rs.ClientID && st.CurrentState["type"] == "remote_audio_tx" && st.CurrentState["compression"] == "OPUS" {
				streamStr := strings.TrimPrefix(st.Object, "stream 0x")
				streamId, err := strconv.ParseUint(streamStr, 16, 32)
				if err != nil {
					log.Println(err)
				} else {
					log.Println("got opus TX stream", streamStr)
					rs.TXAudioStream = uint32(streamId)
				}
			}
		case st := <-interlock.Updates:
			tx := st.CurrentState["state"] == "TRANSMITTING"
			rs.EventBus.Publish(events.TransmitStateChanged{
				Transmitting: tx,
			})
		case pkt := <-vita:
			if pkt.Preamble.Stream_id == rs.WaterfallStream {
				rs.updateWaterfall(pkt)
			}
			if pkt.Preamble.Stream_id == rs.RXAudioStream {
				rs.playOpus(pkt)
			}
		}
	}
}

func (rs *RadioState) GetSlices() map[string]*radioshim.SliceData {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.Slices
}
