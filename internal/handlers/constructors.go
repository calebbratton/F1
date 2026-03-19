package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/calebbratton/f1-api/internal/models"
)

type ConstructorsHandler struct {
	db *pgxpool.Pool
}

func NewConstructorsHandler(db *pgxpool.Pool) *ConstructorsHandler {
	return &ConstructorsHandler{db: db}
}

func (h *ConstructorsHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	rows, err := h.db.Query(context.Background(), `
		SELECT constructor_id, ref, name, nationality, url
		FROM constructors
		ORDER BY name
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query constructors")
		return
	}
	defer rows.Close()

	var total int
	h.db.QueryRow(context.Background(), `SELECT COUNT(*) FROM constructors`).Scan(&total)

	constructors := []models.Constructor{}
	for rows.Next() {
		var c models.Constructor
		if err := rows.Scan(&c.ConstructorID, &c.Ref, &c.Name, &c.Nationality, &c.URL); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to scan constructor")
			return
		}
		constructors = append(constructors, c)
	}

	respondWithMeta(w, http.StatusOK, constructors, Meta{Total: total, Limit: limit, Offset: offset})
}

func (h *ConstructorsHandler) Get(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "ref")

	var c models.Constructor
	err := h.db.QueryRow(context.Background(), `
		SELECT constructor_id, ref, name, nationality, url
		FROM constructors WHERE ref = $1
	`, ref).Scan(&c.ConstructorID, &c.Ref, &c.Name, &c.Nationality, &c.URL)
	if err != nil {
		respondError(w, http.StatusNotFound, "constructor not found")
		return
	}

	respond(w, http.StatusOK, c)
}
