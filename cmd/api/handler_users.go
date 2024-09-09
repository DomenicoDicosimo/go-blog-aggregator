package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/data"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/database"
	"github.com/DomenicoDicosimo/go-blog-aggregator/internal/validator"
	"github.com/google/uuid"
)

func (app *application) HandlerUsersCreate(w http.ResponseWriter, r *http.Request) {

	var input struct {
		Name     string `json:"name" validate:"required,max=500"`
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=8,max=72"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	v.ValidateStruct(input)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
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
		app.serverErrorResponse(w, r, err)
		return
	}

	dbUser, err := app.db.InsertUser(r.Context(), database.InsertUserParams{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.GetPasswordHash(),
		Activated:    user.Activated,
	})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	defaultPermissions := []string{
		"feeds:read",
		"feeds:write",
		"feed_follows:write",
		"feed_follows:read",
		"posts:read",
	}

	err = app.db.GrantPermissionToUser(r.Context(), database.GrantPermissionToUserParams{
		UserID: dbUser.ID,
		Codes:  defaultPermissions,
	})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	token, err := data.NewToken(r.Context(), dbUser.ID, 3*24*time.Hour, data.ScopeActivation, app.db)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.background(func() {

		data := map[string]any{
			"activationToken": token.Plaintext,
			"userID":          dbUser.ID,
		}

		err = app.mailer.Send(user.Email, "user_welcome.tmpl", data)
		if err != nil {
			app.logger.Error(err.Error())
		}
	})

	err = app.writeJSON(w, http.StatusOK, envelope{"user": data.DatabaseUserToUser(dbUser)}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) HandlerUserActivate(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TokenPlaintext string `json:"token" validate:"required,len=26"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	v.ValidateStruct(input)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := data.GetForToken(r.Context(), data.ScopeActivation, input.TokenPlaintext, app.db)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("token", "invalid or expired activation token")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	user.Activated = true

	err = app.db.UpdateUser(r.Context(), database.UpdateUserParams{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.GetPasswordHash(),
		Activated:    user.Activated,
		UpdatedAt:    time.Now().UTC(),
		Version:      user.Version,
	})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.db.DeleteTokenForUser(r.Context(), database.DeleteTokenForUserParams{
		Scope:  data.ScopeActivation,
		UserID: user.ID,
	})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
