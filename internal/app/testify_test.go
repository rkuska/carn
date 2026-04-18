package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertContainsAll(t testing.TB, got string, wants ...string) {
	t.Helper()

	for _, want := range wants {
		assert.Contains(t, got, want)
	}
}

func requireMsgType[T any](t testing.TB, msg any) T {
	t.Helper()

	typed, ok := msg.(T)
	require.Truef(t, ok, "unexpected msg type: %T", msg)
	return typed
}
