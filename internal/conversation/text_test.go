package conversation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncate(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "hello world", Truncate(" \r\nhello\nworld ", 20))
	assert.Equal(t, "hello...", Truncate("hello world", 5))
}

func TestTruncatePreserveNewlines(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "hello\nworld", TruncatePreserveNewlines("hello\r\nworld", 20))
	assert.Equal(t, "hello\n...", TruncatePreserveNewlines("hello\nworld", 5))
}

func TestCompactCWD(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "src/project", CompactCWD("/workspace/src/project"))
	assert.Equal(t, "workspace/project", CompactCWD("/workspace/project"))
	assert.Equal(t, "project", CompactCWD("/project"))
	assert.Equal(t, "me/project", CompactCWD(`C:\Users\me\project`))
	assert.Equal(t, "", CompactCWD(""))
}

func TestProjectName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "project", ProjectName("/workspace/src/project"))
	assert.Equal(t, "project", ProjectName(`C:\Users\me\project`))
	assert.Equal(t, "", ProjectName(""))
}
