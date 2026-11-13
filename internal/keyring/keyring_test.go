package keyring_test

import (
	"errors"
	"testing"

	"github.com/POEditor/poeditor-cli/internal/keyring"
)

func TestMemory(t *testing.T) {
	var kr keyring.Keyring = &keyring.Memory{}

	if _, err := kr.Get("a"); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("empty Get: %v", err)
	}
	if err := kr.Set("a", "tok"); err != nil {
		t.Fatal(err)
	}
	if got, err := kr.Get("a"); err != nil || got != "tok" {
		t.Fatalf("Get = %q, %v", got, err)
	}
	if err := kr.Delete("a"); err != nil {
		t.Fatal(err)
	}
	if err := kr.Delete("a"); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("second Delete: %v", err)
	}
}

func TestMemory_unavailable(t *testing.T) {
	kr := &keyring.Memory{Unavailable: true}

	if err := kr.Set("a", "tok"); !errors.Is(err, keyring.ErrUnavailable) {
		t.Fatalf("Set: %v", err)
	}
	if _, err := kr.Get("a"); !errors.Is(err, keyring.ErrUnavailable) {
		t.Fatalf("Get: %v", err)
	}
}
