package coverage

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSnapshotAggregatesTotalAndPackages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		profile      string
		wantTotal    Ratio
		wantPackages map[string]Ratio
	}{
		{
			name: "root and nested packages",
			profile: strings.Join([]string{
				"mode: set",
				"github.com/rkuska/carn/main.go:10.13,11.76 2 0",
				"github.com/rkuska/carn/internal/app/app.go:1.1,2.1 3 1",
				"github.com/rkuska/carn/internal/app/app.go:3.1,4.1 1 0",
				"github.com/rkuska/carn/internal/archive/archive.go:5.1,8.1 4 2",
			}, "\n"),
			wantTotal: Ratio{
				Covered:    7,
				Statements: 10,
			},
			wantPackages: map[string]Ratio{
				"github.com/rkuska/carn": {
					Covered:    0,
					Statements: 2,
				},
				"github.com/rkuska/carn/internal/app": {
					Covered:    3,
					Statements: 4,
				},
				"github.com/rkuska/carn/internal/archive": {
					Covered:    4,
					Statements: 4,
				},
			},
		},
		{
			name: "count mode treats any positive count as covered",
			profile: strings.Join([]string{
				"mode: count",
				"github.com/rkuska/carn/internal/source/source.go:1.1,3.1 5 0",
				"github.com/rkuska/carn/internal/source/source.go:4.1,8.1 6 3",
			}, "\n"),
			wantTotal: Ratio{
				Covered:    6,
				Statements: 11,
			},
			wantPackages: map[string]Ratio{
				"github.com/rkuska/carn/internal/source": {
					Covered:    6,
					Statements: 11,
				},
			},
		},
		{
			name: "duplicate cover blocks merge once",
			profile: strings.Join([]string{
				"mode: set",
				"github.com/rkuska/carn/internal/coverage/profile.go:1.1,2.1 3 0",
				"github.com/rkuska/carn/internal/coverage/profile.go:1.1,2.1 3 1",
				"github.com/rkuska/carn/internal/coverage/profile.go:3.1,4.1 2 0",
				"github.com/rkuska/carn/internal/coverage/profile.go:3.1,4.1 2 0",
			}, "\n"),
			wantTotal: Ratio{
				Covered:    3,
				Statements: 5,
			},
			wantPackages: map[string]Ratio{
				"github.com/rkuska/carn/internal/coverage": {
					Covered:    3,
					Statements: 5,
				},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseSnapshot(strings.NewReader(testCase.profile))
			require.NoError(t, err)

			assert.Equal(t, testCase.wantTotal, got.Total)
			assert.Equal(t, testCase.wantPackages, got.Packages)
		})
	}
}

func TestParseSnapshotRejectsInvalidProfiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		profile string
	}{
		{
			name: "missing mode header",
			profile: strings.Join([]string{
				"github.com/rkuska/carn/internal/app/app.go:1.1,2.1 3 1",
			}, "\n"),
		},
		{
			name: "malformed block",
			profile: strings.Join([]string{
				"mode: set",
				"github.com/rkuska/carn/internal/app/app.go:1.1,2.1 nope 1",
			}, "\n"),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseSnapshot(strings.NewReader(testCase.profile))
			require.Error(t, err)
		})
	}
}
