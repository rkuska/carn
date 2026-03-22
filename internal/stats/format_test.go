package stats

import "testing"

func TestFormatNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input int
		want  string
	}{
		{name: "raw", input: 999, want: "999"},
		{name: "comma lower bound", input: 1000, want: "1,000"},
		{name: "comma upper bound", input: 99999, want: "99,999"},
		{name: "k whole", input: 100000, want: "100k"},
		{name: "k fractional", input: 328500, want: "328.5k"},
		{name: "m whole", input: 1000000, want: "1M"},
		{name: "m fractional", input: 8200000, want: "8.2M"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := FormatNumber(testCase.input); got != testCase.want {
				t.Fatalf("FormatNumber(%d) = %q, want %q", testCase.input, got, testCase.want)
			}
		})
	}
}
