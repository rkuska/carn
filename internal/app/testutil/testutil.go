package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	el "github.com/rkuska/carn/internal/app/elements"
)

func AssertContainsAll(t testing.TB, got string, wants ...string) {
	t.Helper()

	for _, want := range wants {
		assert.Contains(t, got, want)
	}
}

func AssertNotContainsAll(t testing.TB, got string, unwanted ...string) {
	t.Helper()

	for _, item := range unwanted {
		assert.NotContains(t, got, item)
	}
}

func HelpItemKeys(items []el.HelpItem) []string {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.Key)
	}
	return keys
}

func RequireMsgType[T any](t testing.TB, msg any) T {
	t.Helper()

	typed, ok := msg.(T)
	require.Truef(t, ok, "unexpected msg type: %T", msg)
	return typed
}

func NewTestTheme() *el.Theme {
	return el.NewTheme(true)
}
