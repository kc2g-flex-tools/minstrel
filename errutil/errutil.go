package errutil

import (
	"log"
	"strconv"
)

// MustParseFloat parses a float64 or logs error and returns 0.
// The context parameter provides information about where the parse occurred.
func MustParseFloat(s string, context string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Printf("ParseFloat error in %s: %v (input: %q)", context, err, s)
		return 0
	}
	return f
}

// MustParseInt parses an int or logs error and returns 0.
// The context parameter provides information about where the parse occurred.
func MustParseInt(s string, context string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("ParseInt error in %s: %v (input: %q)", context, err, s)
		return 0
	}
	return i
}

// MustParseUint32 parses a uint32 or logs error and returns 0.
// The context parameter provides information about where the parse occurred.
func MustParseUint32(s string, base int, context string) uint32 {
	u, err := strconv.ParseUint(s, base, 32)
	if err != nil {
		log.Printf("ParseUint32 error in %s: %v (input: %q)", context, err, s)
		return 0
	}
	return uint32(u)
}

// LogError logs non-critical errors with context.
func LogError(context string, err error) {
	if err != nil {
		log.Printf("ERROR [%s]: %v", context, err)
	}
}

// FatalError logs and exits for unrecoverable errors.
func FatalError(context string, err error) {
	log.Fatalf("FATAL [%s]: %v", context, err)
}
