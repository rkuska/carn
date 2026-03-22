package stats

import "time"

type TimeRange struct {
	Start time.Time
	End   time.Time
}

type Snapshot struct {
	Overview Overview
	Activity Activity
	Sessions Sessions
	Tools    Tools
}

type Overview struct {
	SessionCount int
	MessageCount int
	Tokens       TokenTotals
	ByModel      []ModelTokens
	ByProject    []ProjectTokens
	TopSessions  []SessionSummary
}

type Activity struct {
	ActiveDays    int
	TotalDays     int
	CurrentStreak int
	LongestStreak int
	DailySessions []DailyCount
	DailyMessages []DailyCount
	DailyTokens   []DailyCount
	Heatmap       [7][24]int
}

type Sessions struct {
	AverageDuration       time.Duration
	AverageMessages       float64
	UserMessageCount      int
	AssistantMessageCount int
	UserAssistantRatio    float64
	AbandonedCount        int
	AbandonedRate         float64
	DurationHistogram     []HistogramBucket
	MessageHistogram      []HistogramBucket
	TokenGrowth           []PositionTokens
}

type Tools struct {
	TotalCalls             int
	AverageCallsPerSession float64
	ErrorRate              float64
	ReadWriteBashRatio     ToolCategoryRatio
	TopTools               []ToolStat
	CallsPerSession        []HistogramBucket
	ToolErrorRates         []ToolErrorRate
}

type ToolCategoryRatio struct {
	Read  float64
	Write float64
	Bash  float64
}

type TokenTotals struct {
	Total      int
	Input      int
	Output     int
	CacheRead  int
	CacheWrite int
}

type ModelTokens struct {
	Model  string
	Tokens int
}

type ProjectTokens struct {
	Project string
	Tokens  int
}

type SessionSummary struct {
	Project      string
	Slug         string
	Timestamp    time.Time
	MessageCount int
	Duration     time.Duration
	Tokens       int
}

type DailyCount struct {
	Date  time.Time
	Count int
}

type HistogramBucket struct {
	Label string
	Count int
}

type PositionTokens struct {
	Position      int
	AverageTokens float64
	SampleCount   int
}

type ToolStat struct {
	Name  string
	Count int
}

type ToolErrorRate struct {
	Name   string
	Errors int
	Total  int
	Rate   float64
}
