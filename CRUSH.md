# Minstrel Development Guide

## Build Commands
```bash
go build                    # Build the main binary
go build -o minstrel-amd64  # Build for amd64
go build -o minstrel-arm64  # Build for arm64
go mod tidy                 # Clean up dependencies
go mod download             # Download dependencies
```

## Code Style Guidelines

### Import Organization
Group imports in this order with blank lines between:
1. Standard library packages
2. Third-party dependencies  
3. Internal project packages (github.com/kc2g-flex-tools/minstrel/...)

### Naming Conventions
- **Packages**: lowercase single words (audio, midi, ui)
- **Exported types/functions**: CamelCase (RadioState, NewUI)
- **Unexported types/functions**: camelCase (updateGUI, clientID)
- **Interfaces**: CamelCase, often ending in -er (Shim)

### Error Handling
- Use `panic()` for initialization/fatal errors
- Use `log.Fatal()` for runtime critical errors
- Use `log.Println()` for non-critical errors
- Minimal error wrapping - keep it simple

### Code Structure
- Event-driven architecture using Ebiten game engine
- Heavy use of goroutines and channels for concurrency
- Use sync.RWMutex for shared state protection
- Keep packages focused with clear boundaries

## Package Structure

### Main Package (root directory)

**Core Application File:**
- **`main.go`** - Application entry point, configuration, and initialization

### Core Packages

#### `radio/`
Core radio control and state management.
- **`state.go`** - RadioState lifecycle management and main event loop. Handles FlexClient connection, discovery, and coordinates all radio interactions
- **`slices.go`** - Slice state extraction and control operations (tuning, mode changes, antenna selection)
- **`waterfall.go`** - Waterfall VITA packet processing and display control (pan/zoom operations)
- **`streams.go`** - Audio stream lifecycle management (RX/TX stream creation/removal, PTT, VOX)

#### `audio/`
Audio processing, playback, and recording for both RX and TX.
- **`audio.go`** - PulseAudio integration, Opus decoding/encoding, circular buffering for RX audio, VITA packet generation for TX audio
- **`circular_buffer.go`** - Lock-free circular buffer implementation for audio samples

#### `events/`
Event bus system for decoupled communication between components.
- **`events.go`** - Event types and pub/sub bus implementation. Events include: waterfall updates, slice changes, transmit state, radio discovery

#### `radioshim/`
Interface abstraction layer between UI and radio control.
- **`radioshim.go`** - `Shim` interface defining all radio operations. Allows UI to remain independent of RadioState implementation. Includes `SliceData` type and `SliceMap` type alias

#### `midi/`
MIDI controller support for hardware control.
- **`midi.go`** - MIDI input handling for VFO knobs, volume control, and CW paddles

#### `ui/`
All user interface components built with Ebiten and EbitenUI.
- **`ui.go`** - Main UI struct, event loop integration, deferred execution pattern for thread-safe UI updates
- **`waterfall.go`** - Waterfall display rendering with GPU acceleration
- **`waterfall_slice.go`** - Slice indicators and controls overlaid on waterfall
- **`waterfall_controls.go`** - Control panel below waterfall (audio toggle, MOX, VOX, etc.)
- **`radios.go`** - Radio discovery and selection page
- **`transmit_settings.go`** - TX parameter window (RF power, mic gain, etc.)
- **`fonts.go`** - Font loading from embedded assets
- **`widgets.go`** - Custom widget helpers (buttons, text, rounded rectangles)
- **`window.go`** - Modal window system
- **`gradient.go`** - Color gradient utilities for waterfall display

### Utility Packages

#### `errutil/`
Error handling utilities using panic-based approach.
- **`errutil.go`** - `MustParse*` functions for string-to-number conversions, `LogError`, `FatalError`

#### `format/`
Formatting utilities for display.
- **`frequency.go`** - `FrequencyMHz()` formats frequencies with dot separators (e.g., "14.250.000")

#### `persistence/`
Persistent storage management.
- **`client.go`** - `ClientStore` for FlexRadio client UUID persistence using XDG data directories

#### `types/`
Radio-specific type definitions.
- **`streamid.go`** - `StreamID` type for type-safe VITA stream identification with validation and formatting

### Third-Party Wrappers

#### `assets/`
Embedded asset management.
- **`assets.go`** - Embeds fonts directory using `go:embed`

#### `opus/`
Opus codec CGO wrapper for TX audio encoding.
- **`opus.go`** - CGO bindings to libopus for encoder control (bitrate, complexity, VBR)