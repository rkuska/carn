package app

import arch "github.com/rkuska/carn/internal/archive"

func initialImportSyncActivity(analysis arch.ImportAnalysis) arch.SyncActivity {
	if analysis.QueuedFileCount() > 0 {
		return arch.SyncActivitySyncingFiles
	}
	if analysis.StoreNeedsBuild {
		return arch.SyncActivityRebuildingStore
	}
	return ""
}

func importSyncActivityLabel(activity arch.SyncActivity) string {
	switch activity {
	case arch.SyncActivityRebuildingStore:
		return "Rebuilding local store"
	case arch.SyncActivitySyncingFiles:
		return "Importing archive files"
	default:
		return "Importing archive files"
	}
}

func syncActivityShowsCurrentFile(activity arch.SyncActivity) bool {
	return activity == arch.SyncActivitySyncingFiles
}

func syncActivityShowsProgress(activity arch.SyncActivity) bool {
	return activity != arch.SyncActivityRebuildingStore
}
