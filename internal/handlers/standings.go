package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/calebbratton/f1-api/internal/models"
)

type StandingsHandler struct {
	db *pgxpool.Pool
}

func NewStandingsHandler(db *pgxpool.Pool) *StandingsHandler {
	return &StandingsHandler{db: db}
}

// GET /seasons/{season}/driver-standings[?round=N]
func (h *StandingsHandler) DriverStandings(w http.ResponseWriter, r *http.Request) {
	season, err := strconv.Atoi(chi.URLParam(r, "season"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid season")
		return
	}

	args := []any{season}
	roundFilter := ""
	if rp := r.URL.Query().Get("round"); rp != "" {
		round, _ := strconv.Atoi(rp)
		args = append(args, round)
		roundFilter = "AND ra.round <= $2"
	}

	rows, err := h.db.Query(context.Background(), `
		SELECT
			d.driver_id, d.ref, d.forename, d.surname, d.code,
			SUM(res.points)                               AS points,
			COUNT(*) FILTER (WHERE res.position = 1)     AS wins,
			RANK() OVER (ORDER BY SUM(res.points) DESC)  AS position
		FROM results res
		JOIN races ra ON ra.race_id = res.race_id
		JOIN drivers d ON d.driver_id = res.driver_id
		WHERE ra.season = $1 `+roundFilter+`
		GROUP BY d.driver_id, d.ref, d.forename, d.surname, d.code
		ORDER BY points DESC
	`, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query driver standings")
		return
	}
	defer rows.Close()

	standings := []models.DriverStanding{}
	for rows.Next() {
		var s models.DriverStanding
		if err := rows.Scan(
			&s.Driver.DriverID, &s.Driver.Ref, &s.Driver.Forename, &s.Driver.Surname, &s.Driver.Code,
			&s.Points, &s.Wins, &s.Position,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to scan standing")
			return
		}
		standings = append(standings, s)
	}

	respond(w, http.StatusOK, standings)
}

// GET /seasons/{season}/constructor-standings[?round=N]
func (h *StandingsHandler) ConstructorStandings(w http.ResponseWriter, r *http.Request) {
	season, err := strconv.Atoi(chi.URLParam(r, "season"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid season")
		return
	}

	args := []any{season}
	roundFilter := ""
	if rp := r.URL.Query().Get("round"); rp != "" {
		round, _ := strconv.Atoi(rp)
		args = append(args, round)
		roundFilter = "AND ra.round <= $2"
	}

	rows, err := h.db.Query(context.Background(), `
		SELECT
			co.constructor_id, co.ref, co.name,
			SUM(res.points)                               AS points,
			COUNT(*) FILTER (WHERE res.position = 1)     AS wins,
			RANK() OVER (ORDER BY SUM(res.points) DESC)  AS position
		FROM results res
		JOIN races ra ON ra.race_id = res.race_id
		JOIN constructors co ON co.constructor_id = res.constructor_id
		WHERE ra.season = $1 `+roundFilter+`
		GROUP BY co.constructor_id, co.ref, co.name
		ORDER BY points DESC
	`, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query constructor standings")
		return
	}
	defer rows.Close()

	standings := []models.ConstructorStanding{}
	for rows.Next() {
		var s models.ConstructorStanding
		if err := rows.Scan(
			&s.Constructor.ConstructorID, &s.Constructor.Ref, &s.Constructor.Name,
			&s.Points, &s.Wins, &s.Position,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to scan standing")
			return
		}
		standings = append(standings, s)
	}

	respond(w, http.StatusOK, standings)
}
