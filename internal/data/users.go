package data

import (
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name" validate:"required,max=500"`
	Email     string    `json:"email" validate:"required,email"`
	Password  password  `json:"-" validate:"required"`
	Activated bool      `json:"activated"`
	Version   int       `json:"-"`
}

func DatabaseUserToUser(dbUser database.User) User {
	return User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Name:      dbUser.Name,
		Email:     dbUser.Email,
		Password: password{
			hash: dbUser.PasswordHash,
		},
		Activated: dbUser.Activated,
		Version:   dbUser.Version,
	}
}

type password struct{
	plaintext *string
	hash []byte
}

func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12) 
	if err != nil {
		return err 
	}
	p.plaintext = &plaintextPassword 
	p.hash = hash
	return nil
}

func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword)) 
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil 
		default:
			return false, err }
	}
	return true, nil 
}