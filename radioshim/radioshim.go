package radioshim

type Shim interface {
	ToggleAudio(bool)
	ZoomIn()
	ZoomOut()
	FindActiveSlice()
	GetSlices() map[string]SliceData
	TuneSlice(int, float64)
	SetSliceMode(int, string)
	CenterWaterfallAt(float64)
	ActivateSlice(int)
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
}
