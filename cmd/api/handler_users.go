package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/data"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/google/uuid"
)

func (cfg *APIConfig) HandlerUsersCreate(w http.ResponseWriter, r *http.Request) {

	var input struct {
        Name     string `json:"name"`
        Email    string `json:"email"`
        Password string `json:"password"`
    }
	
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	user := &data.User{
        Name:  input.Name,
        Email: input.Email,
    }

    err = user.Password.Set(input.Password)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Couldn't set password")
        return
    }

    v := validator.New()
    v.ValidateStruct(user)
    if !v.Valid() {
        respondWithError(w, http.StatusInternalServerError, v.Errors)
        return
    }

	dbUser, err := cfg.DB.CreateUser(r.Context(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      params.Name,
		Email: params.Email,
		Activated: false,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could't create user")
		return
	}
	
	respondWithJSON(w, http.StatusOK, data.DatabaseUserToUser(dbUser))
}

func (cfg *APIConfig) HandlerUsersGet(w http.ResponseWriter, r *http.Request, user database.User) {
	respondWithJSON(w, http.StatusOK, data.DatabaseUserToUser(user))
}
