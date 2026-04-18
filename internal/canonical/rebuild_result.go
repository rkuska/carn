package canonical

import src "github.com/rkuska/carn/internal/source"

type RebuildResult struct {
	Drift         src.ProviderDriftReports
	MalformedData src.ProviderMalformedDataReports
}
