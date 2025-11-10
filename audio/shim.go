package audio

import (
	"fmt"
	"log"
	"os"

	"github.com/jfreymuth/pulse/proto"
	"github.com/kc2g-flex-tools/minstrel/audioshim"
)

// GetAudioSinks returns a list of available audio output devices
func (a *Audio) GetAudioSinks(callback func([]audioshim.AudioDevice)) {
	go func() {
		sinks, err := a.Context.ListSinks()
		devices := []audioshim.AudioDevice{}
		if err != nil {
			log.Printf("Failed to list audio sinks: %v", err)
			callback(devices)
			return
		}
		for _, sink := range sinks {
			devices = append(devices, audioshim.AudioDevice{
				ID:   sink.ID(),
				Name: sink.Name(),
			})
		}
		callback(devices)
	}()
}

// GetDefaultAudioSink returns the ID of the default audio output device
func (a *Audio) GetDefaultAudioSink() string {
	sink, err := a.Context.DefaultSink()
	if err != nil {
		log.Printf("Failed to get default sink: %v", err)
		return ""
	}
	return sink.ID()
}

// GetAudioSources returns a list of available audio input devices
func (a *Audio) GetAudioSources(callback func([]audioshim.AudioDevice)) {
	go func() {
		sources, err := a.Context.ListSources()
		devices := []audioshim.AudioDevice{}
		if err != nil {
			log.Printf("Failed to list audio sources: %v", err)
			callback(devices)
			return
		}
		for _, source := range sources {
			devices = append(devices, audioshim.AudioDevice{
				ID:   source.ID(),
				Name: source.Name(),
			})
		}
		callback(devices)
	}()
}

// GetDefaultAudioSource returns the ID of the default audio input device
func (a *Audio) GetDefaultAudioSource() string {
	source, err := a.Context.DefaultSource()
	if err != nil {
		log.Printf("Failed to get default source: %v", err)
		return ""
	}
	return source.ID()
}

// SetAudioSink sets the output device and moves the stream if active
// deviceID can be empty string to use system default
func (a *Audio) SetAudioSink(deviceID string) {
	// Check if device is already set to this value
	a.deviceMutex.RLock()
	currentDevice := a.sinkDevice
	a.deviceMutex.RUnlock()

	if currentDevice == deviceID {
		// Already using this device, nothing to do
		return
	}

	// Store the selected device (empty string means default)
	a.SetSinkDevice(deviceID)

	// Check if player is active
	if a.player == nil {
		// No active player, setting will be used on next Start()
		return
	}

	// Move the existing stream to the new device instead of recreating it
	streamInputIndex := a.player.StreamInputIndex()

	// Use RawRequest to send MoveSinkInput command
	err := a.Context.RawRequest(&proto.MoveSinkInput{
		SinkInputIndex: streamInputIndex,
		DeviceIndex:    proto.Undefined,
		DeviceName:     deviceID, // Empty string means default
	}, nil)

	if err != nil {
		log.Printf("Failed to move sink input to device %s: %v", deviceID, err)
	}
}

// SetAudioSource sets the input device and moves the stream if active
func (a *Audio) SetAudioSource(deviceID string) {
	// Check if device is already set to this value
	a.deviceMutex.RLock()
	currentDevice := a.sourceDevice
	a.deviceMutex.RUnlock()

	if currentDevice == deviceID {
		// Already using this device, nothing to do
		return
	}

	// Store the selected device
	a.SetSourceDevice(deviceID)

	// Check if recording is active
	a.txMutex.Lock()
	wasRunning := a.txRunning
	a.txMutex.Unlock()

	if !wasRunning {
		return
	}

	// Find our source output by looking for our application
	// RecordStream doesn't expose its index, so we query the server
	sourceOutputIndex, err := a.findOurSourceOutput()
	if err != nil {
		log.Printf("Failed to find source output: %v", err)
		return
	}

	// Move the existing stream to the new device
	err = a.Context.RawRequest(&proto.MoveSourceOutput{
		SourceOutputIndex: sourceOutputIndex,
		DeviceIndex:       proto.Undefined,
		DeviceName:        deviceID, // Empty string means default
	}, nil)

	if err != nil {
		log.Printf("Failed to move source output to device %s: %v", deviceID, err)
	}
}

// findOurSourceOutput finds the source output index for our recording stream
// by querying PulseAudio for all source outputs and finding the one with our PID
func (a *Audio) findOurSourceOutput() (uint32, error) {
	var reply proto.GetSourceOutputInfoListReply
	err := a.Context.RawRequest(&proto.GetSourceOutputInfoList{}, &reply)
	if err != nil {
		return 0, fmt.Errorf("failed to get source output list: %w", err)
	}

	// Get our process ID
	myPID := fmt.Sprintf("%d", os.Getpid())

	// Find the source output that belongs to our process
	for _, info := range reply {
		if info.Properties != nil {
			if pid, ok := info.Properties["application.process.id"]; ok && pid.String() == myPID {
				return info.SourceOutpuIndex, nil
			}
		}
	}

	return 0, fmt.Errorf("could not find source output for our process (PID %s)", myPID)
}
