package app

import (
	"fmt"
	"strings"

	src "github.com/rkuska/carn/internal/source"
)

func driftNotification(reports src.ProviderDriftReports) (notification, bool) {
	if reports.Empty() {
		return notification{}, false
	}

	providers := reports.Providers()
	if len(providers) == 1 {
		provider := providers[0]
		return infoNotification(fmt.Sprintf(
			"format drift: %d unknown fields/types detected in %s source (check logs)",
			reports.Report(provider).Count(),
			provider,
		)).Notification, true
	}

	parts := make([]string, 0, len(providers))
	for _, provider := range providers {
		parts = append(parts, fmt.Sprintf("%s %d", provider, reports.Report(provider).Count()))
	}
	return infoNotification(fmt.Sprintf(
		"format drift: %s unknown fields/types detected (check logs)",
		strings.Join(parts, ", "),
	)).Notification, true
}
