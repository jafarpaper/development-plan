package activitylog

import (
    "encoding/json"
    "net/http"
    "time"

    driver "github.com/arangodb/go-driver"
)

type Handler struct {
    repo Repository
}

func NewHandler(db driver.Database) *Handler {
    return &Handler{repo: NewRepository(db)}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    var log ActivityLog
    if err := json.NewDecoder(r.Body).Decode(&log); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    now := time.Now()
    log.CreatedAt = &now
    if err := h.repo.Create(r.Context(), &log); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    json.NewEncoder(w).Encode(log)
}

func (h *Handler) Show(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("id")
    log, err := h.repo.GetByID(r.Context(), id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }
    json.NewEncoder(w).Encode(log)
}
