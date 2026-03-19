package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/calebbratton/f1-api/internal/models"
)

type CircuitsHandler struct {
	db *pgxpool.Pool
}

func NewCircuitsHandler(db *pgxpool.Pool) *CircuitsHandler {
	return &CircuitsHandler{db: db}
}

func (h *CircuitsHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)
	country := r.URL.Query().Get("country")

	query := `SELECT circuit_id, ref, name, location, country, lat, lng, url FROM circuits`
	args := []any{limit, offset}
	if country != "" {
		query += ` WHERE LOWER(country) = LOWER($3)`
		args = append(args, country)
	}
	query += ` ORDER BY name LIMIT $1 OFFSET $2`

	rows, err := h.db.Query(context.Background(), query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query circuits")
		return
	}
	defer rows.Close()

	var total int
	h.db.QueryRow(context.Background(), `SELECT COUNT(*) FROM circuits`).Scan(&total)

	circuits := []models.Circuit{}
	for rows.Next() {
		var c models.Circuit
		if err := rows.Scan(&c.CircuitID, &c.Ref, &c.Name, &c.Location, &c.Country, &c.Lat, &c.Lng, &c.URL); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to scan circuit")
			return
		}
		circuits = append(circuits, c)
	}

	respondWithMeta(w, http.StatusOK, circuits, Meta{Total: total, Limit: limit, Offset: offset})
}

func (h *CircuitsHandler) Get(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "ref")

	var c models.Circuit
	err := h.db.QueryRow(context.Background(), `
		SELECT circuit_id, ref, name, location, country, lat, lng, url
		FROM circuits WHERE ref = $1
	`, ref).Scan(&c.CircuitID, &c.Ref, &c.Name, &c.Location, &c.Country, &c.Lat, &c.Lng, &c.URL)
	if err != nil {
		respondError(w, http.StatusNotFound, "circuit not found")
		return
	}

	respond(w, http.StatusOK, c)
}
