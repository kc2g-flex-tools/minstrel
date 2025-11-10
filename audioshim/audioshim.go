package audioshim

// Shim is an interface that abstracts audio operations for the UI layer
type Shim interface {
	GetAudioSinks(callback func([]AudioDevice))
	GetAudioSources(callback func([]AudioDevice))
	GetDefaultAudioSink() string
	GetDefaultAudioSource() string
	SetAudioSink(deviceID string)
	SetAudioSource(deviceID string)
}

// AudioDevice represents an audio input or output device
type AudioDevice struct {
	ID   string
	Name string
}
