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