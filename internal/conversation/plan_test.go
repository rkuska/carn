package conversation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatPlan(t *testing.T) {
	t.Parallel()

	formatted := FormatPlan(Plan{
		FilePath: "/tmp/plans/demo.md",
		Content:  "# Demo\n\nstep 1",
	})

	assert.Equal(t, "Plan: demo.md\n\n# Demo\n\nstep 1", formatted)
}

func TestPlanHelpers(t *testing.T) {
	t.Parallel()

	messages := []Message{
		{
			Role:  RoleUser,
			Text:  "first",
			Plans: []Plan{{FilePath: "a.md", Content: "plan a", Timestamp: time.Unix(1, 0)}},
		},
		{
			Role: RoleAssistant,
			Text: "no plan",
		},
		{
			Role: RoleUser,
			Text: "second",
			Plans: []Plan{
				{FilePath: "b.md", Content: "plan b", Timestamp: time.Unix(2, 0)},
				{FilePath: "c.md", Content: "plan c", Timestamp: time.Unix(3, 0)},
			},
		},
	}

	last, ok := LastPlan(messages)
	assert.True(t, ok)
	assert.Equal(t, "c.md", last.FilePath)

	assert.Equal(t, []Plan{
		{FilePath: "a.md", Content: "plan a", Timestamp: time.Unix(1, 0)},
		{FilePath: "b.md", Content: "plan b", Timestamp: time.Unix(2, 0)},
		{FilePath: "c.md", Content: "plan c", Timestamp: time.Unix(3, 0)},
	}, AllPlans(messages))

	assert.Equal(t, 3, CountPlans(messages))
}

func TestLastPlanMissing(t *testing.T) {
	t.Parallel()

	last, ok := LastPlan([]Message{{Role: RoleAssistant, Text: "hello"}})
	assert.False(t, ok)
	assert.Equal(t, Plan{}, last)
}
