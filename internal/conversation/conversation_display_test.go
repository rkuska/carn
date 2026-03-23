package conversation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildDisplayTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		projectName  string
		displayName  string
		date         string
		relativeHint string
		isSubagent   bool
		branch       string
		partCount    int
		want         string
	}{
		{
			name:         "includes relative hint after date",
			projectName:  "proj",
			displayName:  "session",
			date:         "2025-01-15 10:30",
			relativeHint: "2h ago",
			branch:       "main",
			partCount:    2,
			want:         "proj / session  2025-01-15 10:30 (2h ago)  main  (2 parts)",
		},
		{
			name:        "omits relative hint when empty",
			projectName: "proj",
			displayName: "session",
			date:        "2025-01-15 10:30",
			isSubagent:  true,
			want:        "[sub] proj / session  2025-01-15 10:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(
				t,
				tt.want,
				buildDisplayTitle(
					tt.projectName,
					tt.displayName,
					tt.date,
					tt.relativeHint,
					tt.isSubagent,
					tt.branch,
					tt.partCount,
				),
			)
		})
	}
}
