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

func TestPerformanceMetricIsRatio(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		id   string
		want bool
	}{
		{
			name: "verification pass is ratio",
			id:   perfMetricVerificationPass,
			want: true,
		},
		{
			name: "retry burden is ratio",
			id:   perfMetricRetryBurden,
			want: true,
		},
		{
			name: "correction burden is not ratio",
			id:   perfMetricCorrectionBurden,
			want: false,
		},
		{
			name: "tokens per turn is not ratio",
			id:   perfMetricTokensPerTurn,
			want: false,
		},
		{
			name: "unknown metric defaults to non ratio",
			id:   "unknown",
			want: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, performanceMetricIsRatio(testCase.id))
		})
	}
}

func TestFormatPerformanceDelta(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		id    string
		delta float64
		want  string
	}{
		{
			name:  "ratio metrics use points",
			id:    perfMetricErrorRate,
			delta: 0.12,
			want:  "+12.0 pts",
		},
		{
			name:  "non ratio metrics use plain decimals",
			id:    perfMetricCorrectionBurden,
			delta: -1.4,
			want:  "-1.4",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, formatPerformanceDelta(testCase.id, testCase.delta))
		})
	}
}

func TestTopCountEntryBreaksTiesByName(t *testing.T) {
	t.Parallel()

	name, count := topCountEntry(map[string]int{
		"zeta":  3,
		"beta":  3,
		"alpha": 1,
	})

	assert.Equal(t, "beta", name)
	assert.Equal(t, 3, count)
}
