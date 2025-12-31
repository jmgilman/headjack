package multiplexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatSessionName(t *testing.T) {
	tests := []struct {
		name       string
		instanceID string
		sessionID  string
		expected   string
	}{
		{
			name:       "basic formatting",
			instanceID: "abc123",
			sessionID:  "sess456",
			expected:   "hjk-abc123-sess456",
		},
		{
			name:       "short IDs",
			instanceID: "a",
			sessionID:  "b",
			expected:   "hjk-a-b",
		},
		{
			name:       "longer IDs",
			instanceID: "instance-12345678",
			sessionID:  "happy-panda",
			expected:   "hjk-instance-12345678-happy-panda",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSessionName(tt.instanceID, tt.sessionID)
			assert.Equal(t, tt.expected, result)
		})
	}
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
	}

	for _, tc := range testCases {
		formatted := FormatSessionName(tc.instanceID, tc.sessionID)
		parsedInstance, parsedSession := ParseSessionName(formatted)

		assert.Equal(t, tc.instanceID, parsedInstance, "instanceID mismatch for %s", formatted)
		assert.Equal(t, tc.sessionID, parsedSession, "sessionID mismatch for %s", formatted)
	}
}

func TestSessionPrefix(t *testing.T) {
	assert.Equal(t, "hjk", SessionPrefix)
}
