package stats

import (
	"time"

	conv "github.com/rkuska/carn/internal/conversation"
)

type TimeRange struct {
	Start time.Time
	End   time.Time
}

type Snapshot struct {
	Overview    Overview
	Activity    Activity
	Sessions    Sessions
	Tools       Tools
	Cache       Cache
	Performance Performance
}

type Overview struct {
	SessionCount      int
	MessageCount      int
	Tokens            TokenTotals
	TokenTrend        TokenTrend
	ByModel           []ModelTokens
	ByProject         []ProjectTokens
	ByProviderVersion []ProviderVersionTokens
	TopSessions       []SessionSummary
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
	ReadWriteBashShare     ToolCategoryShare
	TopTools               []ToolStat
	CallsPerSession        []HistogramBucket
	ToolErrorRates         []ToolRateStat
	ToolRejectRates        []ToolRateStat
}

type ToolCategoryShare struct {
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

type ProviderVersionTokens struct {
	Provider conv.Provider
	Version  string
	Tokens   int
}

type SessionSummary struct {
	Project      string
	Slug         string
	SessionID    string
	FilePath     string
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
	Position            int
	AveragePromptTokens float64
	AverageTurnTokens   float64
	SampleCount         int
}

type SplitTurnSeries struct {
	Key     string
	Metrics []PositionTokenMetrics
}

type SessionToolMetrics struct {
	Timestamp        time.Time
	ToolCounts       map[string]int
	ActionCounts     map[string]int
	ToolErrorCounts  map[string]int
	ToolRejectCounts map[string]int
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

type Performance struct {
	Scope       PerformanceScope
	Overall     PerformanceScore
	Outcome     PerformanceLane
	Discipline  PerformanceLane
	Efficiency  PerformanceLane
	Robustness  PerformanceLane
	Diagnostics []PerformanceDiagnostic
}

type PerformanceScope struct {
	SessionCount         int
	Providers            []string
	Models               []string
	PrimaryProvider      string
	PrimaryModel         string
	SingleProvider       bool
	SingleModel          bool
	SingleFamily         bool
	CurrentRange         TimeRange
	BaselineRange        TimeRange
	SequenceLoaded       bool
	SequenceSampleCount  int
	BaselineSessionCount int
}

type PerformanceScore struct {
	Score    int
	HasScore bool
	Trend    TrendDirection
}

type PerformanceLane struct {
	Label    string
	Detail   string
	Score    int
	HasScore bool
	Trend    TrendDirection
	Metrics  []PerformanceMetric
}

type PerformanceMetricStatus int

const (
	PerformanceMetricStatusNone PerformanceMetricStatus = iota
	PerformanceMetricStatusBetter
	PerformanceMetricStatusWorse
	PerformanceMetricStatusFlat
	PerformanceMetricStatusLowSample
)

type PerformanceMetric struct {
	ID             string
	Label          string
	Value          string
	Detail         string
	Question       string
	Formula        string
	Current        float64
	Baseline       float64
	DeltaText      string
	HasBaseline    bool
	Score          int
	ScoreWeight    float64
	HasScore       bool
	Trend          TrendDirection
	Status         PerformanceMetricStatus
	SampleCount    int
	HigherIsBetter bool
	Series         []PerformancePoint
}

type PerformancePoint struct {
	Timestamp   time.Time
	Value       float64
	SampleCount int
}

type PerformanceDiagnostic struct {
	Group       string
	Label       string
	Value       string
	Detail      string
	Current     float64
	Baseline    float64
	HasBaseline bool
	Trend       TrendDirection
	Series      []PerformancePoint
}

type PerformanceSequenceSession = conv.PerformanceSequenceSession
type SessionTurnMetrics = conv.SessionTurnMetrics
type TurnTokens = conv.TurnTokens
type ActivityBucketRow = conv.ActivityBucketRow

type Cache struct {
	TotalCacheRead  int
	TotalCacheWrite int
	TotalPrompt     int
	HitRate         float64
	WriteRate       float64
	MissRate        float64
	ReuseRatio      float64
	Main            CacheSegment
	Subagent        CacheSegment
	DailyHitRate    []DailyRate
	DailyReuseRatio []DailyRate
	DurationBuckets []CacheDurationBucket
}

type CacheSegment struct {
	SessionCount int
	CacheRead    int
	CacheWrite   int
	Prompt       int
	HitRate      float64
	MissTokens   int
}

type CacheDurationBucket struct {
	Label      string
	HitRate    float64
	ReuseRatio float64
	Sessions   int
}

type DailyRate struct {
	Date        time.Time
	Rate        float64
	HasActivity bool
}
