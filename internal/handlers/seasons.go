package handlers

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/calebbratton/f1-api/internal/models"
)

type SeasonsHandler struct {
	db *pgxpool.Pool
}

func NewSeasonsHandler(db *pgxpool.Pool) *SeasonsHandler {
	return &SeasonsHandler{db: db}
}

func (h *SeasonsHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(context.Background(), `
		SELECT DISTINCT season FROM races ORDER BY season DESC
	`)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query seasons")
		return
	}
	defer rows.Close()

	seasons := []models.Season{}
	for rows.Next() {
		var s models.Season
		if err := rows.Scan(&s.Year); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to scan season")
			return
		}
		seasons = append(seasons, s)
	}

	respond(w, http.StatusOK, seasons)
}
