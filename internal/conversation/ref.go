package conversation

type Provider string

const ProviderClaude Provider = "claude"

type Ref struct {
	Provider Provider
	ID       string
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
