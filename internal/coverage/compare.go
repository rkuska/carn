package coverage

import "sort"

const (
	totalRegressionTolerance   = 0.05
	packageRegressionTolerance = 0.5
)

func Compare(baseline Baseline, current Snapshot) []Regression {
	regressions := make([]Regression, 0)
	if totalCoverageRegressed(baseline.Total, current.Total) {
		regressions = append(regressions, Regression{
			Scope:    ScopeTotal,
			Name:     "total",
			Baseline: baseline.Total,
			Current:  current.Total,
		})
	}

	for pkg, baselineRatio := range baseline.Packages {
		currentRatio, ok := current.Packages[pkg]
		if !ok {
			continue
		}
		if !packageCoverageRegressed(baselineRatio, currentRatio) {
			continue
		}

		regressions = append(regressions, Regression{
			Scope:    ScopePackage,
			Name:     pkg,
			Baseline: baselineRatio,
			Current:  currentRatio,
		})
	}

	sort.Slice(regressions, func(i, j int) bool {
		if regressions[i].Scope != regressions[j].Scope {
			return regressions[i].Scope == ScopeTotal
		}
		return regressions[i].Name < regressions[j].Name
	})

	return regressions
}

func totalCoverageRegressed(baseline, current Ratio) bool {
	if !current.lessThan(baseline) {
		return false
	}
	return baseline.Percent()-current.Percent() > totalRegressionTolerance
}

func packageCoverageRegressed(baseline, current Ratio) bool {
	if !current.lessThan(baseline) {
		return false
	}
	return baseline.Percent()-current.Percent() > packageRegressionTolerance
}
