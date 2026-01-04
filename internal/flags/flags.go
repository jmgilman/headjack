// Package flags provides parsing, merging, and reconstruction of container runtime flags.
package flags

import (
	"errors"
	"fmt"
	"sort"
)

// Flags represents runtime flags as a key-value map.
// Values can be:
//   - string: generates --key=value
//   - bool: true generates --key, false omits the flag
//   - []string: generates --key=v for each element
type Flags map[string]any

// Sentinel errors for flag operations.
var (
	// ErrInvalidFlagValue is returned when a flag value has an unsupported type.
	ErrInvalidFlagValue = errors.New("invalid flag value type")
)

// FromConfig validates and normalizes config values into Flags.
// Accepts string, bool, []string, and []any (converted to []string).
func FromConfig(cfg map[string]any) (Flags, error) {
	if cfg == nil {
		return make(Flags), nil
	}

	result := make(Flags, len(cfg))
	for k, v := range cfg {
		switch val := v.(type) {
		case string:
			result[k] = val
		case bool:
			result[k] = val
		case []string:
			result[k] = val
		case []any:
			// Convert []any to []string (common from YAML parsing)
			strs := make([]string, 0, len(val))
			for _, item := range val {
				s, ok := item.(string)
				if !ok {
					return nil, fmt.Errorf("%w: %s array contains non-string value %T", ErrInvalidFlagValue, k, item)
				}
				strs = append(strs, s)
			}
			result[k] = strs
		default:
			return nil, fmt.Errorf("%w: %s has unsupported type %T", ErrInvalidFlagValue, k, v)
		}
	}
	return result, nil
}

// Merge combines two Flags maps with override taking precedence.
// Keys in override replace keys in base.
func Merge(base, override Flags) Flags {
	if base == nil && override == nil {
		return make(Flags)
	}
	if base == nil {
		return copyFlags(override)
	}
	if override == nil {
		return copyFlags(base)
	}

	result := make(Flags, len(base)+len(override))

	// Copy base
	for k, v := range base {
		result[k] = v
	}

	// Override with higher precedence values
	for k, v := range override {
		result[k] = v
	}

	return result
}

// copyFlags creates a shallow copy of Flags.
func copyFlags(f Flags) Flags {
	result := make(Flags, len(f))
	for k, v := range f {
		result[k] = v
	}
	return result
}

// ToArgs reconstructs Flags into CLI arguments.
// Output is sorted by key for deterministic ordering.
//
// Conversion rules:
//   - string: "--key=value"
//   - bool true: "--key"
//   - bool false: (omitted)
//   - []string: "--key=v1", "--key=v2", ...
func ToArgs(f Flags) []string {
	if len(f) == 0 {
		return nil
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(f))
	for k := range f {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var args []string
	for _, k := range keys {
		v := f[k]
		switch val := v.(type) {
		case string:
			args = append(args, fmt.Sprintf("--%s=%s", k, val))
		case bool:
			if val {
				args = append(args, "--"+k)
			}
			// false: omit entirely
		case []string:
			for _, s := range val {
				args = append(args, fmt.Sprintf("--%s=%s", k, s))
			}
		}
	}
	return args
}
