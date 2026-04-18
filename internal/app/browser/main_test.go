package browser

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	syncPaletteFromElements()
	os.Exit(m.Run())
}
