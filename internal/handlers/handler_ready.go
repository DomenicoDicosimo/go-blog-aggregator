package handlers

import "net/http"

func HandlerReadiness(w http.ResponseWriter, r *http.Request) {
	type okResponse struct {
		Status string `json:"status"`
	}
	respondWithJSON(w, http.StatusOK, okResponse{Status: "ok"})
}

func HandlerError(w http.ResponseWriter, r *http.Request) {
	respondWithError(w, 500, "Internal Server Error")
}
