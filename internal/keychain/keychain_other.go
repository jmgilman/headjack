//go:build !darwin

package keychain

type keychain struct{}

// New creates a new Keychain. On non-macOS platforms, all operations return ErrUnsupportedPlatform.
func New() Keychain {
	return &keychain{}
}

func (k *keychain) Set(_, _ string) error {
	return ErrUnsupportedPlatform
}

func (k *keychain) Get(_ string) (string, error) {
	return "", ErrUnsupportedPlatform
}

func (k *keychain) Delete(_ string) error {
	return ErrUnsupportedPlatform
}
