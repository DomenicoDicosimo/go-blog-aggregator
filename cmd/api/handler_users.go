package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/data"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/validator"
	"github.com/google/uuid"
)

func (cfg *APIConfig) HandlerUsersCreate(w http.ResponseWriter, r *http.Request) {

	var input struct {
		Name     string `json:"name" validate:"required,max=500"`
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"-" validate:"required"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	v := validator.New()
	v.ValidateStruct(input)
	if !v.Valid() {
		respondWithJSON(w, http.StatusUnprocessableEntity, v.Errors)
		return
	}

	user := &data.User{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
		Version:   1,
	}

	err = user.Password.Set(input.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error setting password")
		return
	}

	dbUser, err := cfg.DB.InsertUser(r.Context(), database.InsertUserParams{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.GetPasswordHash(),
		Activated:    user.Activated,
	})

	respondWithJSON(w, http.StatusOK, data.DatabaseUserToUser(dbUser))
}

func (cfg *APIConfig) HandlerUsersGet(w http.ResponseWriter, r *http.Request, user database.User) {
	respondWithJSON(w, http.StatusOK, data.DatabaseUserToUser(user))
}
