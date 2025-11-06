// Waterfall VITA packet processing and display control
// Extracted from flexclient.go on 2025-11-02

package radio

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hb9fxq/flexlib-go/vita"
	"github.com/kc2g-flex-tools/flexclient"

	"github.com/kc2g-flex-tools/minstrel/errutil"
	"github.com/kc2g-flex-tools/minstrel/events"
)

func (rs *RadioState) updateWaterfall(pkt flexclient.VitaPacket) {
	data := vita.ParseVitaWaterfall(pkt.Payload, pkt.Preamble)
	if data.TotalBinsInFrame != rs.wfState.width {
		rs.EventBus.Publish(events.WaterfallBinsConfigured{
			Width: data.TotalBinsInFrame,
		})
		rs.wfState.width = data.TotalBinsInFrame
		rs.wfState.bins = make([]uint16, data.TotalBinsInFrame)
	}

	// TODO: FlexLib does a fancy thing here where it keeps several "in-progress" rows
	// keyed by timecode, and flushes them out as they fill, which I guess can be useful
	// in case of packet reordering.
	if data.Timecode != rs.wfState.timecode {
		rs.wfState.timecode = data.Timecode
		low := data.FrameLowFreq
		// The +1 is very confusing and probably wrong,
		// and yet it seems to produce a correct result.
		high := low + uint64(data.TotalBinsInFrame-1)*(data.BinBandwidth+1)
		rs.wfState.dataLow = float64(low) / 1e6
		rs.wfState.dataHigh = float64(high) / 1e6
		rs.wfState.binsFilled = 0
	}

	copy(rs.wfState.bins[int(data.FirstBinIndex):int(data.FirstBinIndex)+int(data.Width)], data.Data)
	rs.wfState.binsFilled += data.Width

	if rs.wfState.binsFilled == rs.wfState.width {
		rs.EventBus.Publish(events.WaterfallDataRangeChanged{
			Low:  rs.wfState.dataLow,
			High: rs.wfState.dataHigh,
		})
		rs.EventBus.Publish(events.WaterfallRowReceived{
			Bins:       rs.wfState.bins,
			BlackLevel: data.AutoBlackLevel,
		})
	}
}

func (rs *RadioState) getWaterfallAndPan() (flexclient.Object, flexclient.Object) {
	if !rs.WaterfallStream.IsValid() {
		return nil, nil
	}
	fc := rs.FlexClient
	wf, ok := fc.GetObject(fmt.Sprintf("display waterfall %s", rs.WaterfallStream))
	if !ok {
		return nil, nil
	}
	panId := wf["panadapter"]
	pan, _ := fc.GetObject(fmt.Sprintf("display pan %s", panId))
	return wf, pan
}

// setPanParameter sets a panadapter parameter using PanSet
func (rs *RadioState) setPanParameter(key, value string) error {
	wf, pan := rs.getWaterfallAndPan()
	if wf == nil || pan == nil {
		return fmt.Errorf("waterfall or pan not available")
	}
	_, err := rs.FlexClient.PanSet(context.Background(), wf["panadapter"], flexclient.Object{key: value})
	if err != nil {
		log.Printf("PanSet %s error: %v", key, err)
	}
	return err
}

func (rs *RadioState) ZoomIn() {
	_, pan := rs.getWaterfallAndPan()
	if pan == nil {
		return
	}
	bw := errutil.MustParseFloat(pan["bandwidth"], "pan bandwidth")
	minBw := errutil.MustParseFloat(pan["min_bw"], "pan min_bw")
	bw = max(bw/2, minBw)
	rs.setPanParameter("bandwidth", fmt.Sprintf("%f", bw))
}

func (rs *RadioState) ZoomOut() {
	_, pan := rs.getWaterfallAndPan()
	if pan == nil {
		return
	}
	bw := errutil.MustParseFloat(pan["bandwidth"], "pan bandwidth")
	maxBw := errutil.MustParseFloat(pan["max_bw"], "pan max_bw")
	bw = min(bw*2, maxBw)
	rs.setPanParameter("bandwidth", fmt.Sprintf("%f", bw))
}

func (rs *RadioState) FindActiveSlice() {
	for objName, slice := range rs.FlexClient.FindObjects("slice ") {
		if slice["active"] == "0" {
			continue
		}
		index := strings.TrimPrefix(objName, "slice ")
		freq := errutil.MustParseFloat(slice["RF_frequency"], "slice RF_frequency")
		rs.FlexClient.SliceTuneOpts(context.Background(), index, freq, flexclient.Object{"autopan": "1"})
	}
}

func (rs *RadioState) CenterWaterfallAt(freq float64) {
	rs.setPanParameter("center", fmt.Sprintf("%f", freq))
}
