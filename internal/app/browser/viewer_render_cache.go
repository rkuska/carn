package browser

const viewerRenderCacheLimit = 8

type viewerRenderKey struct {
	conversationKey       string
	conversationName      string
	projectName           string
	conversationTimestamp int64
	sessionID             string
	sessionTimestamp      int64
	sessionLastTimestamp  int64
	messageCount          int
	contentWidth          int
	wrapWidth             int
	opts                  transcriptOptions
	planExpanded          bool
	selectionMode         bool
	glamourStyle          string
	timestampFormat       string
}

type viewerRenderValue struct {
	rawContent  string
	baseContent string
	searchLines []searchLineIndex
	turnAnchors []int
}

func (m viewerModel) renderKey() viewerRenderKey {
	return viewerRenderKey{
		conversationKey:       m.conversation.CacheKey(),
		conversationName:      m.conversation.DisplayName(),
		projectName:           m.conversation.Project.DisplayName,
		conversationTimestamp: m.conversation.Timestamp().UnixNano(),
		sessionID:             m.session.Meta.ID,
		sessionTimestamp:      m.session.Meta.Timestamp.UnixNano(),
		sessionLastTimestamp:  m.session.Meta.LastTimestamp.UnixNano(),
		messageCount:          len(m.session.Messages),
		contentWidth:          m.contentWidth(),
		wrapWidth:             m.markdownWrapWidth(),
		opts:                  m.opts,
		planExpanded:          m.planExpanded,
		selectionMode:         m.selectionMode,
		glamourStyle:          m.glamourStyle,
		timestampFormat:       m.timestampFormat,
	}
}

func (m viewerModel) cachedRender(key viewerRenderKey) (viewerRenderValue, bool) {
	if m.renderCache == nil {
		return viewerRenderValue{}, false
	}
	value, ok := m.renderCache[key]
	return value, ok
}

func (m viewerModel) storeRenderCache(key viewerRenderKey, value viewerRenderValue) viewerModel {
	if m.renderCache == nil {
		m.renderCache = make(map[viewerRenderKey]viewerRenderValue, viewerRenderCacheLimit)
	}
	if _, ok := m.renderCache[key]; !ok && len(m.renderCache) >= viewerRenderCacheLimit {
		m.renderCache = make(map[viewerRenderKey]viewerRenderValue, viewerRenderCacheLimit)
	}
	m.renderCache[key] = value
	return m
}
