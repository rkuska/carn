package app

import "testing"

func requireAs[T any](t *testing.T, v any) T {
	t.Helper()
	result, ok := v.(T)
	if !ok {
		t.Fatalf("type assertion failed: expected %T, got %T", result, v)
	}
	return result
}
