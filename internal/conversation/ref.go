package conversation

import "strings"

type Provider string

const (
	ProviderClaude Provider = "claude"
	ProviderCodex  Provider = "codex"
)

type Ref struct {
	Provider Provider
	ID       string
}

func (p Provider) Label() string {
	switch p {
	case ProviderClaude:
		return "Claude"
	case ProviderCodex:
		return "Codex"
	case "":
		return ""
	default:
		return strings.TrimSpace(string(p))
	}
}

func (r Ref) CacheKey() string {
	if r.ID == "" {
		return ""
	}
	if r.Provider == "" {
		return r.ID
	}
	return string(r.Provider) + ":" + r.ID
}
