package persistence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/adrg/xdg"
)

// MIDISettings contains persistent MIDI configuration
type MIDISettings struct {
	Port            string `json:"port"`
	VFOControl      byte   `json:"vfo_control,omitempty"`
	VolControl      byte   `json:"vol_control,omitempty"`
	LeftPaddleNote  byte   `json:"left_paddle_note,omitempty"`
	RightPaddleNote byte   `json:"right_paddle_note,omitempty"`
}

// Settings contains all persistent application settings
type Settings struct {
	ClientID string       `json:"client_id"`
	MIDI     MIDISettings `json:"midi"`
}

// SettingsStore handles persistent storage of application settings
type SettingsStore struct {
	filepath string
}

// NewSettingsStore creates a new SettingsStore instance
func NewSettingsStore() (*SettingsStore, error) {
	filepath, err := xdg.DataFile("minstrel/settings.json")
	if err != nil {
		return nil, fmt.Errorf("failed to get data file path: %w", err)
	}
	return &SettingsStore{filepath: filepath}, nil
}

// Load retrieves the stored settings
func (ss *SettingsStore) Load() (*Settings, error) {
	file, err := os.Open(ss.filepath)
	if err != nil {
		if os.IsNotExist(err) {
			// Settings file doesn't exist - check for legacy client store
			settings, migrated, migrateErr := ss.migrateFromLegacy()
			if migrateErr != nil {
				// Migration failed, return empty settings
				return &Settings{}, nil
			}
			if migrated {
				// Successfully migrated - save and return
				if saveErr := ss.Save(settings); saveErr != nil {
					return settings, fmt.Errorf("migrated settings but failed to save: %w", saveErr)
				}
			}
			return settings, nil
		}
		return nil, err
	}
	defer file.Close()

	var settings Settings
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&settings); err != nil {
		return nil, fmt.Errorf("failed to decode settings: %w", err)
	}

	return &settings, nil
}

// migrateFromLegacy attempts to migrate from the old client_id file
func (ss *SettingsStore) migrateFromLegacy() (*Settings, bool, error) {
	legacyPath, err := xdg.DataFile("minstrel/client_id")
	if err != nil {
		return &Settings{}, false, fmt.Errorf("failed to get legacy path: %w", err)
	}

	// Check if legacy file exists
	legacyFile, err := os.Open(legacyPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No legacy file - not an error, just return empty settings
			return &Settings{}, false, nil
		}
		return &Settings{}, false, fmt.Errorf("failed to open legacy file: %w", err)
	}
	defer legacyFile.Close()

	// Read legacy client ID
	contents, err := os.ReadFile(legacyPath)
	if err != nil {
		return &Settings{}, false, fmt.Errorf("failed to read legacy file: %w", err)
	}

	clientID := string(contents)
	// Trim whitespace and newlines
	clientID = string(bytes.TrimSpace(contents))

	if clientID == "" {
		// Empty legacy file - just delete it
		os.Remove(legacyPath)
		return &Settings{}, false, nil
	}

	// Create settings with migrated client ID
	settings := &Settings{
		ClientID: clientID,
	}

	// Delete the legacy file
	if err := os.Remove(legacyPath); err != nil {
		// Log but don't fail - we got the data
		fmt.Printf("Warning: migrated client ID but failed to delete legacy file: %v\n", err)
	} else {
		fmt.Printf("Migrated client ID from legacy storage: %s\n", clientID)
	}

	return settings, true, nil
}

// Save stores the settings to disk
func (ss *SettingsStore) Save(settings *Settings) error {
	file, err := os.Create(ss.filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(settings); err != nil {
		return fmt.Errorf("failed to encode settings: %w", err)
	}

	return nil
}

// Compatibility functions for legacy client ID storage
// These can be removed once all users have migrated

// NewClientStore creates a new SettingsStore (legacy name)
func NewClientStore() (*SettingsStore, error) {
	return NewSettingsStore()
}

// LoadClientID retrieves just the client ID (legacy function)
func (ss *SettingsStore) LoadClientID() (string, error) {
	settings, err := ss.Load()
	if err != nil {
		return "", err
	}
	return settings.ClientID, nil
}

// SaveClientID stores just the client ID (legacy function)
func (ss *SettingsStore) SaveClientID(id string) error {
	settings, err := ss.Load()
	if err != nil {
		settings = &Settings{}
	}
	settings.ClientID = id
	return ss.Save(settings)
}
