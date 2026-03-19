package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/calebbratton/f1-api/internal/models"
)

type ResultsHandler struct {
	db *pgxpool.Pool
}

func NewResultsHandler(db *pgxpool.Pool) *ResultsHandler {
	return &ResultsHandler{db: db}
}

func (h *ResultsHandler) GetByRace(w http.ResponseWriter, r *http.Request) {
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

	rows, err := h.db.Query(context.Background(), `
		SELECT res.result_id,
		       ra.race_id, ra.season, ra.round, ra.name,
		       d.driver_id, d.ref, d.forename, d.surname, d.code,
		       co.constructor_id, co.ref, co.name,
		       res.grid, res.position, res.position_text, res.points,
		       res.laps, res.status, res.time, res.fastest_lap_rank, res.fastest_lap_time, res.fastest_lap_speed
		FROM results res
		JOIN races ra ON ra.race_id = res.race_id
		JOIN drivers d ON d.driver_id = res.driver_id
		JOIN constructors co ON co.constructor_id = res.constructor_id
		WHERE ra.season = $1 AND ra.round = $2
		ORDER BY res.position_order
	`, season, round)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query results")
		return
	}
	defer rows.Close()

	results := []models.Result{}
	for rows.Next() {
		var res models.Result
		if err := rows.Scan(
			&res.ResultID,
			&res.Race.RaceID, &res.Race.Season, &res.Race.Round, &res.Race.Name,
			&res.Driver.DriverID, &res.Driver.Ref, &res.Driver.Forename, &res.Driver.Surname, &res.Driver.Code,
			&res.Constructor.ConstructorID, &res.Constructor.Ref, &res.Constructor.Name,
			&res.Grid, &res.Position, &res.PositionText, &res.Points,
			&res.Laps, &res.Status, &res.Time, &res.FastestLapRank, &res.FastestLapTime, &res.FastestLapSpeed,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to scan result")
			return
		}
		results = append(results, res)
	}

	respond(w, http.StatusOK, results)
}

func (h *ResultsHandler) GetByDriver(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "ref")
	limit, offset := parsePagination(r)
	season := r.URL.Query().Get("season")

	query := `
		SELECT res.result_id,
		       ra.race_id, ra.season, ra.round, ra.name,
		       d.driver_id, d.ref, d.forename, d.surname, d.code,
		       co.constructor_id, co.ref, co.name,
		       res.grid, res.position, res.position_text, res.points,
		       res.laps, res.status, res.time, res.fastest_lap_rank, res.fastest_lap_time, res.fastest_lap_speed
		FROM results res
		JOIN races ra ON ra.race_id = res.race_id
		JOIN drivers d ON d.driver_id = res.driver_id
		JOIN constructors co ON co.constructor_id = res.constructor_id
		WHERE d.ref = $1`
	args := []any{ref, limit, offset}
	if season != "" {
		query += ` AND ra.season = $4`
		s, _ := strconv.Atoi(season)
		args = append(args, s)
	}
	query += ` ORDER BY ra.season DESC, ra.round DESC LIMIT $2 OFFSET $3`

	rows, err := h.db.Query(context.Background(), query, args...)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query results")
		return
	}
	defer rows.Close()

	results := []models.Result{}
	for rows.Next() {
		var res models.Result
		if err := rows.Scan(
			&res.ResultID,
			&res.Race.RaceID, &res.Race.Season, &res.Race.Round, &res.Race.Name,
			&res.Driver.DriverID, &res.Driver.Ref, &res.Driver.Forename, &res.Driver.Surname, &res.Driver.Code,
			&res.Constructor.ConstructorID, &res.Constructor.Ref, &res.Constructor.Name,
			&res.Grid, &res.Position, &res.PositionText, &res.Points,
			&res.Laps, &res.Status, &res.Time, &res.FastestLapRank, &res.FastestLapTime, &res.FastestLapSpeed,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to scan result")
			return
		}
		results = append(results, res)
	}

	respond(w, http.StatusOK, results)
}
