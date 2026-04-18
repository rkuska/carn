package stats

import (
	"os"
	"testing"

	el "github.com/rkuska/carn/internal/app/elements"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func testTheme() *el.Theme {
	return el.NewTheme(true)
}
