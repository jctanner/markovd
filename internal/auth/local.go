package auth

import (
	"context"
	"fmt"

	"github.com/jctanner/markovd/internal/db"
	"github.com/jctanner/markovd/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type LocalProvider struct {
	db *db.DB
}

func NewLocalProvider(database *db.DB) *LocalProvider {
	return &LocalProvider{db: database}
}

func (p *LocalProvider) Authenticate(username, password string) (*models.User, error) {
	user, err := p.db.GetUserByUsername(context.Background(), username)
	if err != nil {
		return nil, fmt.Errorf("looking up user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	return user, nil
}

func (p *LocalProvider) CreateUser(username, password string) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}
	return p.db.CreateUser(context.Background(), username, string(hash))
}
