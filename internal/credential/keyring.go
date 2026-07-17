package credential

import (
	"errors"

	keyring "github.com/zalando/go-keyring"
)

type keyringBackend struct{}

func (keyringBackend) Set(service, account, secret string) error {
	return keyring.Set(service, account, secret)
}

func (keyringBackend) Get(service, account string) (string, error) {
	secret, err := keyring.Get(service, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrSecretNotFound
	}
	return secret, err
}

func (keyringBackend) Delete(service, account string) error {
	err := keyring.Delete(service, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrSecretNotFound
	}
	return err
}
