package conversation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSessionMetaDisplaySlugFallbacks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		meta SessionMeta
		want string
	}{
		{
			name: "uses slug when present",
			meta: SessionMeta{Slug: "named", Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			want: "named",
		},
		{
			name: "uses truncated first message",
			meta: SessionMeta{
				FirstMessage: "this is a longer prompt that should be truncated for display",
				Timestamp:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			want: "this is a longer prompt that should be t...",
		},
		{
			name: "falls back to untitled",
			meta: SessionMeta{Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			want: "untitled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.meta.DisplaySlug())
		})
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "45s", FormatDuration(45*time.Second))
	assert.Equal(t, "12m", FormatDuration(12*time.Minute))
	assert.Equal(t, "2h", FormatDuration(2*time.Hour))
	assert.Equal(t, "2h 15m", FormatDuration(2*time.Hour+15*time.Minute))
}
