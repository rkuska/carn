package browser

import (
	"testing"

	"github.com/rkuska/carn/internal/app/testutil"
)

const archiveMatchesSourceSubtitle = "analysis complete; archive already matches the configured sources"
const testResyncBetaSlug = "beta"

func assertContainsAll(t testing.TB, got string, wants ...string) {
	t.Helper()
	testutil.AssertContainsAll(t, got, wants...)
}

func assertNotContainsAll(t testing.TB, got string, unwanted ...string) {
	t.Helper()
	testutil.AssertNotContainsAll(t, got, unwanted...)
}

func requireMsgType[T any](t testing.TB, msg any) T {
	t.Helper()
	return testutil.RequireMsgType[T](t, msg)
}

func helpItemKeys(items []helpItem) []string {
	return testutil.HelpItemKeys(items)
}
