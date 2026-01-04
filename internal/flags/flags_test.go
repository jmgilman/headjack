package flags

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromConfig(t *testing.T) {
	t.Run("nil input returns empty flags", func(t *testing.T) {
		result, err := FromConfig(nil)

		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("string values", func(t *testing.T) {
		input := map[string]any{
			"memory":  "2g",
			"systemd": "always",
		}

		result, err := FromConfig(input)

		require.NoError(t, err)
		assert.Equal(t, "2g", result["memory"])
		assert.Equal(t, "always", result["systemd"])
	})

	t.Run("bool values", func(t *testing.T) {
		input := map[string]any{
			"privileged": true,
			"debug":      false,
		}

		result, err := FromConfig(input)

		require.NoError(t, err)
		assert.Equal(t, true, result["privileged"])
		assert.Equal(t, false, result["debug"])
	})

	t.Run("string slice values", func(t *testing.T) {
		input := map[string]any{
			"volume": []string{"/path1:/cont1", "/path2:/cont2"},
		}

		result, err := FromConfig(input)

		require.NoError(t, err)
		assert.Equal(t, []string{"/path1:/cont1", "/path2:/cont2"}, result["volume"])
	})

	t.Run("any slice converted to string slice", func(t *testing.T) {
		input := map[string]any{
			"volume": []any{"/path1:/cont1", "/path2:/cont2"},
		}

		result, err := FromConfig(input)

		require.NoError(t, err)
		assert.Equal(t, []string{"/path1:/cont1", "/path2:/cont2"}, result["volume"])
	})

	t.Run("mixed types", func(t *testing.T) {
		input := map[string]any{
			"memory":     "2g",
			"privileged": true,
			"volume":     []string{"/a:/b"},
		}

		result, err := FromConfig(input)

		require.NoError(t, err)
		assert.Equal(t, "2g", result["memory"])
		assert.Equal(t, true, result["privileged"])
		assert.Equal(t, []string{"/a:/b"}, result["volume"])
	})

	t.Run("error on unsupported type", func(t *testing.T) {
		input := map[string]any{
			"invalid": 123, // int not supported
		}

		_, err := FromConfig(input)

		require.ErrorIs(t, err, ErrInvalidFlagValue)
		assert.Contains(t, err.Error(), "invalid")
	})

	t.Run("error on non-string in array", func(t *testing.T) {
		input := map[string]any{
			"volume": []any{"/path", 123}, // int in array
		}

		_, err := FromConfig(input)

		require.ErrorIs(t, err, ErrInvalidFlagValue)
		assert.Contains(t, err.Error(), "volume")
	})
}

func TestMerge(t *testing.T) {
	t.Run("nil inputs return empty flags", func(t *testing.T) {
		result := Merge(nil, nil)

		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("nil base returns copy of override", func(t *testing.T) {
		override := Flags{"memory": "2g"}

		result := Merge(nil, override)

		assert.Equal(t, "2g", result["memory"])
		// Verify it's a copy
		override["memory"] = "4g"
		assert.Equal(t, "2g", result["memory"])
	})

	t.Run("nil override returns copy of base", func(t *testing.T) {
		base := Flags{"memory": "2g"}

		result := Merge(base, nil)

		assert.Equal(t, "2g", result["memory"])
		// Verify it's a copy
		base["memory"] = "4g"
		assert.Equal(t, "2g", result["memory"])
	})

	t.Run("override takes precedence for same key", func(t *testing.T) {
		base := Flags{"memory": "1g", "cpus": "2"}
		override := Flags{"memory": "4g"}

		result := Merge(base, override)

		assert.Equal(t, "4g", result["memory"])
		assert.Equal(t, "2", result["cpus"])
	})

	t.Run("combines keys from both", func(t *testing.T) {
		base := Flags{"memory": "2g"}
		override := Flags{"cpus": "4"}

		result := Merge(base, override)

		assert.Equal(t, "2g", result["memory"])
		assert.Equal(t, "4", result["cpus"])
	})

	t.Run("override bool replaces base string", func(t *testing.T) {
		base := Flags{"flag": "value"}
		override := Flags{"flag": true}

		result := Merge(base, override)

		assert.Equal(t, true, result["flag"])
	})

	t.Run("override array replaces base string", func(t *testing.T) {
		base := Flags{"volume": "/a:/b"}
		override := Flags{"volume": []string{"/c:/d", "/e:/f"}}

		result := Merge(base, override)

		assert.Equal(t, []string{"/c:/d", "/e:/f"}, result["volume"])
	})
}

func TestToArgs(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result := ToArgs(nil)

		assert.Nil(t, result)
	})

	t.Run("empty input returns nil", func(t *testing.T) {
		result := ToArgs(Flags{})

		assert.Nil(t, result)
	})

	t.Run("string value", func(t *testing.T) {
		result := ToArgs(Flags{"memory": "2g"})

		assert.Equal(t, []string{"--memory=2g"}, result)
	})

	t.Run("bool true", func(t *testing.T) {
		result := ToArgs(Flags{"privileged": true})

		assert.Equal(t, []string{"--privileged"}, result)
	})

	t.Run("bool false omitted", func(t *testing.T) {
		result := ToArgs(Flags{"debug": false})

		assert.Empty(t, result)
	})

	t.Run("string array", func(t *testing.T) {
		result := ToArgs(Flags{"volume": []string{"/a:/b", "/c:/d"}})

		assert.Equal(t, []string{"--volume=/a:/b", "--volume=/c:/d"}, result)
	})

	t.Run("mixed types sorted by key", func(t *testing.T) {
		result := ToArgs(Flags{
			"privileged": true,
			"memory":     "2g",
			"cpus":       "4",
		})

		// Should be sorted: cpus, memory, privileged
		assert.Equal(t, []string{"--cpus=4", "--memory=2g", "--privileged"}, result)
	})

	t.Run("complex example", func(t *testing.T) {
		result := ToArgs(Flags{
			"systemd":    "always",
			"privileged": true,
			"debug":      false, // omitted
			"volume":     []string{"/a:/b", "/c:/d"},
		})

		// Sorted: privileged, systemd, volume (debug omitted)
		assert.Equal(t, []string{
			"--privileged",
			"--systemd=always",
			"--volume=/a:/b",
			"--volume=/c:/d",
		}, result)
	})

	t.Run("value with equals sign", func(t *testing.T) {
		result := ToArgs(Flags{"env": "FOO=bar"})

		assert.Equal(t, []string{"--env=FOO=bar"}, result)
	})
}

func TestRoundTrip(t *testing.T) {
	t.Run("config to flags to args", func(t *testing.T) {
		cfg := map[string]any{
			"memory":     "2g",
			"privileged": true,
			"volume":     []string{"/a:/b"},
		}

		flags, err := FromConfig(cfg)
		require.NoError(t, err)

		args := ToArgs(flags)

		assert.Contains(t, args, "--memory=2g")
		assert.Contains(t, args, "--privileged")
		assert.Contains(t, args, "--volume=/a:/b")
	})

	t.Run("merge then to args", func(t *testing.T) {
		baseFlags, err := FromConfig(map[string]any{"systemd": "always", "memory": "1g"})
		require.NoError(t, err)
		configFlags, err := FromConfig(map[string]any{"memory": "4g", "privileged": true})
		require.NoError(t, err)

		merged := Merge(baseFlags, configFlags)
		args := ToArgs(merged)

		// Config should win for memory
		assert.Contains(t, args, "--memory=4g")
		assert.NotContains(t, args, "--memory=1g")
		// Both sources contribute
		assert.Contains(t, args, "--systemd=always")
		assert.Contains(t, args, "--privileged")
	})
}
