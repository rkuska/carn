package stats

import "time"

type VersionValue struct {
	Version string
	Value   int
}

type GroupedHistogramBucket struct {
	Label    string
	Total    int
	Versions []VersionValue
}

type GroupedNamedStat struct {
	Name     string
	Total    int
	Versions []VersionValue
}

type GroupedRateStat struct {
	Name     string
	Count    int
	Total    int
	Rate     float64
	Versions []VersionValue
}

type GroupedDailyShare struct {
	Date        time.Time
	Prompt      int
	Total       int
	HasActivity bool
	Versions    []VersionValue
}

type ToolsByVersion struct {
	CallsPerSession []GroupedHistogramBucket
	TopTools        []GroupedNamedStat
	ToolErrorRates  []GroupedRateStat
	ToolRejectRates []GroupedRateStat
}

type CacheByVersion struct {
	DailyReadShare  []GroupedDailyShare
	DailyWriteShare []GroupedDailyShare
	SegmentRows     []GroupedNamedStat
	ReadDuration    []GroupedHistogramBucket
	WriteDuration   []GroupedHistogramBucket
}
