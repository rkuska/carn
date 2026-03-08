package app

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	initPalette(true)
	os.Exit(m.Run())
}
