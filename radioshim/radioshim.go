package radioshim

type Shim interface {
	ToggleAudio(bool)
	ZoomIn()
	ZoomOut()
	FindActiveSlice()
	GetSlices() map[string]*SliceData
	TuneSlice(*SliceData, float64, bool)
	SetSliceMode(int, string)
	SetSliceRXAnt(int, string)
	SetSliceTXAnt(int, string)
	CenterWaterfallAt(float64)
	ActivateSlice(int)
	TuneSliceStep(*SliceData, int)
	SetSliceVolume(index int, volume int)
	RemoveSlice(int)
	SetPTT(bool)
	SetVOX(bool)
	SetTransmitParam(key string, value int)
	SetAMCarrierLevel(level int)
	SetMicLevel(level int)
	GetMicList(callback func([]string))
	SetMicInput(micName string)
}

type SliceData struct {
	Present       bool
	Active        bool
	Index         int
	Freq          float64
	FreqFormatted string
	Mode          string
	Modes         []string
	RXAnt         string
	TXAnt         string
	RXAntList     []string
	TXAntList     []string
	FiltHigh      float64
	FiltLow       float64
	TuneStep      float64
	Volume        int
}
