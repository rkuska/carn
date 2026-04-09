package conversation

type NormalizedActionType string

const (
	NormalizedActionRead     NormalizedActionType = "read"
	NormalizedActionSearch   NormalizedActionType = "search"
	NormalizedActionMutate   NormalizedActionType = "mutate"
	NormalizedActionRewrite  NormalizedActionType = "rewrite"
	NormalizedActionExecute  NormalizedActionType = "execute"
	NormalizedActionTest     NormalizedActionType = "test"
	NormalizedActionBuild    NormalizedActionType = "build"
	NormalizedActionWeb      NormalizedActionType = "web"
	NormalizedActionPlan     NormalizedActionType = "plan"
	NormalizedActionDelegate NormalizedActionType = "delegate"
	NormalizedActionOther    NormalizedActionType = "other"
)

type ActionTargetType string

const (
	ActionTargetFilePath ActionTargetType = "file_path"
	ActionTargetPattern  ActionTargetType = "pattern"
	ActionTargetQuery    ActionTargetType = "query"
	ActionTargetURL      ActionTargetType = "url"
	ActionTargetCommand  ActionTargetType = "command"
	ActionTargetPlanPath ActionTargetType = "plan_path"
)

type ActionTarget struct {
	Type  ActionTargetType
	Value string
}

type NormalizedAction struct {
	Type    NormalizedActionType
	Targets []ActionTarget
}

func (a NormalizedAction) IsZero() bool {
	return a.Type == ""
}

type MessagePerformanceMeta struct {
	ReasoningBlockCount     int
	ReasoningRedactionCount int
	StopReason              string
	Phase                   string
	Effort                  string
}

type SessionPerformanceMeta struct {
	ReasoningBlockCount     int
	ReasoningRedactionCount int
	MaxThinkingTokens       int
	ModelContextWindow      int
	DurationMS              int
	RetryAttemptCount       int
	RetryDelayMS            int
	MaxRetries              int
	AbortCount              int
	CompactionCount         int
	MicroCompactionCount    int
	TaskStartedCount        int
	TaskCompleteCount       int
	ContextCompactedCount   int
	RateLimitSnapshotCount  int
	APIErrorCounts          map[string]int
	StopReasonCounts        map[string]int
	PhaseCounts             map[string]int
	EffortCounts            map[string]int
	ServerToolUseCounts     map[string]int
	ServiceTierCounts       map[string]int
	SpeedCounts             map[string]int
}
