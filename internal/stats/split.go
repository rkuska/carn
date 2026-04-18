package stats

import (
	"strings"

	conv "github.com/rkuska/carn/internal/conversation"
)

const UnknownSplitKey = "unknown"

type SplitDimension int

const (
	SplitDimensionNone SplitDimension = iota
	SplitDimensionProvider
	SplitDimensionVersion
	SplitDimensionModel
	SplitDimensionProject
)

func (d SplitDimension) Label() string {
	switch d {
	case SplitDimensionProvider:
		return "Provider"
	case SplitDimensionVersion:
		return "Version"
	case SplitDimensionModel:
		return "Model"
	case SplitDimensionProject:
		return "Project"
	case SplitDimensionNone:
		return ""
	default:
		return ""
	}
}

func (d SplitDimension) IsActive() bool {
	return d != SplitDimensionNone
}

func (d SplitDimension) SessionKey(session conv.SessionMeta) string {
	switch d {
	case SplitDimensionProvider:
		return providerLabelOrUnknown(session.Provider)
	case SplitDimensionVersion:
		return NormalizeVersionLabel(session.Version)
	case SplitDimensionModel:
		return labelOrUnknown(session.Model)
	case SplitDimensionProject:
		return labelOrUnknown(session.Project.DisplayName)
	case SplitDimensionNone:
		return ""
	default:
		return ""
	}
}

func (d SplitDimension) TurnMetricsKey(metrics conv.SessionTurnMetrics) string {
	switch d {
	case SplitDimensionProvider:
		return providerLabelOrUnknown(metrics.Provider)
	case SplitDimensionVersion:
		return NormalizeVersionLabel(metrics.Version)
	case SplitDimensionNone, SplitDimensionModel, SplitDimensionProject:
		return ""
	default:
		return ""
	}
}

func (d SplitDimension) SupportsTurnMetrics() bool {
	return d == SplitDimensionProvider || d == SplitDimensionVersion
}

func providerLabelOrUnknown(p conv.Provider) string {
	label := p.Label()
	if label == "" {
		return UnknownSplitKey
	}
	return label
}

func labelOrUnknown(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return UnknownSplitKey
	}
	return trimmed
}
