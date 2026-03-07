package app

import "testing"

func TestIsSystemInterrupt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "tool use interrupt",
			text: "[Request interrupted by user for tool use]",
			want: true,
		},
		{
			name: "plain interrupt",
			text: "[Request interrupted by user]",
			want: true,
		},
		{
			name: "normal user text",
			text: "Please help me with this code",
			want: false,
		},
		{
			name: "empty string",
			text: "",
			want: false,
		},
		{
			name: "partial match",
			text: "[Request interrupted",
			want: false,
		},
		{
			name: "substring of interrupt",
			text: "Request interrupted by user for tool use",
			want: false,
		},
		{
			name: "local command caveat",
			text: "<local-command-caveat>Caveat: The messages below were generated" +
				" by the user while running local commands.</local-command-caveat>",
			want: true,
		},
		{
			name: "command name",
			text: "<command-name>/compact</command-name>\n            <command-message>compact</command-message>",
			want: true,
		},
		{
			name: "local command stdout",
			text: "<local-command-stdout>Compacted</local-command-stdout>",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isSystemInterrupt(tt.text)
			if got != tt.want {
				t.Errorf("isSystemInterrupt(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}
