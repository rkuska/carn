package conversation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatRelativeTime(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 23, 15, 30, 0, 0, time.UTC)

	tests := []struct {
		name      string
		timestamp time.Time
		want      string
	}{
		{
			name:      "zero time returns empty string",
			timestamp: time.Time{},
			want:      "",
		},
		{
			name:      "under one minute returns now",
			timestamp: now.Add(-59 * time.Second),
			want:      "now",
		},
		{
			name:      "exactly one minute returns minutes",
			timestamp: now.Add(-1 * time.Minute),
			want:      "1m ago",
		},
		{
			name:      "under one hour returns truncated minutes",
			timestamp: now.Add(-(59*time.Minute + 59*time.Second)),
			want:      "59m ago",
		},
		{
			name:      "exactly one hour returns hours",
			timestamp: now.Add(-1 * time.Hour),
			want:      "1h ago",
		},
		{
			name:      "under one day returns truncated hours",
			timestamp: now.Add(-(23*time.Hour + 59*time.Minute)),
			want:      "23h ago",
		},
		{
			name:      "exactly one day returns days",
			timestamp: now.Add(-24 * time.Hour),
			want:      "1d ago",
		},
		{
			name:      "under thirty days returns truncated days",
			timestamp: now.Add(-(29*24*time.Hour + 23*time.Hour)),
			want:      "29d ago",
		},
		{
			name:      "thirty days or more returns empty string",
			timestamp: now.Add(-30 * 24 * time.Hour),
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, FormatRelativeTime(tt.timestamp, now))
		})
	}
}
