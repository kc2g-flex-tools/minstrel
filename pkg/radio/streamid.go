package radio

import (
	"fmt"
	"strconv"
	"strings"
)

// StreamID represents a FlexRadio VITA stream identifier
type StreamID uint32

// String returns the stream ID formatted as a hex string
func (s StreamID) String() string {
	return fmt.Sprintf("0x%08X", uint32(s))
}

// StringLower returns the stream ID formatted as a lowercase hex string
func (s StreamID) StringLower() string {
	return fmt.Sprintf("0x%08x", uint32(s))
}

// IsValid returns true if the stream ID is non-zero
func (s StreamID) IsValid() bool {
	return s != 0
}

// ParseStreamID parses a hex string into a StreamID
func ParseStreamID(s string) (StreamID, error) {
	s = strings.TrimPrefix(s, "stream ")
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	id, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, err
	}
	return StreamID(id), nil
}

// MustParseStreamID parses a hex string into a StreamID and panics on error
func MustParseStreamID(s, context string) StreamID {
	id, err := ParseStreamID(s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse stream ID %q for %s: %v", s, context, err))
	}
	return id
}
