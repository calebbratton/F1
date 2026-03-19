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

// GET /seasons/{season}/driver-standings
func (h *StandingsHandler) DriverStandings(w http.ResponseWriter, r *http.Request) {
	season, err := strconv.Atoi(chi.URLParam(r, "season"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid season")
		return
	}
	// Optionally filter by round (latest if not specified)
	roundParam := r.URL.Query().Get("round")

	var rows interface{ Next() bool; Scan(...any) error; Close() }
	if roundParam != "" {
		round, _ := strconv.Atoi(roundParam)
		rows, err = h.db.Query(context.Background(), `
			SELECT ds.standing_id,
			       ra.race_id, ra.season, ra.round, ra.name,
			       d.driver_id, d.ref, d.forename, d.surname, d.code,
			       ds.points, ds.position, ds.wins
			FROM driver_standings ds
			JOIN races ra ON ra.race_id = ds.race_id
			JOIN drivers d ON d.driver_id = ds.driver_id
			WHERE ra.season = $1 AND ra.round = $2
			ORDER BY ds.position
		`, season, round)
	} else {
		// Latest round of the season
		rows, err = h.db.Query(context.Background(), `
			SELECT ds.standing_id,
			       ra.race_id, ra.season, ra.round, ra.name,
			       d.driver_id, d.ref, d.forename, d.surname, d.code,
			       ds.points, ds.position, ds.wins
			FROM driver_standings ds
			JOIN races ra ON ra.race_id = ds.race_id
			JOIN drivers d ON d.driver_id = ds.driver_id
			WHERE ra.race_id = (
				SELECT race_id FROM races
				WHERE season = $1
				ORDER BY round DESC
				LIMIT 1
			)
			ORDER BY ds.position
		`, season)
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query driver standings")
		return
	}
	defer rows.Close()

	standings := []models.DriverStanding{}
	for rows.Next() {
		var s models.DriverStanding
		if err := rows.Scan(
			&s.StandingID,
			&s.Race.RaceID, &s.Race.Season, &s.Race.Round, &s.Race.Name,
			&s.Driver.DriverID, &s.Driver.Ref, &s.Driver.Forename, &s.Driver.Surname, &s.Driver.Code,
			&s.Points, &s.Position, &s.Wins,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to scan standing")
			return
		}
		standings = append(standings, s)
	}

	respond(w, http.StatusOK, standings)
}

// GET /seasons/{season}/constructor-standings
func (h *StandingsHandler) ConstructorStandings(w http.ResponseWriter, r *http.Request) {
	season, err := strconv.Atoi(chi.URLParam(r, "season"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid season")
		return
	}
	roundParam := r.URL.Query().Get("round")

	var rows interface{ Next() bool; Scan(...any) error; Close() }
	if roundParam != "" {
		round, _ := strconv.Atoi(roundParam)
		rows, err = h.db.Query(context.Background(), `
			SELECT cs.standing_id,
			       ra.race_id, ra.season, ra.round, ra.name,
			       co.constructor_id, co.ref, co.name,
			       cs.points, cs.position, cs.wins
			FROM constructor_standings cs
			JOIN races ra ON ra.race_id = cs.race_id
			JOIN constructors co ON co.constructor_id = cs.constructor_id
			WHERE ra.season = $1 AND ra.round = $2
			ORDER BY cs.position
		`, season, round)
	} else {
		rows, err = h.db.Query(context.Background(), `
			SELECT cs.standing_id,
			       ra.race_id, ra.season, ra.round, ra.name,
			       co.constructor_id, co.ref, co.name,
			       cs.points, cs.position, cs.wins
			FROM constructor_standings cs
			JOIN races ra ON ra.race_id = cs.race_id
			JOIN constructors co ON co.constructor_id = cs.constructor_id
			WHERE ra.race_id = (
				SELECT race_id FROM races
				WHERE season = $1
				ORDER BY round DESC
				LIMIT 1
			)
			ORDER BY cs.position
		`, season)
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query constructor standings")
		return
	}
	defer rows.Close()

	standings := []models.ConstructorStanding{}
	for rows.Next() {
		var s models.ConstructorStanding
		if err := rows.Scan(
			&s.StandingID,
			&s.Race.RaceID, &s.Race.Season, &s.Race.Round, &s.Race.Name,
			&s.Constructor.ConstructorID, &s.Constructor.Ref, &s.Constructor.Name,
			&s.Points, &s.Position, &s.Wins,
		); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to scan standing")
			return
		}
		standings = append(standings, s)
	}

	respond(w, http.StatusOK, standings)
}
