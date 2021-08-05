package secretloader

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

type SecretLoader interface {
	Get() (bool, []byte, error)
}

type FileSecretLoader struct {
	secretPath string
	secret     []byte
	secretMu   sync.Mutex

	expiryTime   time.Time
	expiryPeriod time.Duration
}

func NewSecretLoader(secretPath string, expiryPeriod time.Duration) *FileSecretLoader {
	return &FileSecretLoader{
		secretPath:   secretPath,
		expiryTime:   time.Now(),
		expiryPeriod: expiryPeriod,
	}
}

func (sl *FileSecretLoader) Get() (bool, []byte, error) {
	sl.secretMu.Lock()
	defer sl.secretMu.Unlock()

	secretBefore := sl.secret

	if time.Now().After(sl.expiryTime) {
		if err := sl.loadSecret(); err != nil {
			return false, nil, err
		}
	}

	return !bytes.Equal(secretBefore, sl.secret), sl.secret, nil
}

func (sl *FileSecretLoader) loadSecret() error {
	secret, err := os.ReadFile(sl.secretPath)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to stat secret: %w", err)
	} else if err != nil {
		return err
	}

	sl.expiryTime = time.Now().Add(sl.expiryPeriod)
	sl.secret = secret

	return nil
}
