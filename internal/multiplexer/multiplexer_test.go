package multiplexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatSessionName(t *testing.T) {
	t.Run("basic formatting", func(t *testing.T) {
		result, err := FormatSessionName("abc123", "sess456")
		require.NoError(t, err)
		assert.Equal(t, "hjk-abc123-sess456", result)
	})

	t.Run("short IDs", func(t *testing.T) {
		result, err := FormatSessionName("a", "b")
		require.NoError(t, err)
		assert.Equal(t, "hjk-a-b", result)
	})

	t.Run("session ID with hyphens allowed", func(t *testing.T) {
		result, err := FormatSessionName("instance", "happy-panda")
		require.NoError(t, err)
		assert.Equal(t, "hjk-instance-happy-panda", result)
	})

	t.Run("rejects instance ID with hyphens", func(t *testing.T) {
		_, err := FormatSessionName("instance-123", "session")
		assert.ErrorIs(t, err, ErrInvalidInstanceID)
	})

	t.Run("rejects instance ID with multiple hyphens", func(t *testing.T) {
		_, err := FormatSessionName("my-instance-id", "session")
		assert.ErrorIs(t, err, ErrInvalidInstanceID)
	})
}

func TestParseSessionName(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		expectedInstanceID string
		expectedSessionID  string
	}{
		{
			name:               "valid format",
			input:              "hjk-abc123-sess456",
			expectedInstanceID: "abc123",
			expectedSessionID:  "sess456",
		},
		{
			name:               "short IDs",
			input:              "hjk-a-b",
			expectedInstanceID: "a",
			expectedSessionID:  "b",
		},
		{
			name:               "session ID with hyphens",
			input:              "hjk-instance-happy-panda",
			expectedInstanceID: "instance",
			expectedSessionID:  "happy-panda",
		},
		{
			name:               "invalid - no prefix",
			input:              "abc-123-456",
			expectedInstanceID: "",
			expectedSessionID:  "",
		},
		{
			name:               "invalid - wrong prefix",
			input:              "xyz-abc-123",
			expectedInstanceID: "",
			expectedSessionID:  "",
		},
		{
			name:               "invalid - too short",
			input:              "hjk-ab",
			expectedInstanceID: "",
			expectedSessionID:  "",
		},
		{
			name:               "invalid - empty",
			input:              "",
			expectedInstanceID: "",
			expectedSessionID:  "",
		},
		{
			name:               "invalid - only prefix",
			input:              "hjk-",
			expectedInstanceID: "",
			expectedSessionID:  "",
		},
		{
			name:               "invalid - missing session ID",
			input:              "hjk-instance",
			expectedInstanceID: "",
			expectedSessionID:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instanceID, sessionID := ParseSessionName(tt.input)
			assert.Equal(t, tt.expectedInstanceID, instanceID)
			assert.Equal(t, tt.expectedSessionID, sessionID)
		})
	}
}

func TestFormatAndParseRoundTrip(t *testing.T) {
	testCases := []struct {
		instanceID string
		sessionID  string
	}{
		{"abc123", "sess456"},
		{"a", "b"},
		{"instance", "session"},
		{"inst", "happy-panda"},
	}

	for _, tc := range testCases {
		formatted, err := FormatSessionName(tc.instanceID, tc.sessionID)
		require.NoError(t, err)

		parsedInstance, parsedSession := ParseSessionName(formatted)

		assert.Equal(t, tc.instanceID, parsedInstance, "instanceID mismatch for %s", formatted)
		assert.Equal(t, tc.sessionID, parsedSession, "sessionID mismatch for %s", formatted)
	}
}

func TestSessionPrefix(t *testing.T) {
	assert.Equal(t, "hjk", SessionPrefix)
}
