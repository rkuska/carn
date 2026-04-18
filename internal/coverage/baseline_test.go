package coverage

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareDetectsTotalAndPackageRegressions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		baseline Baseline
		current  Snapshot
		want     []Regression
	}{
		{
			name: "repo total and package regression",
			baseline: Baseline{
				SchemaVersion: currentSchemaVersion,
				ModulePath:    "github.com/rkuska/carn",
				Total: Ratio{
					Covered:    8,
					Statements: 10,
				},
				Packages: map[string]Ratio{
					"github.com/rkuska/carn/internal/app": {
						Covered:    4,
						Statements: 5,
					},
					"github.com/rkuska/carn/internal/archive": {
						Covered:    4,
						Statements: 5,
					},
				},
			},
			current: Snapshot{
				Total: Ratio{
					Covered:    7,
					Statements: 10,
				},
				Packages: map[string]Ratio{
					"github.com/rkuska/carn/internal/app": {
						Covered:    3,
						Statements: 5,
					},
					"github.com/rkuska/carn/internal/archive": {
						Covered:    4,
						Statements: 5,
					},
				},
			},
			want: []Regression{
				{
					Scope: ScopeTotal,
					Name:  "total",
					Baseline: Ratio{
						Covered:    8,
						Statements: 10,
					},
					Current: Ratio{
						Covered:    7,
						Statements: 10,
					},
				},
				{
					Scope: ScopePackage,
					Name:  "github.com/rkuska/carn/internal/app",
					Baseline: Ratio{
						Covered:    4,
						Statements: 5,
					},
					Current: Ratio{
						Covered:    3,
						Statements: 5,
					},
				},
			},
		},
		{
			name: "new and removed packages do not fail package comparison",
			baseline: Baseline{
				SchemaVersion: currentSchemaVersion,
				ModulePath:    "github.com/rkuska/carn",
				Total: Ratio{
					Covered:    8,
					Statements: 10,
				},
				Packages: map[string]Ratio{
					"github.com/rkuska/carn/internal/app": {
						Covered:    4,
						Statements: 5,
					},
				},
			},
			current: Snapshot{
				Total: Ratio{
					Covered:    9,
					Statements: 11,
				},
				Packages: map[string]Ratio{
					"github.com/rkuska/carn/internal/app": {
						Covered:    4,
						Statements: 5,
					},
					"github.com/rkuska/carn/internal/newpkg": {
						Covered:    5,
						Statements: 6,
					},
				},
			},
			want: []Regression{},
		},
		{
			name: "small package wobble within tolerance passes",
			baseline: Baseline{
				SchemaVersion: currentSchemaVersion,
				ModulePath:    "github.com/rkuska/carn",
				Total: Ratio{
					Covered:    80,
					Statements: 100,
				},
				Packages: map[string]Ratio{
					"github.com/rkuska/carn/internal/elements": {
						Covered:    1285,
						Statements: 1500,
					},
				},
			},
			current: Snapshot{
				Total: Ratio{
					Covered:    80,
					Statements: 100,
				},
				Packages: map[string]Ratio{
					"github.com/rkuska/carn/internal/elements": {
						Covered:    1284,
						Statements: 1500,
					},
				},
			},
			want: []Regression{},
		},
		{
			name: "small total wobble within tolerance passes",
			baseline: Baseline{
				SchemaVersion: currentSchemaVersion,
				ModulePath:    "github.com/rkuska/carn",
				Total: Ratio{
					Covered:    12425,
					Statements: 15593,
				},
			},
			current: Snapshot{
				Total: Ratio{
					Covered:    12422,
					Statements: 15593,
				},
			},
			want: []Regression{},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.want, Compare(testCase.baseline, testCase.current))
		})
	}
}

func TestWriteBaselineRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "COVERAGE_BASELINE.json")
	want := Baseline{
		SchemaVersion: currentSchemaVersion,
		ModulePath:    "github.com/rkuska/carn",
		Total: Ratio{
			Covered:    9,
			Statements: 10,
		},
		Packages: map[string]Ratio{
			"github.com/rkuska/carn/internal/app": {
				Covered:    4,
				Statements: 5,
			},
			"github.com/rkuska/carn/internal/archive": {
				Covered:    5,
				Statements: 5,
			},
		},
	}

	require.NoError(t, WriteBaseline(path, want))

	got, err := ReadBaseline(path)
	require.NoError(t, err)

	assert.Equal(t, want, got)
}
