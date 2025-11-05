// RadioState lifecycle management and main event loop
// Extracted from flexclient.go on 2025-11-02

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/kc2g-flex-tools/flexclient"

	"github.com/kc2g-flex-tools/minstrel/audio"
	"github.com/kc2g-flex-tools/minstrel/events"
	"github.com/kc2g-flex-tools/minstrel/pkg/errutil"
	"github.com/kc2g-flex-tools/minstrel/radioshim"
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
	stationName     string
	profileName     string
	discoveryCancel context.CancelFunc
}

func NewRadioState(audioCtx *audio.Audio, eventBus *events.Bus, station, profile string) *RadioState {
	rs := &RadioState{
		Audio:       audioCtx,
		EventBus:    eventBus,
		stationName: station,
		profileName: profile,
	}
	return rs
}

// StartDiscovery begins discovering radios on the network
func (rs *RadioState) StartDiscovery(ctx context.Context) {
	discoveryCtx, cancel := context.WithCancel(ctx)
	rs.discoveryCancel = cancel

	discoverChan := make(chan []map[string]string, 1)
	go func() {
		log.Println("start discovery")
		err := flexclient.DiscoverAll(discoveryCtx, 10*time.Second, discoverChan)
		log.Println("finished discovery")
		if err != nil {
			log.Fatal(err)
		}
	}()
	go func() {
		for data := range discoverChan {
			rs.EventBus.Publish(events.RadiosDiscovered{
				Radios: data,
			})
		}
	}()
}

// ConnectToRadio establishes connection to a radio at the given address
func (rs *RadioState) ConnectToRadio(ctx context.Context, address string) error {
	// Cancel discovery if running
	if rs.discoveryCancel != nil {
		rs.discoveryCancel()
	}

	fc, err := flexclient.NewFlexClient(address)
	if err != nil {
		return err
	}

	rs.FlexClient = fc

	go func() {
		fc.Run()
		rs.EventBus.Publish(events.RadioDisconnected{
			Error: "flexclient exited",
		})
		log.Fatal("flexclient exited")
	}()

	rs.EventBus.Publish(events.RadioConnected{})
	go rs.Run(ctx)

	return nil
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
	transmit := fc.Subscribe(flexclient.Subscription{
		Prefix:  "transmit",
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
	fc.SendAndWait("client station " + strings.ReplaceAll(rs.stationName, " ", "\x7f"))

	if rs.profileName != "" {
		fc.SendAndWait("profile global load " + rs.profileName)
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
				streamId := errutil.MustParseUint32(streamStr, 16, "waterfall stream ID")
				if streamId != 0 {
					if rs.WaterfallStream == 0 {
						log.Println("my waterfall is", streamStr)
						rs.WaterfallStream = streamId
						wf, _ := rs.getWaterfallAndPan()
						_, err := fc.PanSet(context.Background(), wf["panadapter"], flexclient.Object{"xpixels": "1000"})
						if err != nil {
							log.Println("PanSet error:", err)
						}
					}
					center := errutil.MustParseFloat(st.CurrentState["center"], "waterfall center")
					span := errutil.MustParseFloat(st.CurrentState["bandwidth"], "waterfall bandwidth")
					rs.EventBus.Publish(events.WaterfallDisplayRangeChanged{
						Low:  center - span/2,
						High: center + span/2,
					})
				}
			}
		case st := <-streams.Updates:
			if st.CurrentState["client_handle"] == rs.ClientID && st.CurrentState["type"] == "remote_audio_rx" && st.CurrentState["compression"] == "OPUS" {
				streamStr := strings.TrimPrefix(st.Object, "stream 0x")
				streamId := errutil.MustParseUint32(streamStr, 16, "RX audio stream ID")
				if streamId != 0 {
					log.Println("got opus RX stream", streamStr)
					rs.RXAudioStream = streamId
				}
			}
			if st.CurrentState["client_handle"] == rs.ClientID && st.CurrentState["type"] == "remote_audio_tx" && st.CurrentState["compression"] == "OPUS" {
				streamStr := strings.TrimPrefix(st.Object, "stream 0x")
				streamId := errutil.MustParseUint32(streamStr, 16, "TX audio stream ID")
				if streamId != 0 {
					log.Println("got opus TX stream", streamStr)
					rs.TXAudioStream = streamId
				}
			}
		case st := <-interlock.Updates:
			tx := st.CurrentState["state"] == "TRANSMITTING"
			rs.EventBus.Publish(events.TransmitStateChanged{
				Transmitting: tx,
			})
		case st := <-transmit.Updates:
			if voxEnable, ok := st.CurrentState["vox_enable"]; ok {
				rs.EventBus.Publish(events.VOXStateChanged{
					Enabled: voxEnable == "1",
				})
			}
			// Publish all transmit parameters for settings window
			rs.EventBus.Publish(events.TransmitParamsChanged{
				Params: st.CurrentState,
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
