package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		id    string
		value float64
		want  string
	}{
		{
			name:  "verification pass uses percent",
			id:    perfMetricVerificationPass,
			value: 0.82,
			want:  "82.0%",
		},
		{
			name:  "first pass resolution uses percent",
			id:    perfMetricFirstPassResolution,
			value: 0.05,
			want:  "5.0%",
		},
		{
			name:  "blind edit rate uses percent",
			id:    perfMetricBlindEditRate,
			value: 0.14,
			want:  "14.0%",
		},
		{
			name:  "tokens per turn uses single decimal",
			id:    perfMetricTokensPerTurn,
			value: 140,
			want:  "140.0",
		},
		{
			name:  "default metrics use two decimals",
			id:    perfMetricCorrectionBurden,
			value: 3.2,
			want:  "3.20",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, FormatValue(testCase.id, testCase.value))
		})
	}
}
