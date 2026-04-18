package source

import (
	"context"
	"sort"
	"strings"

	"github.com/rs/zerolog"

	conv "github.com/rkuska/carn/internal/conversation"
)

// DriftFinding identifies one unknown field or type observed during a scan.
type DriftFinding struct {
	Category string
	Value    string
}

// DriftReport accumulates unique findings for one provider scan pass.
type DriftReport struct {
	findings map[DriftFinding]struct{}
}

func NewDriftReport() DriftReport {
	return DriftReport{
		findings: make(map[DriftFinding]struct{}),
	}
}

func (r *DriftReport) Record(category, value string) {
	category = strings.TrimSpace(category)
	value = strings.TrimSpace(value)
	if category == "" || value == "" {
		return
	}
	if r.findings == nil {
		r.findings = make(map[DriftFinding]struct{})
	}
	r.findings[DriftFinding{Category: category, Value: value}] = struct{}{}
}

func (r DriftReport) Empty() bool {
	return len(r.findings) == 0
}

func (r DriftReport) Count() int {
	return len(r.findings)
}

func (r *DriftReport) Merge(other DriftReport) {
	if len(other.findings) == 0 {
		return
	}
	if r.findings == nil {
		r.findings = make(map[DriftFinding]struct{}, len(other.findings))
	}
	for finding := range other.findings {
		r.findings[finding] = struct{}{}
	}
}

func (r DriftReport) Findings() []DriftFinding {
	findings := make([]DriftFinding, 0, len(r.findings))
	for finding := range r.findings {
		findings = append(findings, finding)
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Category != findings[j].Category {
			return findings[i].Category < findings[j].Category
		}
		return findings[i].Value < findings[j].Value
	})
	return findings
}

func (r DriftReport) GroupedByCategory() map[string][]string {
	grouped := make(map[string][]string)
	for _, finding := range r.Findings() {
		grouped[finding.Category] = append(grouped[finding.Category], finding.Value)
	}
	return grouped
}

func (r DriftReport) Log(ctx context.Context, provider conv.Provider) {
	if r.Empty() {
		return
	}
	for category, values := range r.GroupedByCategory() {
		zerolog.Ctx(ctx).Warn().
			Str("provider", string(provider)).
			Str("category", category).
			Int("count", len(values)).
			Msgf("format drift detected: %s", strings.Join(values, ", "))
	}
}

// ProviderDriftReports keeps per-provider drift findings for one import or rebuild.
type ProviderDriftReports = ProviderReports[DriftReport, *DriftReport]

func NewProviderDriftReports() ProviderDriftReports {
	return NewProviderReports[DriftReport]()
}
