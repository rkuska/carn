package canonical

import (
	"sync"
	"time"
)

var (
	timeLocationMu       sync.RWMutex
	timeLocationOverride *time.Location
)

func canonicalTimeLocation() *time.Location {
	timeLocationMu.RLock()
	override := timeLocationOverride
	timeLocationMu.RUnlock()
	if override != nil {
		return override
	}
	return time.Local
}

func SetTimeLocationForTesting(location *time.Location) func() {
	timeLocationMu.Lock()
	previous := timeLocationOverride
	timeLocationOverride = location
	timeLocationMu.Unlock()

	return func() {
		timeLocationMu.Lock()
		timeLocationOverride = previous
		timeLocationMu.Unlock()
	}
}
