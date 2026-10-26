package browser

import conv "github.com/rkuska/carn/internal/conversation"

type contentFlags struct {
	hasThinking    bool
	hasToolCalls   bool
	hasToolResults bool
	hasPlans       bool
	hasSidechain   bool
	hasSystem      bool
	hasContext     bool // set by setContextSeries from the Context Series, not scanned
}

func scanContentFlags(messages []conv.Message) contentFlags {
	var flags contentFlags
	for _, msg := range messages {
		flags = flags.accumulate(msg)
		if flags.allSet() {
			break
		}
	}
	return flags
}

func (f contentFlags) accumulate(msg conv.Message) contentFlags {
	if msg.IsVisible() {
		f.hasThinking = f.hasThinking || msg.HasThinking()
		f.hasToolCalls = f.hasToolCalls || len(msg.ToolCalls) > 0
		f.hasToolResults = f.hasToolResults || len(msg.ToolResults) > 0
		f.hasPlans = f.hasPlans || len(msg.Plans) > 0
	}
	f.hasSidechain = f.hasSidechain || msg.IsSidechain
	f.hasSystem = f.hasSystem || !msg.IsVisible()
	return f
}

func (f contentFlags) allSet() bool {
	return f.hasThinking &&
		f.hasToolCalls &&
		f.hasToolResults &&
		f.hasPlans &&
		f.hasSidechain &&
		f.hasSystem
}
