package source

import (
	"slices"

	conv "github.com/rkuska/carn/internal/conversation"
)

type providerReport[R any] interface {
	*R
	Empty() bool
	Count() int
	Merge(R)
}

// ProviderReports keeps one report per provider for one import or rebuild pass.
type ProviderReports[R any, PR providerReport[R]] struct {
	reports map[conv.Provider]R
}

func NewProviderReports[R any, PR providerReport[R]]() ProviderReports[R, PR] {
	return ProviderReports[R, PR]{
		reports: make(map[conv.Provider]R),
	}
}

func (r *ProviderReports[R, PR]) MergeProvider(provider conv.Provider, report R) {
	if provider == "" || PR(&report).Empty() {
		return
	}
	if r.reports == nil {
		r.reports = make(map[conv.Provider]R)
	}
	current := r.reports[provider]
	PR(&current).Merge(report)
	r.reports[provider] = current
}

func (r *ProviderReports[R, PR]) Merge(other ProviderReports[R, PR]) {
	for provider, report := range other.reports {
		r.MergeProvider(provider, report)
	}
}

func (r ProviderReports[R, PR]) Empty() bool {
	return len(r.reports) == 0
}

func (r ProviderReports[R, PR]) Count() int {
	total := 0
	for _, report := range r.reports {
		total += PR(&report).Count()
	}
	return total
}

func (r ProviderReports[R, PR]) Providers() []conv.Provider {
	providers := make([]conv.Provider, 0, len(r.reports))
	for provider, report := range r.reports {
		if PR(&report).Empty() {
			continue
		}
		providers = append(providers, provider)
	}
	slices.Sort(providers)
	return providers
}

func (r ProviderReports[R, PR]) Report(provider conv.Provider) R {
	return r.reports[provider]
}
