package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestExtractExitPlanResult(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 3, 8, 18, 55, 39, 0, time.UTC)

	tests := []struct {
		name        string
		raw         json.RawMessage
		wantOK      bool
		wantPath    string
		wantContent string
	}{
		{
			name: "accepted plan",
			raw: json.RawMessage(
				`{"plan":"# My Plan\n\nStep 1",` +
					`"filePath":"/Users/x/.claude/plans/my-slug.md",` +
					`"isAgent":false}`),
			wantOK:      true,
			wantPath:    "/Users/x/.claude/plans/my-slug.md",
			wantContent: "# My Plan\n\nStep 1",
		},
		{
			name:   "rejected plan is plain string",
			raw:    json.RawMessage(`"Error: The user doesn't want to proceed with this tool use."`),
			wantOK: false,
		},
		{
			name:   "rejected plan user rejected",
			raw:    json.RawMessage(`"User rejected tool use"`),
			wantOK: false,
		},
		{
			name:   "empty input",
			raw:    json.RawMessage(``),
			wantOK: false,
		},
		{
			name:   "null input",
			raw:    json.RawMessage(`null`),
			wantOK: false,
		},
		{
			name:   "malformed JSON",
			raw:    json.RawMessage(`{bad json`),
			wantOK: false,
		},
		{
			name:   "missing filePath",
			raw:    json.RawMessage(`{"plan":"# My Plan"}`),
			wantOK: false,
		},
		{
			name:   "missing plan content",
			raw:    json.RawMessage(`{"filePath":"/Users/x/.claude/plans/my-slug.md"}`),
			wantOK: false,
		},
		{
			name:   "empty plan content",
			raw:    json.RawMessage(`{"plan":"","filePath":"/Users/x/.claude/plans/my-slug.md"}`),
			wantOK: false,
		},
		{
			name:   "empty filePath",
			raw:    json.RawMessage(`{"plan":"# My Plan","filePath":""}`),
			wantOK: false,
		},
		{
			name:   "non-plan toolUseResult like Edit",
			raw:    json.RawMessage(`{"type":"update","filePath":"/tmp/file.go","content":"package main","structuredPatch":[]}`),
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := extractExitPlanResult(tt.raw, ts)
			if ok != tt.wantOK {
				t.Fatalf("extractExitPlanResult() ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got.filePath != tt.wantPath {
				t.Errorf("filePath = %q, want %q", got.filePath, tt.wantPath)
			}
			if got.content != tt.wantContent {
				t.Errorf("content = %q, want %q", got.content, tt.wantContent)
			}
			if !got.timestamp.Equal(ts) {
				t.Errorf("timestamp = %v, want %v", got.timestamp, ts)
			}
		})
	}
}

func TestWritePlanReadPlan(t *testing.T) {
	t.Parallel()

	ts := time.Date(2025, 3, 15, 14, 30, 0, 0, time.UTC)
	original := plan{
		filePath:  "/home/user/.claude/plans/test-plan.md",
		content:   "# Test Plan\n\n## Step 1\nDo something\n\n## Step 2\nDo something else",
		timestamp: ts,
	}

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if err := writePlan(w, original); err != nil {
		t.Fatalf("writePlan: %v", err)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}

	r := bufio.NewReader(&buf)
	got, err := readPlan(r)
	if err != nil {
		t.Fatalf("readPlan: %v", err)
	}

	if got.filePath != original.filePath {
		t.Errorf("filePath = %q, want %q", got.filePath, original.filePath)
	}
	if got.content != original.content {
		t.Errorf("content = %q, want %q", got.content, original.content)
	}
	if !got.timestamp.Equal(original.timestamp) {
		t.Errorf("timestamp = %v, want %v", got.timestamp, original.timestamp)
	}
}

func TestWritePlanReadPlanZeroTimestamp(t *testing.T) {
	t.Parallel()

	original := plan{
		filePath: "/home/user/.claude/plans/empty-ts.md",
		content:  "# Plan",
	}

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	if err := writePlan(w, original); err != nil {
		t.Fatalf("writePlan: %v", err)
	}
	if err := w.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}

	r := bufio.NewReader(&buf)
	got, err := readPlan(r)
	if err != nil {
		t.Fatalf("readPlan: %v", err)
	}

	if !got.timestamp.IsZero() {
		t.Errorf("timestamp = %v, want zero", got.timestamp)
	}
}

func TestCountPlansInMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		messages []message
		want     int
	}{
		{
			name:     "no messages",
			messages: nil,
			want:     0,
		},
		{
			name: "no plans",
			messages: []message{
				{role: roleUser, text: "hello"},
			},
			want: 0,
		},
		{
			name: "one plan on user message",
			messages: []message{
				{role: roleUser, plans: []plan{{filePath: "a.md", content: "plan a"}}},
			},
			want: 1,
		},
		{
			name: "multiple plans across messages",
			messages: []message{
				{role: roleUser, plans: []plan{{filePath: "a.md", content: "plan a"}}},
				{role: roleAssistant, text: "no plans here"},
				{role: roleUser, plans: []plan{
					{filePath: "b.md", content: "plan b"},
					{filePath: "c.md", content: "plan c"},
				}},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := countPlansInMessages(tt.messages); got != tt.want {
				t.Errorf("countPlansInMessages() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDeduplicatePlans(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		messages []parsedMessage
		// wantPlans[i] is the expected plan count on messages[i]
		wantPlans []int
	}{
		{
			name:      "no messages",
			messages:  nil,
			wantPlans: nil,
		},
		{
			name: "no plans",
			messages: []parsedMessage{
				{role: roleUser, text: "hello"},
			},
			wantPlans: []int{0},
		},
		{
			name: "single plan preserved",
			messages: []parsedMessage{
				{role: roleUser, plans: []plan{{filePath: "a.md", content: "v1"}}},
			},
			wantPlans: []int{1},
		},
		{
			name: "same filePath twice keeps last",
			messages: []parsedMessage{
				{role: roleUser, plans: []plan{{filePath: "a.md", content: "v1"}}},
				{role: roleAssistant, text: "reworking"},
				{role: roleUser, plans: []plan{{filePath: "a.md", content: "v2"}}},
			},
			wantPlans: []int{0, 0, 1},
		},
		{
			name: "different filePaths both kept",
			messages: []parsedMessage{
				{role: roleUser, plans: []plan{{filePath: "a.md", content: "plan a"}}},
				{role: roleUser, plans: []plan{{filePath: "b.md", content: "plan b"}}},
			},
			wantPlans: []int{1, 1},
		},
		{
			name: "three iterations keeps only last",
			messages: []parsedMessage{
				{role: roleUser, plans: []plan{{filePath: "a.md", content: "v1"}}},
				{role: roleUser, plans: []plan{{filePath: "a.md", content: "v2"}}},
				{role: roleUser, plans: []plan{{filePath: "a.md", content: "v3"}}},
			},
			wantPlans: []int{0, 0, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			deduplicatePlans(tt.messages)
			for i, msg := range tt.messages {
				if len(msg.plans) != tt.wantPlans[i] {
					t.Errorf("messages[%d] plans = %d, want %d", i, len(msg.plans), tt.wantPlans[i])
				}
			}
		})
	}
}

func TestDeduplicatePlansPreservesLastContent(t *testing.T) {
	t.Parallel()

	messages := []parsedMessage{
		{role: roleUser, plans: []plan{{filePath: "a.md", content: "v1"}}},
		{role: roleUser, plans: []plan{{filePath: "a.md", content: "v2"}}},
	}
	deduplicatePlans(messages)

	if messages[1].plans[0].content != "v2" {
		t.Errorf("content = %q, want %q", messages[1].plans[0].content, "v2")
	}
}

func TestLastPlan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		messages    []message
		wantOK      bool
		wantContent string
	}{
		{
			name:     "no messages",
			messages: nil,
			wantOK:   false,
		},
		{
			name:     "no plans",
			messages: []message{{role: roleUser, text: "hello"}},
			wantOK:   false,
		},
		{
			name: "single plan",
			messages: []message{
				{role: roleUser, plans: []plan{{filePath: "a.md", content: "the plan"}}},
			},
			wantOK:      true,
			wantContent: "the plan",
		},
		{
			name: "returns last chronologically",
			messages: []message{
				{role: roleUser, plans: []plan{{filePath: "a.md", content: "old"}}},
				{role: roleUser, plans: []plan{{filePath: "b.md", content: "new"}}},
			},
			wantOK:      true,
			wantContent: "new",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := lastPlan(tt.messages)
			if ok != tt.wantOK {
				t.Fatalf("lastPlan() ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got.content != tt.wantContent {
				t.Errorf("content = %q, want %q", got.content, tt.wantContent)
			}
		})
	}
}

func TestFormatPlan(t *testing.T) {
	t.Parallel()

	p := plan{
		filePath: "/home/user/.claude/plans/extract-plans.md",
		content:  "# Extract Plans\n\n## Step 1\nAdd plan type",
	}

	got := formatPlan(p)
	want := "Plan: extract-plans.md\n\n# Extract Plans\n\n## Step 1\nAdd plan type"
	if got != want {
		t.Errorf("formatPlan() = %q, want %q", got, want)
	}
}
