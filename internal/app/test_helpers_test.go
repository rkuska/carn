package app

import (
	"testing"

	"github.com/rkuska/carn/internal/app/testutil"
)

func requireAs[T any](t *testing.T, v any) T {
	t.Helper()
	result, ok := v.(T)
	if !ok {
		t.Fatalf("type assertion failed: expected %T, got %T", result, v)
	}
	return result
}

func assertContainsAll(t testing.TB, got string, wants ...string) {
	t.Helper()
	testutil.AssertContainsAll(t, got, wants...)
}

func requireMsgType[T any](t testing.TB, msg any) T {
	t.Helper()
	return testutil.RequireMsgType[T](t, msg)
}
