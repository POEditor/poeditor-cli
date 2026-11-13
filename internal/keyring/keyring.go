// Package keyring stores API tokens in the OS credential store (macOS Keychain,
// Windows Credential Manager, Secret Service on Linux), one entry per account.
package keyring

import (
	"errors"
	"sync"

	gokeyring "github.com/zalando/go-keyring"
)

// Service is the keyring service name every token is stored under.
const Service = "poeditor-cli"

// ErrNotFound means the account has no token in the keyring.
var ErrNotFound = errors.New("token not found in keyring")

// Keyring is where tokens live. Any error other than ErrNotFound from Set means
// the keyring is unusable (e.g. no Secret Service on a headless Linux box).
type Keyring interface {
	Get(account string) (string, error)
	Set(account, token string) error
	Delete(account string) error
}

// System returns the OS keyring.
func System() Keyring { return systemKeyring{} }

type systemKeyring struct{}

func (systemKeyring) Get(account string) (string, error) {
	token, err := gokeyring.Get(Service, account)
	if errors.Is(err, gokeyring.ErrNotFound) {
		return "", ErrNotFound
	}
	return token, err
}

func (systemKeyring) Set(account, token string) error {
	return gokeyring.Set(Service, account, token)
}

func (systemKeyring) Delete(account string) error {
	err := gokeyring.Delete(Service, account)
	if errors.Is(err, gokeyring.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

// Memory is an in-process keyring for tests. Unavailable makes every call fail
// the way a missing OS keyring does.
type Memory struct {
	Unavailable bool

	mu     sync.Mutex
	tokens map[string]string
}

// ErrUnavailable is what Memory returns when Unavailable is set.
var ErrUnavailable = errors.New("keyring unavailable")

func (m *Memory) Get(account string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Unavailable {
		return "", ErrUnavailable
	}
	token, ok := m.tokens[account]
	if !ok {
		return "", ErrNotFound
	}
	return token, nil
}

func (m *Memory) Set(account, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Unavailable {
		return ErrUnavailable
	}
	if m.tokens == nil {
		m.tokens = map[string]string{}
	}
	m.tokens[account] = token
	return nil
}

func (m *Memory) Delete(account string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Unavailable {
		return ErrUnavailable
	}
	if _, ok := m.tokens[account]; !ok {
		return ErrNotFound
	}
	delete(m.tokens, account)
	return nil
}
