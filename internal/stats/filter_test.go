package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFilterByTimeRange(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 20, 23, 59, 59, 0, time.UTC)
	sessions := []sessionMeta{
		testMeta("before", start.Add(-time.Second)),
		testMeta("start", start),
		testMeta("middle", start.Add(48*time.Hour)),
		testMeta("end", end),
		testMeta("after", end.Add(time.Second)),
	}

	got := FilterByTimeRange(sessions, TimeRange{Start: start, End: end})
	if len(got) != 3 {
		t.Fatalf("len(FilterByTimeRange) = %d, want 3", len(got))
	}
	if got[0].ID != "start" || got[1].ID != "middle" || got[2].ID != "end" {
		t.Fatalf("FilterByTimeRange returned unexpected sessions: %#v", got)
	}
}

func TestFilterByTimeRangeZeroValueReturnsAll(t *testing.T) {
	t.Parallel()

	sessions := []sessionMeta{
		testMeta("one", time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)),
		testMeta("two", time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC)),
	}

	got := FilterByTimeRange(sessions, TimeRange{})
	if len(got) != len(sessions) {
		t.Fatalf("len(FilterByTimeRange) = %d, want %d", len(got), len(sessions))
	}
}

func TestFilterByTimeRangeEmptyInputReturnsNil(t *testing.T) {
	t.Parallel()

	assert.Nil(t, FilterByTimeRange(nil, TimeRange{
		Start: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 1, 20, 23, 59, 59, 0, time.UTC),
	}))
}
