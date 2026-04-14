package browser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const archiveMatchesSourceSubtitle = "analysis complete; archive already matches the configured sources"
const testResyncBetaSlug = "beta"

func requireMsgType[T any](t testing.TB, msg any) T {
	t.Helper()

	typed, ok := msg.(T)
	require.Truef(t, ok, "unexpected msg type: %T", msg)
	return typed
}

func assertContainsAll(t testing.TB, got string, wants ...string) {
	t.Helper()

	for _, want := range wants {
		assert.Contains(t, got, want)
	}
}

func assertNotContainsAll(t testing.TB, got string, unwanted ...string) {
	t.Helper()

	for _, item := range unwanted {
		assert.NotContains(t, got, item)
	}
}
