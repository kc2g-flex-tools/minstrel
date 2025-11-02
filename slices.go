// Slice state extraction and control operations
// Extracted from flexclient.go on 2025-11-02

package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/kc2g-flex-tools/flexclient"

	"github.com/kc2g-flex-tools/minstrel/events"
	"github.com/kc2g-flex-tools/minstrel/radioshim"
)

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
	slices := map[string]*radioshim.SliceData{}
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
		out.Volume, _ = strconv.Atoi(slice["audio_level"])
		slices[letter] = &out
	}
	rs.mu.Lock()
	rs.Slices = slices
	rs.mu.Unlock()

	// Publish event with slice data
	rs.EventBus.Publish(events.SlicesUpdated{
		Slices: slices,
	})
}

func (rs *RadioState) SetSliceVolume(index int, volume int) {
	volStr := strconv.Itoa(volume)
	_, err := rs.FlexClient.SliceSet(context.Background(), fmt.Sprintf("%d", index), flexclient.Object{"audio_level": volStr})
	if err != nil {
		log.Println("SliceSet error:", err)
	}
}

func (rs *RadioState) ActivateSlice(index int) {
	_, err := rs.FlexClient.SliceSet(context.Background(), fmt.Sprintf("%d", index), flexclient.Object{"active": "1"})
	if err != nil {
		log.Println("SliceSet error:", err)
	}
}

func (rs *RadioState) TuneSlice(data *radioshim.SliceData, freq float64, snap bool) {
	if snap {
		freq = math.Round(freq/data.TuneStep) * data.TuneStep
	}
	_, err := rs.FlexClient.SliceTune(context.Background(), fmt.Sprintf("%d", data.Index), freq)
	if err != nil {
		log.Println("SliceTune error:", err)
	}
	data.Freq = freq
}

func (rs *RadioState) TuneSliceStep(data *radioshim.SliceData, steps int) {
	newFreq := data.Freq + float64(steps)*data.TuneStep
	_, err := rs.FlexClient.SliceTune(context.Background(), fmt.Sprintf("%d", data.Index), newFreq)
	if err != nil {
		log.Println("SliceTune error:", err)
	}
}

func (rs *RadioState) SetSliceMode(index int, mode string) {
	_, err := rs.FlexClient.SliceSet(context.Background(), fmt.Sprintf("%d", index), flexclient.Object{"mode": mode})
	if err != nil {
		log.Println("SliceSet error:", err)
	}
}
