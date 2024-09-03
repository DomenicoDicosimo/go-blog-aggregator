package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/google/uuid"
)

const (
	ScopeActivation = "activation"
)

type Token struct {
	Plaintext string
	Hash      []byte
	UserID    uuid.UUID
	Expiry    time.Time
	Scope     string
}

func DatabaseTokenToToken(dbToken database.Token) Token {
	return Token{
		Hash:   dbToken.Hash,
		UserID: dbToken.UserID,
		Expiry: dbToken.Expiry,
		Scope:  dbToken.Scope,
	}
}

func New(ctx context.Context, userID uuid.UUID, ttl time.Duration, scope string, db *database.Queries) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = db.InsertToken(ctx, database.InsertTokenParams{
		Hash:   token.Hash,
		UserID: token.UserID,
		Expiry: token.Expiry,
		Scope:  token.Scope,
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}

func generateToken(userID uuid.UUID, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)

	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}
