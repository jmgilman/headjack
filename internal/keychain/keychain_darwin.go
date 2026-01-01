//go:build darwin

package keychain

import gokeychain "github.com/keybase/go-keychain"

// serviceName is the service identifier used for all headjack credentials.
const serviceName = "com.headjack.cli"

type keychain struct{}

// New creates a new Keychain backed by macOS Keychain.
func New() Keychain {
	return &keychain{}
}

func (k *keychain) Set(account, secret string) error {
	// First try to delete any existing entry to avoid duplicates.
	// Ignore errors since the item may not exist.
	_ = k.Delete(account) //nolint:errcheck // Intentionally ignoring - item may not exist

	item := gokeychain.NewItem()
	item.SetSecClass(gokeychain.SecClassGenericPassword)
	item.SetService(serviceName)
	item.SetAccount(account)
	item.SetLabel("Headjack - " + account)
	item.SetData([]byte(secret))
	item.SetSynchronizable(gokeychain.SynchronizableNo)
	item.SetAccessible(gokeychain.AccessibleWhenUnlocked)

	return gokeychain.AddItem(item)
}

func (k *keychain) Get(account string) (string, error) {
	query := gokeychain.NewItem()
	query.SetSecClass(gokeychain.SecClassGenericPassword)
	query.SetService(serviceName)
	query.SetAccount(account)
	query.SetMatchLimit(gokeychain.MatchLimitOne)
	query.SetReturnData(true)

	results, err := gokeychain.QueryItem(query)
	if err == gokeychain.ErrorItemNotFound {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", ErrNotFound
	}

	return string(results[0].Data), nil
}

func (k *keychain) Delete(account string) error {
	item := gokeychain.NewItem()
	item.SetSecClass(gokeychain.SecClassGenericPassword)
	item.SetService(serviceName)
	item.SetAccount(account)

	err := gokeychain.DeleteItem(item)
	if err == gokeychain.ErrorItemNotFound {
		return nil
	}
	return err
}
