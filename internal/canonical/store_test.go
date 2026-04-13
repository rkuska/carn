package canonical

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
	src "github.com/rkuska/carn/internal/source"
)

type stubSource struct {
	scanConversations []conversation
	sessions          map[string]sessionFull
	sessionLoads      map[string]sessionFull
}

type metadataAwareStubSource struct {
	stubSource
}

func (s stubSource) Provider() conversationProvider {
	return conversationProvider("claude")
}

func (s stubSource) Scan(context.Context, string) (src.ScanResult, error) {
	return src.ScanResult{Conversations: s.scanConversations}, nil
}

func (s stubSource) Load(_ context.Context, conversation conversation) (sessionFull, error) {
	return s.sessions[conversation.CacheKey()], nil
}

func (s stubSource) LoadSession(
	_ context.Context,
	_ conversation,
	meta sessionMeta,
) (sessionFull, error) {
	if session, ok := s.sessionLoads[meta.ID]; ok {
		return session, nil
	}
	return sessionFull{}, nil
}

func (metadataAwareStubSource) UsesScannedToolOutcomeCounts() bool {
	return true
}

func TestParseConversationsParallelBuildsTranscriptsAndSearchUnits(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	convValue := writeTestConversation(t, dir, "project-a", "session-1", "slug-1", []string{
		"first line",
		"second line",
	})
	source := stubSource{
		sessions: map[string]sessionFull{
			convValue.CacheKey(): {
				Meta: sessionMeta{ID: "session-1"},
				Messages: []message{
					{Role: role("assistant"), Text: "first line"},
					{Role: role("assistant"), Text: "second line"},
				},
			},
		},
	}

	transcripts, corpus, err := parseConversationsParallel(context.Background(), source, []conversation{convValue})
	require.NoError(t, err)
	require.Len(t, transcripts, 1)
	assert.Equal(
		t,
		buildSearchUnits(convValue.CacheKey(), transcripts[convValue.CacheKey()]),
		corpus.byConversation[convValue.CacheKey()],
	)
}

func TestBuildSearchUnitsIncludesLinkedSubagentMessages(t *testing.T) {
	t.Parallel()

	units := buildSearchUnits("codex:main", sessionFull{
		Messages: []message{
			{Role: role("assistant"), Text: "Implemented support for codex sessions."},
			{Role: role("user"), Text: "Planck is inspecting the parser.", IsAgentDivider: true},
			{Role: role("user"), Text: "Inspect the parser."},
			{Role: role("assistant"), Text: "Parser inspected."},
		},
	})

	texts := make([]string, 0, len(units))
	for _, unit := range units {
		texts = append(texts, unit.text)
	}

	assert.Contains(t, texts, "Inspect the parser.")
	assert.Contains(t, texts, "Parser inspected.")
}

func TestBuildSearchUnitsSkipsHiddenSystemMessages(t *testing.T) {
	t.Parallel()

	units := buildSearchUnits("codex:main", sessionFull{
		Messages: []message{
			{
				Role:       role("system"),
				Text:       "hidden system prompt",
				Visibility: messageVisibility("hidden_system"),
			},
			{
				Role: role("assistant"),
				Text: "visible response",
				ToolCalls: []toolCall{{
					Name:    "exec_command",
					Summary: "ran tests",
				}},
				Plans: []plan{{
					Content: "follow up tomorrow",
				}},
			},
		},
	})

	texts := make([]string, 0, len(units))
	for _, unit := range units {
		texts = append(texts, unit.text)
	}

	assert.NotContains(t, texts, "hidden system prompt")
	assert.Contains(t, texts, "visible response")
	assert.Contains(t, texts, "ran tests")
	assert.Contains(t, texts, "follow up tomorrow")
}

func TestStoreRebuildAllPersistsStreamingSearchAndPlans(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	convValue := conversation{
		Ref:     conversationRef{Provider: conversationProvider("claude"), ID: "session-1"},
		Name:    "demo",
		Project: project{DisplayName: "project-a"},
		Sessions: []sessionMeta{{
			ID:            "session-1",
			Slug:          "demo",
			Timestamp:     time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
			LastTimestamp: time.Date(2026, 3, 8, 10, 5, 0, 0, time.UTC),
			FilePath:      "/raw/session-1.jsonl",
			Project:       project{DisplayName: "project-a"},
		}},
	}
	source := stubSource{
		scanConversations: []conversation{convValue},
		sessions: map[string]sessionFull{
			convValue.CacheKey(): {
				Meta: convValue.Sessions[0],
				Messages: []message{
					{Role: role("assistant"), Text: "index this answer", Plans: []plan{{
						FilePath:  "plan.md",
						Content:   "finish the work",
						Timestamp: time.Date(2026, 3, 8, 10, 1, 0, 0, time.UTC),
					}}},
				},
			},
		},
	}
	store := New(nil, source)
	require.NoError(t, os.MkdirAll(src.ProviderRawDir(archiveDir, conversationProvider("claude")), 0o755))

	_, err := store.RebuildAll(context.Background(), archiveDir, nil)
	require.NoError(t, err)

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 1)
	assert.Equal(t, 1, conversations[0].PlanCount)

	session, err := store.Load(context.Background(), archiveDir, conversations[0])
	require.NoError(t, err)
	require.Len(t, session.Messages, 1)
	assert.Equal(t, "index this answer", session.Messages[0].Text)

	results, available, err := store.DeepSearch(context.Background(), archiveDir, "index this answer", conversations)
	require.NoError(t, err)
	assert.True(t, available)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].SearchPreview, "index this answer")
}

func TestStoreRebuildAllBackfillsStatsRowsWhenProjectionVersionIsStale(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	rawDir := src.ProviderRawDir(archiveDir, conversationProvider("claude"))
	convValue := writeTestConversation(t, rawDir, "project-a", "session-1", "slug-1", []string{
		"answer",
	})

	session := sessionFull{
		Meta: convValue.Sessions[0],
		Messages: []message{
			{
				Role:      role("user"),
				Text:      "question",
				Timestamp: convValue.Sessions[0].Timestamp,
			},
			{
				Role:      role("assistant"),
				Text:      "answer",
				Timestamp: convValue.Sessions[0].Timestamp.Add(time.Minute),
				Usage: conv.TokenUsage{
					InputTokens:  100,
					OutputTokens: 50,
				},
			},
		},
	}

	require.NoError(t, writeCanonicalStoreAtomically(
		context.Background(),
		archiveDir,
		[]conversation{convValue},
		map[string]sessionFull{
			convValue.CacheKey(): session,
		},
		searchCorpus{},
		nil,
		nil,
	))
	setSQLiteMetaValue(t, archiveDir, metaProjectionKey, strconv.Itoa(storeProjectionVersion-1))

	source := stubSource{
		scanConversations: []conversation{convValue},
		sessions: map[string]sessionFull{
			convValue.CacheKey(): session,
		},
		sessionLoads: map[string]sessionFull{
			"session-1": session,
		},
	}
	store := New(stubStatsCollector{
		rows: map[string]conv.SessionStatsData{
			"session-1": {
				PerformanceSequence: conv.PerformanceSequenceSession{
					Timestamp:         convValue.Sessions[0].Timestamp,
					Mutated:           true,
					FirstPassResolved: true,
					MutationCount:     1,
					ActionCount:       1,
				},
				TurnMetrics: conv.SessionTurnMetrics{
					Timestamp: convValue.Sessions[0].Timestamp,
					Turns: []conv.TurnTokens{{
						InputTokens: 100,
						TurnTokens:  150,
					}},
				},
			},
		},
	}, source)

	needsRebuild, err := store.NeedsRebuild(context.Background(), archiveDir)
	require.NoError(t, err)
	assert.True(t, needsRebuild)

	sequence, err := store.QueryPerformanceSequence(context.Background(), archiveDir, []string{convValue.CacheKey()})
	require.NoError(t, err)
	assert.Empty(t, sequence)

	turnMetrics, err := store.QueryTurnMetrics(context.Background(), archiveDir, []string{convValue.CacheKey()})
	require.NoError(t, err)
	assert.Empty(t, turnMetrics)

	_, err = store.RebuildAll(context.Background(), archiveDir, nil)
	require.NoError(t, err)

	sequence, err = store.QueryPerformanceSequence(context.Background(), archiveDir, []string{convValue.CacheKey()})
	require.NoError(t, err)
	require.Len(t, sequence, 1)
	assert.True(t, sequence[0].Mutated)
	assert.True(t, sequence[0].FirstPassResolved)

	turnMetrics, err = store.QueryTurnMetrics(context.Background(), archiveDir, []string{convValue.CacheKey()})
	require.NoError(t, err)
	require.Len(t, turnMetrics, 1)
	assert.Equal(t, []conv.TurnTokens{{InputTokens: 100, TurnTokens: 150}}, turnMetrics[0].Turns)
}

func TestStoreRebuildAllPersistsTranscriptDerivedToolOutcomesPerSession(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	require.NoError(t, os.MkdirAll(src.ProviderRawDir(archiveDir, conversationProvider("claude")), 0o755))

	convValue := conversation{
		Ref:     conversationRef{Provider: conversationProvider("claude"), ID: "thread-1"},
		Name:    "demo",
		Project: project{DisplayName: "project-a"},
		Sessions: []sessionMeta{
			{
				ID:              "session-1",
				Slug:            "demo",
				Timestamp:       time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
				LastTimestamp:   time.Date(2026, 3, 8, 10, 5, 0, 0, time.UTC),
				FilePath:        "/raw/session-1.jsonl",
				Project:         project{DisplayName: "project-a"},
				ToolCounts:      map[string]int{"Read": 99},
				ToolErrorCounts: map[string]int{"Read": 99},
			},
			{
				ID:              "session-2",
				Slug:            "follow-up",
				Timestamp:       time.Date(2026, 3, 8, 11, 0, 0, 0, time.UTC),
				LastTimestamp:   time.Date(2026, 3, 8, 11, 10, 0, 0, time.UTC),
				FilePath:        "/raw/session-2.jsonl",
				Project:         project{DisplayName: "project-a"},
				ToolCounts:      map[string]int{"Bash": 99},
				ToolErrorCounts: map[string]int{"Bash": 99},
			},
		},
	}

	source := stubSource{
		scanConversations: []conversation{convValue},
		sessions: map[string]sessionFull{
			convValue.CacheKey(): {
				Meta: convValue.Sessions[0],
				Messages: []message{
					{Role: role("assistant"), Text: "merged transcript"},
				},
			},
		},
		sessionLoads: map[string]sessionFull{
			"session-1": {
				Meta: convValue.Sessions[0],
				Messages: []message{
					{
						ToolCalls: []toolCall{{Name: "Read"}, {Name: "Read"}},
						ToolResults: []toolResult{{
							ToolName: "Read",
							IsError:  true,
							Content:  "file missing",
						}},
					},
				},
			},
			"session-2": {
				Meta: convValue.Sessions[1],
				Messages: []message{
					{
						ToolCalls: []toolCall{{Name: "Bash"}},
						ToolResults: []toolResult{{
							ToolName: "Bash",
							IsError:  true,
							Content:  "User rejected tool use",
						}},
					},
				},
			},
		},
	}

	store := New(nil, source)

	_, err := store.RebuildAll(context.Background(), archiveDir, nil)
	require.NoError(t, err)

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 1)
	require.Len(t, conversations[0].Sessions, 2)
	assert.Equal(t, map[string]int{"Read": 2}, conversations[0].Sessions[0].ToolCounts)
	assert.Equal(t, map[string]int{"Read": 1}, conversations[0].Sessions[0].ToolErrorCounts)
	assert.Nil(t, conversations[0].Sessions[0].ToolRejectCounts)
	assert.Equal(t, map[string]int{"Bash": 1}, conversations[0].Sessions[1].ToolCounts)
	assert.Nil(t, conversations[0].Sessions[1].ToolErrorCounts)
	assert.Equal(t, map[string]int{"Bash": 1}, conversations[0].Sessions[1].ToolRejectCounts)

	session, err := store.Load(context.Background(), archiveDir, conversations[0])
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"Read": 2}, session.Meta.ToolCounts)
	assert.Equal(t, map[string]int{"Read": 1}, session.Meta.ToolErrorCounts)
	assert.Nil(t, session.Meta.ToolRejectCounts)
}

func TestStoreRebuildAllUsesScannedToolOutcomesWhenSourceOptsIn(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	require.NoError(t, os.MkdirAll(src.ProviderRawDir(archiveDir, conversationProvider("claude")), 0o755))

	convValue := conversation{
		Ref:     conversationRef{Provider: conversationProvider("claude"), ID: "thread-1"},
		Name:    "demo",
		Project: project{DisplayName: "project-a"},
		Sessions: []sessionMeta{
			{
				ID:               "session-1",
				Slug:             "demo",
				Timestamp:        time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
				LastTimestamp:    time.Date(2026, 3, 8, 10, 5, 0, 0, time.UTC),
				FilePath:         "/raw/session-1.jsonl",
				Project:          project{DisplayName: "project-a"},
				ToolCounts:       map[string]int{"Read": 2},
				ToolErrorCounts:  map[string]int{"Read": 1},
				ToolRejectCounts: map[string]int{"Read": 1},
			},
		},
	}

	source := metadataAwareStubSource{
		stubSource: stubSource{
			scanConversations: []conversation{convValue},
			sessions: map[string]sessionFull{
				convValue.CacheKey(): {
					Meta: convValue.Sessions[0],
					Messages: []message{
						{Role: role("assistant"), Text: "merged transcript"},
					},
				},
			},
		},
	}

	store := New(nil, source)

	_, err := store.RebuildAll(context.Background(), archiveDir, nil)
	require.NoError(t, err)

	conversations, err := store.List(context.Background(), archiveDir)
	require.NoError(t, err)
	require.Len(t, conversations, 1)
	require.Len(t, conversations[0].Sessions, 1)
	assert.Equal(t, map[string]int{"Read": 2}, conversations[0].Sessions[0].ToolCounts)
	assert.Equal(t, map[string]int{"Read": 1}, conversations[0].Sessions[0].ToolErrorCounts)
	assert.Equal(t, map[string]int{"Read": 1}, conversations[0].Sessions[0].ToolRejectCounts)

	session, err := store.Load(context.Background(), archiveDir, conversations[0])
	require.NoError(t, err)
	assert.Equal(t, map[string]int{"Read": 2}, session.Meta.ToolCounts)
	assert.Equal(t, map[string]int{"Read": 1}, session.Meta.ToolErrorCounts)
	assert.Equal(t, map[string]int{"Read": 1}, session.Meta.ToolRejectCounts)
}

func writeTestConversation(
	t *testing.T,
	dir, projectName, sessionID, slug string,
	assistantTexts []string,
) conversation {
	t.Helper()

	path := filepath.Join(dir, projectName, sessionID+".jsonl")
	lines := []string{makeJSONLRecord("user", slug, sessionID)}
	timestamp := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)
	for i, text := range assistantTexts {
		lines = append(lines, testAssistantRecord(timestamp.Add(time.Duration(i)*time.Minute), text))
	}
	writeTestFile(t, path, strings.Join(lines, "\n"))

	proj := project{DisplayName: projectName}
	return conversation{
		Ref:     conversationRef{Provider: conversationProvider("claude"), ID: sessionID},
		Name:    slug,
		Project: proj,
		Sessions: []sessionMeta{{
			ID:        sessionID,
			Slug:      slug,
			Timestamp: timestamp,
			FilePath:  path,
			Project:   proj,
		}},
	}
}

func testAssistantRecord(ts time.Time, text string) string {
	return strings.Join([]string{
		`{"type":"assistant","timestamp":"`,
		ts.Format(time.RFC3339Nano),
		`","message":{"role":"assistant","model":"claude","content":[`,
		`{"type":"text","text":"`,
		text,
		`"}]}}`,
	}, "")
}

func makeJSONLRecord(role, slug, sessionID string) string {
	return strings.Join([]string{
		`{"type":"`, role,
		`","sessionId":"`, sessionID,
		`","slug":"`, slug,
		`","timestamp":"2026-03-08T10:00:00Z",`,
		`"cwd":"/tmp",`,
		`"message":{"role":"`, role, `","content":"hello"}}`,
	}, "")
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
