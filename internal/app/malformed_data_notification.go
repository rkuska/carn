package app

import (
	"fmt"
	"strings"

	src "github.com/rkuska/carn/internal/source"
)

func malformedDataNotification(reports src.ProviderMalformedDataReports) (notification, bool) {
	if reports.Empty() {
		return notification{}, false
	}

	providers := reports.Providers()
	if len(providers) == 1 {
		provider := providers[0]
		return errorNotification(fmt.Sprintf(
			"rebuild warnings: skipped %d malformed item in %s source (check logs)",
			reports.Report(provider).Count(),
			provider,
		)).Notification, true
	}

	parts := make([]string, 0, len(providers))
	for _, provider := range providers {
		parts = append(parts, fmt.Sprintf("%s %d", provider, reports.Report(provider).Count()))
	}
	return errorNotification(fmt.Sprintf(
		"rebuild warnings: skipped malformed items (%s; check logs)",
		strings.Join(parts, ", "),
	)).Notification, true
}
