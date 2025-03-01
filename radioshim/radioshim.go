package radioshim

type Shim interface {
	ToggleAudio(bool)
	ZoomIn()
	ZoomOut()
	FindActiveSlice()
	GetSlices() map[string]SliceData
	TuneSlice(SliceData, float64, bool)
	SetSliceMode(int, string)
	CenterWaterfallAt(float64)
	ActivateSlice(int)
	TuneSliceStep(SliceData, int)
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
	FiltHigh      float64
	FiltLow       float64
	TuneStep      float64
}
