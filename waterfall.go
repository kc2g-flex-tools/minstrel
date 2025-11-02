// Waterfall VITA packet processing and display control
// Extracted from flexclient.go on 2025-11-02

package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hb9fxq/flexlib-go/vita"
	"github.com/kc2g-flex-tools/flexclient"

	"github.com/kc2g-flex-tools/minstrel/events"
)

func (rs *RadioState) updateWaterfall(pkt flexclient.VitaPacket) {
	data := vita.ParseVitaWaterfall(pkt.Payload, pkt.Preamble)
	if data.TotalBinsInFrame != rs.WFState.width {
		rs.EventBus.Publish(events.WaterfallBinsConfigured{
			Width: data.TotalBinsInFrame,
		})
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
		rs.EventBus.Publish(events.WaterfallDataRangeChanged{
			Low:  rs.WFState.dataLow,
			High: rs.WFState.dataHigh,
		})
		rs.EventBus.Publish(events.WaterfallRowReceived{
			Bins:       rs.WFState.bins,
			BlackLevel: data.AutoBlackLevel,
		})
	}
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
	_, err := rs.FlexClient.PanSet(context.Background(), wf["panadapter"], flexclient.Object{"bandwidth": fmt.Sprintf("%f", bw)})
	if err != nil {
		log.Println("PanSet error:", err)
	}
}

func (rs *RadioState) ZoomOut() {
	wf, pan := rs.getWaterfallAndPan()
	if wf == nil || pan == nil {
		return
	}
	bw, _ := strconv.ParseFloat(pan["bandwidth"], 64)
	maxBw, _ := strconv.ParseFloat(pan["max_bw"], 64)
	bw = min(bw*2, maxBw)
	_, err := rs.FlexClient.PanSet(context.Background(), wf["panadapter"], flexclient.Object{"bandwidth": fmt.Sprintf("%f", bw)})
	if err != nil {
		log.Println("PanSet error:", err)
	}
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

func (rs *RadioState) CenterWaterfallAt(freq float64) {
	wf, pan := rs.getWaterfallAndPan()
	if wf == nil || pan == nil {
		return
	}
	_, err := rs.FlexClient.PanSet(context.Background(), wf["panadapter"], flexclient.Object{"center": fmt.Sprintf("%f", freq)})
	if err != nil {
		log.Println("PanSet error:", err)
	}
}
