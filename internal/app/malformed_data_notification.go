package app

import src "github.com/rkuska/carn/internal/source"

func malformedDataNotification(reports src.ProviderMalformedDataReports) (notification, bool) {
	return providerCountsNotification(
		reports,
		"rebuild warnings: skipped %d malformed items in %s source (check logs)",
		"rebuild warnings: skipped malformed items (%s; check logs)",
		func(text string) notification {
			return errorNotification(text).Notification
		},
	)
}
