package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"rateLimiting/pkg/db"
	"rateLimiting/pkg/response"
	"rateLimiting/pkg/token"

	"github.com/gorilla/mux"
)

type UserHandler struct {
	ClientRepo *token.RateLimiter
	Db         *db.DB
}

func (h *UserHandler) MockRequest(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Request allowed"))
}

func (h *UserHandler) DeleteClient(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clientID := vars["CLIENT_ID"]

	err := h.ClientRepo.DeleteClient(clientID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		response.ResponseJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	err = h.Db.DeleteClient(clientID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response.ResponseJSON(w, http.StatusInternalServerError, "Ошибка при удалении записи в БД")
		return
	}

	log.Printf("Delete Client: IP: %s", clientID)
	response.ResponseJSON(w, http.StatusOK, "Success")
}

func (h *UserHandler) AddClient(w http.ResponseWriter, r *http.Request) {
	var settings struct {
		ID       string  `json:"client_id"`
		Capacity float64 `json:"capacity"`
		Rate     float64 `json:"rate_per_sec"`
	}
	err := json.NewDecoder(r.Body).Decode(&settings)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response.ResponseJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	err = h.ClientRepo.AddClient(settings.ID, settings.Capacity, settings.Rate)
	if err != nil {
		w.WriteHeader(http.StatusConflict)
		response.ResponseJSON(w, http.StatusConflict, err.Error())
		return
	}

	err = h.Db.UpdateOrInsertClient(settings.ID, settings.Capacity, settings.Rate)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response.ResponseJSON(w, http.StatusInternalServerError, "Ошибка при добавлении записи в БД")
		return
	}

	log.Printf("Add Client: IP: %s, Capacity: %f, RefillRate: %f", settings.ID, settings.Capacity, settings.Rate)
	response.ResponseJSON(w, http.StatusCreated, "Success")
}

// JSON Query
// Capacity float64 `json:"capacity"`
// Rate     float64 `json:"rate_per_sec"`
func (h *UserHandler) EditClient(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clientID := vars["CLIENT_ID"]

	var settings struct {
		Capacity float64 `json:"capacity"`
		Rate     float64 `json:"rate_per_sec"`
	}

	err := json.NewDecoder(r.Body).Decode(&settings)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response.ResponseJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	err = h.ClientRepo.SetClientSettings(clientID, settings.Capacity, settings.Rate)
	if err != nil {
		w.WriteHeader(http.StatusConflict)
		response.ResponseJSON(w, http.StatusConflict, err.Error())
		return
	}

	err = h.Db.UpdateOrInsertClient(clientID, settings.Capacity, settings.Rate)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		response.ResponseJSON(w, http.StatusInternalServerError, "Ошибка при обновлении записи в БД")
		return
	}

	log.Printf("Update Client: IP: %s, Capacity: %f, RefillRate: %f", clientID, settings.Capacity, settings.Rate)
	response.ResponseJSON(w, http.StatusCreated, "Success")
}
