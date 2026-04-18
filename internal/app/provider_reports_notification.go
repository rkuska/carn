package app

import (
	"fmt"
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
)

type providerCountReport interface {
	Count() int
}

type providerCountReports[R providerCountReport] interface {
	Empty() bool
	Providers() []conv.Provider
	Report(conv.Provider) R
}

func providerCountsNotification[R providerCountReport, PR providerCountReports[R]](
	reports PR,
	singleFmt string,
	multiFmt string,
	build func(string) notification,
) (notification, bool) {
	if reports.Empty() {
		return notification{}, false
	}

	providers := reports.Providers()
	if len(providers) == 1 {
		provider := providers[0]
		return build(fmt.Sprintf(
			singleFmt,
			reports.Report(provider).Count(),
			provider,
		)), true
	}

	parts := make([]string, 0, len(providers))
	for _, provider := range providers {
		parts = append(parts, fmt.Sprintf("%s %d", provider, reports.Report(provider).Count()))
	}
	return build(fmt.Sprintf(multiFmt, strings.Join(parts, ", "))), true
}
