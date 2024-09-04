package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

var AnonymousUser = &User{}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  password  `json:"-"`
	Activated bool      `json:"activated"`
	Version   int32     `json:"-"`
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

func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

func GetForToken(ctx context.Context, tokenScope, tokenPlaintext string, db *database.Queries) (User, error) {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

	dbUser, err := db.GetForToken(ctx, database.GetForTokenParams{
		Hash:   tokenHash[:],
		Scope:  tokenScope,
		Expiry: time.Now(),
	})
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return User{}, ErrRecordNotFound
		default:
			return User{}, err
		}
	}

	return DatabaseUserToUser(dbUser), nil
}

type password struct {
	plaintext *string
	hash      []byte
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

func (u *User) GetPasswordHash() []byte {
	return u.Password.hash
}

func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}
