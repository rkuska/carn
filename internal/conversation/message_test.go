package conversation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageIsVisible(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  Message
		want bool
	}{
		{
			name: "zero value visible",
			msg:  Message{},
			want: true,
		},
		{
			name: "explicit visible",
			msg:  Message{Visibility: MessageVisibilityVisible},
			want: true,
		},
		{
			name: "hidden system hidden",
			msg:  Message{Visibility: MessageVisibilityHiddenSystem},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.msg.IsVisible())
		})
	}
}

func TestMessageHasThinking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  Message
		want bool
	}{
		{
			name: "visible thinking",
			msg:  Message{Thinking: "deep thought"},
			want: true,
		},
		{
			name: "hidden thinking only",
			msg:  Message{HasHiddenThinking: true},
			want: true,
		},
		{
			name: "no thinking",
			msg:  Message{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.msg.HasThinking())
		})
	}
}
