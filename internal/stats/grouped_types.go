package stats

import "time"

type SplitValue struct {
	Key   string
	Value int
}

type SplitHistogramBucket struct {
	Label  string
	Total  int
	Splits []SplitValue
}

type SplitNamedStat struct {
	Name   string
	Total  int
	Splits []SplitValue
}

type SplitRateStat struct {
	Name   string
	Count  int
	Total  int
	Rate   float64
	Splits []SplitValue
}

type SplitDailyShare struct {
	Date         time.Time
	Prompt       int
	Total        int
	HasActivity  bool
	Splits       []SplitValue
	PromptSplits []SplitValue
}

type SplitDailyValueSeries struct {
	Key    string
	Values []DailyValue
}

type ToolsBySplit struct {
	CallsPerSession []SplitHistogramBucket
	TopTools        []SplitNamedStat
	ToolErrorRates  []SplitRateStat
	ToolRejectRates []SplitRateStat
}

type CacheBySplit struct {
	DailyReadShare  []SplitDailyShare
	DailyWriteShare []SplitDailyShare
	SegmentRows     []SplitNamedStat
	ReadDuration    []SplitHistogramBucket
	WriteDuration   []SplitHistogramBucket
}
