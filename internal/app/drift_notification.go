package app

import src "github.com/rkuska/carn/internal/source"

func driftNotification(reports src.ProviderDriftReports) (notification, bool) {
	return providerCountsNotification(
		reports,
		"format drift: %d unknown fields/types detected in %s source (check logs)",
		"format drift: %s unknown fields/types detected (check logs)",
		func(text string) notification {
			return infoNotification(text).Notification
		},
	)
}
