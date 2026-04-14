package browser

import arch "github.com/rkuska/carn/internal/archive"

func resyncSyncActivityLabel(activity arch.SyncActivity) string {
	if activity == arch.SyncActivityRebuildingStore {
		return "rebuilding local store"
	}
	return ""
}
