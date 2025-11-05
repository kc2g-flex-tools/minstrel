package format

import (
	"fmt"
	"math"
)

// FrequencyMHz formats a frequency in MHz with dot separators
// Example: 14.250000 MHz -> "14.250.000"
func FrequencyMHz(fMHz float64) string {
	if fMHz == 0 {
		return "0"
	}

	freq := int(math.Round(fMHz * 1e6))
	parts := []string{}

	for freq > 0 {
		part := freq % 1000
		freq = freq / 1000

		if freq > 0 {
			parts = append([]string{fmt.Sprintf("%03d", part)}, parts...)
		} else {
			parts = append([]string{fmt.Sprintf("%d", part)}, parts...)
		}
	}

	result := ""
	for i, part := range parts {
		if i == 0 {
			result = part
		} else {
			result = result + "." + part
		}
	}

	return result
}
