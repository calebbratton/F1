package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/calebbratton/f1-api/internal/models"
)

type RacesHandler struct {
	db *pgxpool.Pool
}

func NewRacesHandler(db *pgxpool.Pool) *RacesHandler {
	return &RacesHandler{db: db}
}

func (h *RacesHandler) ListBySeason(w http.ResponseWriter, r *http.Request) {
	season, err := strconv.Atoi(chi.URLParam(r, "season"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid season")
		return
	}

	rows, err := h.db.Query(context.Background(), `
		SELECT r.race_id, r.season, r.round, r.name, r.date, r.time, r.url,
		       c.circuit_id, c.ref, c.name, c.location, c.country, c.lat, c.lng, c.url
		FROM races r
		JOIN circuits c ON c.circuit_id = r.circuit_id
		WHERE r.season = $1
		ORDER BY r.round
	`, season)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query races")
		return
	}
	defer rows.Close()

	races := []models.Race{}
	for rows.Next() {
		var race models.Race
		if err := rows.Scan(
			&race.RaceID, &race.Season, &race.Round, &race.Name, &race.Date, &race.Time, &race.URL,
			&race.Circuit.CircuitID, &race.Circuit.Ref, &race.Circuit.Name,
			&race.Circuit.Location, &race.Circuit.Country, &race.Circuit.Lat, &race.Circuit.Lng, &race.Circuit.URL,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to scan race")
			return
		}
		races = append(races, race)
	}

	respond(w, http.StatusOK, races)
}

func (h *RacesHandler) GetRace(w http.ResponseWriter, r *http.Request) {
	season, err := strconv.Atoi(chi.URLParam(r, "season"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid season")
		return
	}
	round, err := strconv.Atoi(chi.URLParam(r, "round"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid round")
		return
	}

	var race models.Race
	err = h.db.QueryRow(context.Background(), `
		SELECT r.race_id, r.season, r.round, r.name, r.date, r.time, r.url,
		       c.circuit_id, c.ref, c.name, c.location, c.country, c.lat, c.lng, c.url
		FROM races r
		JOIN circuits c ON c.circuit_id = r.circuit_id
		WHERE r.season = $1 AND r.round = $2
	`, season, round).Scan(
		&race.RaceID, &race.Season, &race.Round, &race.Name, &race.Date, &race.Time, &race.URL,
		&race.Circuit.CircuitID, &race.Circuit.Ref, &race.Circuit.Name,
		&race.Circuit.Location, &race.Circuit.Country, &race.Circuit.Lat, &race.Circuit.Lng, &race.Circuit.URL,
	)
	if err != nil {
		respondError(w, http.StatusNotFound, "race not found")
		return
	}

	respond(w, http.StatusOK, race)
}
