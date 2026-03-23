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
	TokenTrend   TokenTrend
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
	ClaudeTurnMetrics     []PositionTokenMetrics
}

type Tools struct {
	TotalCalls             int
	AverageCallsPerSession float64
	ErrorRate              float64
	RejectionRate          float64
	ReadWriteBashRatio     ToolCategoryRatio
	TopTools               []ToolStat
	CallsPerSession        []HistogramBucket
	ToolErrorRates         []ToolRateStat
	ToolRejectRates        []ToolRateStat
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

type PositionTokenMetrics struct {
	Position           int
	AverageInputTokens float64
	AverageTurnTokens  float64
	SampleCount        int
}

type SessionTurnMetrics struct {
	Timestamp time.Time
	Turns     []TurnTokens
}

type SessionToolMetrics struct {
	Timestamp        time.Time
	ToolCounts       map[string]int
	ToolErrorCounts  map[string]int
	ToolRejectCounts map[string]int
}

type TurnTokens struct {
	InputTokens int
	TurnTokens  int
}

type ToolStat struct {
	Name  string
	Count int
}

type ToolRateStat struct {
	Name  string
	Count int
	Total int
	Rate  float64
}

type TrendDirection int

const (
	TrendDirectionNone TrendDirection = iota
	TrendDirectionUp
	TrendDirectionDown
	TrendDirectionFlat
)

type TokenTrend struct {
	Direction     TrendDirection
	PercentChange int
}
