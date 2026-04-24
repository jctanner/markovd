package auth

import "github.com/jctanner/markovd/internal/models"

type Provider interface {
	Authenticate(username, password string) (*models.User, error)
	CreateUser(username, password string) (*models.User, error)
}
