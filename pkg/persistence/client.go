package persistence

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/adrg/xdg"
)

// ClientStore handles persistent storage of the FlexRadio client ID
type ClientStore struct {
	filepath string
}

// NewClientStore creates a new ClientStore instance
func NewClientStore() (*ClientStore, error) {
	filepath, err := xdg.DataFile("minstrel/client_id")
	if err != nil {
		return nil, fmt.Errorf("failed to get data file path: %w", err)
	}
	return &ClientStore{filepath: filepath}, nil
}

// Load retrieves the stored client ID
func (cs *ClientStore) Load() (string, error) {
	file, err := os.Open(cs.filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	contents, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(contents)), nil
}

// Save stores the client ID to disk
func (cs *ClientStore) Save(id string) error {
	file, err := os.Create(cs.filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "%s\n", id)
	return err
}
