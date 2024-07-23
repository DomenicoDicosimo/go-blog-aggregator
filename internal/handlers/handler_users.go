package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/models"
	"github.com/google/uuid"
)

type userParameters struct {
	Name string `json:"name" validate:"required,min=2,max=100"`
}

func validateUserParams(params userParameters) error {
	return validate.Struct(params)
}

func (cfg *APIConfig) HandlerUsersCreate(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	params := userParameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters")
		return
	}

	err = validateUserParams(params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid input parameters")
		return
	}

	user, err := cfg.DB.CreateUser(r.Context(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      params.Name,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could't create user")
		return
	}

	respondWithJSON(w, http.StatusOK, models.DatabaseUserToUser(user))
}

func (cfg *APIConfig) HandlerUsersGet(w http.ResponseWriter, r *http.Request, user database.User) {
	respondWithJSON(w, http.StatusOK, models.DatabaseUserToUser(user))
}
