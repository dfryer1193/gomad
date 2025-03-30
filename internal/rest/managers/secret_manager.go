package managers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/dfryer1193/gomad/internal/data/repository"
	"github.com/dfryer1193/gomad/internal/data/repository/postgres"
)

type SecretManager interface {
	SaveSecret(repoName string) (string, error)
	GetSecret(repoName string) (string, error)
	Close()
}

type secretManager struct {
	repo repository.SecretRepository
}

var (
	secretMgr  *secretManager
	secretOnce sync.Once
)

func GetSecretManager() SecretManager {
	secretOnce.Do(func() {
		secretMgr = &secretManager{
			repo: postgres.GetSecretsRepository(),
		}
	})

	return secretMgr
}

func (s *secretManager) SaveSecret(repoName string) (string, error) {
	secret, err := generateRandomSecret()
	if err != nil {
		return "", fmt.Errorf("failed to generate secret: %w", err)
	}
	return s.repo.InsertSecret(repoName, secret)
}

func (s *secretManager) GetSecret(repoName string) (string, error) {
	return s.repo.GetSecret(repoName)
}

func (s *secretManager) Close() {
	s.repo.Close()
}

func generateRandomSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	secret := hex.EncodeToString(bytes)

	return secret, nil
}
