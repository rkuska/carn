package canonical

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conv "github.com/rkuska/carn/internal/conversation"
)

func TestMarshalUnmarshalTurnTokensRoundTripExtendedFields(t *testing.T) {
	t.Parallel()

	in := []conv.TurnTokens{
		{
			PromptTokens:        200,
			TurnTokens:          250,
			CacheReadTokens:     5_000,
			CacheCreationTokens: 1_200,
			MemoryWriteCount:    2,
		},
		{
			PromptTokens: 100,
			TurnTokens:   120,
		},
	}

	raw, err := marshalTurnTokens(in)
	require.NoError(t, err)

	out, err := unmarshalTurnTokens(raw)
	require.NoError(t, err)
	require.Len(t, out, 2)

	assert.Equal(t, in[0], out[0])
	assert.Equal(t, in[1], out[1])

	// legacy rows that lack new fields still decode with zero values for the
	// extensions, so existing turns_json payloads stay readable across the
	// schema bump.
	legacyRaw := `[{"prompt_tokens":50,"turn_tokens":60}]`
	legacy, err := unmarshalTurnTokens(legacyRaw)
	require.NoError(t, err)
	require.Len(t, legacy, 1)
	assert.Equal(t, 50, legacy[0].PromptTokens)
	assert.Equal(t, 60, legacy[0].TurnTokens)
	assert.Zero(t, legacy[0].CacheReadTokens)
	assert.Zero(t, legacy[0].CacheCreationTokens)
	assert.Zero(t, legacy[0].MemoryWriteCount)
}
