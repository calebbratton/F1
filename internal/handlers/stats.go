package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StatsHandler struct {
	db *pgxpool.Pool
}

func NewStatsHandler(db *pgxpool.Pool) *StatsHandler {
	return &StatsHandler{db: db}
}

// GET /stats/points-progression?season=2024
// Returns cumulative points per driver per round — used for the championship battle chart.
func (h *StatsHandler) PointsProgression(w http.ResponseWriter, r *http.Request) {
	season, err := strconv.Atoi(r.URL.Query().Get("season"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "season is required")
		return
	}

	rows, err := h.db.Query(context.Background(), `
		SELECT
			ra.round,
			ra.name AS race_name,
			d.ref   AS driver_ref,
			d.forename || ' ' || d.surname AS driver_name,
			SUM(res.points) OVER (
				PARTITION BY d.driver_id ORDER BY ra.round
			) AS cumulative_points
		FROM results res
		JOIN races   ra ON ra.race_id   = res.race_id
		JOIN drivers d  ON d.driver_id  = res.driver_id
		WHERE ra.season = $1
		ORDER BY ra.round, cumulative_points DESC
	`, season)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type roundEntry struct {
		Round     int     `json:"round"`
		RaceName  string  `json:"race_name"`
		DriverRef string  `json:"driver_ref"`
		Name      string  `json:"name"`
		Points    float64 `json:"points"`
	}

	// Collect raw rows then pivot into per-driver series for Chart.js.
	type series struct {
		Ref    string    `json:"ref"`
		Name   string    `json:"name"`
		Points []float64 `json:"points"`
	}

	var rounds []int
	var raceNames []string
	driverOrder := []string{}
	driverMap := map[string]*series{}
	roundSeen := map[int]bool{}

	for rows.Next() {
		var e roundEntry
		if err := rows.Scan(&e.Round, &e.RaceName, &e.DriverRef, &e.Name, &e.Points); err != nil {
			respondError(w, http.StatusInternalServerError, "scan failed")
			return
		}
		if !roundSeen[e.Round] {
			rounds = append(rounds, e.Round)
			raceNames = append(raceNames, e.RaceName)
			roundSeen[e.Round] = true
		}
		if _, ok := driverMap[e.DriverRef]; !ok {
			driverMap[e.DriverRef] = &series{Ref: e.DriverRef, Name: e.Name}
			driverOrder = append(driverOrder, e.DriverRef)
		}
		driverMap[e.DriverRef].Points = append(driverMap[e.DriverRef].Points, e.Points)
	}

	drivers := make([]*series, 0, len(driverOrder))
	for _, ref := range driverOrder {
		drivers = append(drivers, driverMap[ref])
	}

	respond(w, http.StatusOK, map[string]any{
		"season":     season,
		"rounds":     rounds,
		"race_names": raceNames,
		"drivers":    drivers,
	})
}

// GET /stats/constructor-wins?from=2010&to=2024
// Returns win counts per constructor per year — stacked bar chart.
func (h *StatsHandler) ConstructorWins(w http.ResponseWriter, r *http.Request) {
	from, _ := strconv.Atoi(r.URL.Query().Get("from"))
	to, _ := strconv.Atoi(r.URL.Query().Get("to"))
	if from == 0 {
		from = 2010
	}
	if to == 0 {
		to = time.Now().Year()
	}

	rows, err := h.db.Query(context.Background(), `
		SELECT ra.season, co.name, COUNT(*) AS wins
		FROM results res
		JOIN races        ra ON ra.race_id        = res.race_id
		JOIN constructors co ON co.constructor_id = res.constructor_id
		WHERE res.position = 1
		  AND ra.season BETWEEN $1 AND $2
		GROUP BY ra.season, co.name
		ORDER BY ra.season, wins DESC
	`, from, to)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type row struct {
		Season      int
		Constructor string
		Wins        int
	}

	var rawRows []row
	seasonSet := map[int]bool{}
	constructorSet := map[string]bool{}
	for rows.Next() {
		var r row
		rows.Scan(&r.Season, &r.Constructor, &r.Wins)
		rawRows = append(rawRows, r)
		seasonSet[r.Season] = true
		constructorSet[r.Constructor] = true
	}

	// Build sorted year list and constructor list.
	years := sortedInts(seasonSet)
	yearIdx := map[int]int{}
	for i, y := range years {
		yearIdx[y] = i
	}

	// Pivot: constructor → wins per year.
	type series struct {
		Name string `json:"name"`
		Wins []int  `json:"wins"`
	}
	constructorData := map[string][]int{}
	for c := range constructorSet {
		constructorData[c] = make([]int, len(years))
	}
	for _, r := range rawRows {
		constructorData[r.Constructor][yearIdx[r.Season]] += r.Wins
	}

	// Only return constructors that have at least one win in the range.
	var seriesList []series
	for name, wins := range constructorData {
		total := 0
		for _, w := range wins {
			total += w
		}
		if total > 0 {
			seriesList = append(seriesList, series{Name: name, Wins: wins})
		}
	}

	respond(w, http.StatusOK, map[string]any{
		"years":        years,
		"constructors": seriesList,
	})
}

// GET /drivers/{ref}/career
// Returns per-season stats for a driver — used for the career arc chart.
func (h *StatsHandler) DriverCareer(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "ref")

	// Verify the driver exists.
	var driverName string
	if err := h.db.QueryRow(context.Background(),
		`SELECT forename || ' ' || surname FROM drivers WHERE ref = $1`, ref,
	).Scan(&driverName); err != nil {
		respondError(w, http.StatusNotFound, "driver not found")
		return
	}

	rows, err := h.db.Query(context.Background(), `
		SELECT
			ra.season,
			COUNT(*)                                          AS races,
			SUM(res.points)                                   AS points,
			SUM(CASE WHEN res.position = 1  THEN 1 ELSE 0 END) AS wins,
			SUM(CASE WHEN res.position <= 3 THEN 1 ELSE 0 END) AS podiums,
			SUM(CASE WHEN res.grid = 1      THEN 1 ELSE 0 END) AS poles,
			ROUND(AVG(res.position)::numeric, 2)              AS avg_finish
		FROM results res
		JOIN races   ra ON ra.race_id  = res.race_id
		JOIN drivers d  ON d.driver_id = res.driver_id
		WHERE d.ref = $1
		GROUP BY ra.season
		ORDER BY ra.season
	`, ref)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type seasonStat struct {
		Season    int     `json:"season"`
		Races     int     `json:"races"`
		Points    float64 `json:"points"`
		Wins      int     `json:"wins"`
		Podiums   int     `json:"podiums"`
		Poles     int     `json:"poles"`
		AvgFinish float64 `json:"avg_finish"`
	}
	var stats []seasonStat
	for rows.Next() {
		var s seasonStat
		rows.Scan(&s.Season, &s.Races, &s.Points, &s.Wins, &s.Podiums, &s.Poles, &s.AvgFinish)
		stats = append(stats, s)
	}

	respond(w, http.StatusOK, map[string]any{
		"driver": map[string]string{"ref": ref, "name": driverName},
		"career": stats,
	})
}

// GET /stats/compare?d1=hamilton&d2=verstappen&season=2021
// Head-to-head stats for two drivers across all shared races (optional season filter).
func (h *StatsHandler) HeadToHead(w http.ResponseWriter, r *http.Request) {
	d1Ref := r.URL.Query().Get("d1")
	d2Ref := r.URL.Query().Get("d2")
	season := r.URL.Query().Get("season")
	if d1Ref == "" || d2Ref == "" {
		respondError(w, http.StatusBadRequest, "d1 and d2 are required")
		return
	}

	type driverInfo struct {
		Ref  string `json:"ref"`
		Name string `json:"name"`
	}
	var d1, d2 driverInfo
	d1.Ref = d1Ref
	d2.Ref = d2Ref
	h.db.QueryRow(context.Background(),
		`SELECT forename || ' ' || surname FROM drivers WHERE ref = $1`, d1Ref,
	).Scan(&d1.Name)
	h.db.QueryRow(context.Background(),
		`SELECT forename || ' ' || surname FROM drivers WHERE ref = $1`, d2Ref,
	).Scan(&d2.Name)
	if d1.Name == "" || d2.Name == "" {
		respondError(w, http.StatusNotFound, "one or both drivers not found")
		return
	}

	seasonFilter := ""
	args := []any{d1Ref, d2Ref}
	if season != "" {
		seasonFilter = "AND ra.season = $3"
		args = append(args, season)
	}

	query := `
		WITH d1 AS (
			SELECT res.race_id, res.position, res.points, res.grid
			FROM results res
			JOIN drivers d ON d.driver_id = res.driver_id
			JOIN races   ra ON ra.race_id  = res.race_id
			WHERE d.ref = $1 ` + seasonFilter + `
		),
		d2 AS (
			SELECT res.race_id, res.position, res.points, res.grid
			FROM results res
			JOIN drivers d ON d.driver_id = res.driver_id
			JOIN races   ra ON ra.race_id  = res.race_id
			WHERE d.ref = $2 ` + seasonFilter + `
		),
		combined AS (
			SELECT d1.position AS p1, d2.position AS p2,
			       d1.points AS pts1, d2.points AS pts2,
			       d1.grid AS grid1, d2.grid AS grid2
			FROM d1 JOIN d2 ON d1.race_id = d2.race_id
			WHERE d1.position IS NOT NULL AND d2.position IS NOT NULL
		)
		SELECT
			COUNT(*)                                           AS races,
			SUM(CASE WHEN p1 < p2 THEN 1 ELSE 0 END)          AS d1_ahead,
			SUM(CASE WHEN p2 < p1 THEN 1 ELSE 0 END)          AS d2_ahead,
			ROUND(AVG(p1::float)::numeric, 2)                 AS d1_avg_pos,
			ROUND(AVG(p2::float)::numeric, 2)                 AS d2_avg_pos,
			COALESCE(SUM(pts1), 0)                             AS d1_pts,
			COALESCE(SUM(pts2), 0)                             AS d2_pts,
			SUM(CASE WHEN p1 = 1  THEN 1 ELSE 0 END)          AS d1_wins,
			SUM(CASE WHEN p2 = 1  THEN 1 ELSE 0 END)          AS d2_wins,
			SUM(CASE WHEN p1 <= 3 THEN 1 ELSE 0 END)          AS d1_podiums,
			SUM(CASE WHEN p2 <= 3 THEN 1 ELSE 0 END)          AS d2_podiums,
			SUM(CASE WHEN grid1 = 1 THEN 1 ELSE 0 END)        AS d1_poles,
			SUM(CASE WHEN grid2 = 1 THEN 1 ELSE 0 END)        AS d2_poles
		FROM combined
	`

	var (
		races, d1Ahead, d2Ahead, d1Wins, d2Wins, d1Podiums, d2Podiums, d1Poles, d2Poles int
		d1AvgPos, d2AvgPos, d1Pts, d2Pts                                                 float64
	)
	err := h.db.QueryRow(context.Background(), query, args...).Scan(
		&races, &d1Ahead, &d2Ahead,
		&d1AvgPos, &d2AvgPos,
		&d1Pts, &d2Pts,
		&d1Wins, &d2Wins,
		&d1Podiums, &d2Podiums,
		&d1Poles, &d2Poles,
	)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "query failed")
		return
	}

	respond(w, http.StatusOK, map[string]any{
		"races_together": races,
		"driver1": map[string]any{
			"ref": d1.Ref, "name": d1.Name,
			"ahead": d1Ahead, "wins": d1Wins, "podiums": d1Podiums,
			"poles": d1Poles, "avg_position": d1AvgPos, "points": d1Pts,
		},
		"driver2": map[string]any{
			"ref": d2.Ref, "name": d2.Name,
			"ahead": d2Ahead, "wins": d2Wins, "podiums": d2Podiums,
			"poles": d2Poles, "avg_position": d2AvgPos, "points": d2Pts,
		},
	})
}

// GET /stats/grid-vs-finish?season=2024
// Returns counts of (grid position, finish position) pairs — heatmap data.
func (h *StatsHandler) GridVsFinish(w http.ResponseWriter, r *http.Request) {
	season := r.URL.Query().Get("season")

	var (
		rows interface {
			Next() bool
			Scan(...any) error
			Close()
		}
		err error
	)

	if season != "" {
		yr, _ := strconv.Atoi(season)
		rows, err = h.db.Query(context.Background(), `
			SELECT res.grid, res.position, COUNT(*) AS cnt
			FROM results res
			JOIN races ra ON ra.race_id = res.race_id
			WHERE res.grid > 0 AND res.position IS NOT NULL
			  AND ra.season = $1
			GROUP BY res.grid, res.position
			ORDER BY res.grid, res.position
		`, yr)
	} else {
		rows, err = h.db.Query(context.Background(), `
			SELECT grid, position, COUNT(*) AS cnt
			FROM results
			WHERE grid > 0 AND position IS NOT NULL
			GROUP BY grid, position
			ORDER BY grid, position
		`)
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type cell struct {
		Grid     int `json:"grid"`
		Position int `json:"position"`
		Count    int `json:"count"`
	}
	var data []cell
	for rows.Next() {
		var c cell
		rows.Scan(&c.Grid, &c.Position, &c.Count)
		data = append(data, c)
	}

	respond(w, http.StatusOK, map[string]any{"data": data})
}

// GET /circuits/{ref}/form
// Returns top drivers by performance at a specific circuit.
func (h *StatsHandler) CircuitForm(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "ref")

	var circuitName string
	if err := h.db.QueryRow(context.Background(),
		`SELECT name FROM circuits WHERE ref = $1`, ref,
	).Scan(&circuitName); err != nil {
		respondError(w, http.StatusNotFound, "circuit not found")
		return
	}

	rows, err := h.db.Query(context.Background(), `
		SELECT
			d.ref,
			d.forename || ' ' || d.surname                      AS name,
			COUNT(*)                                              AS races,
			SUM(CASE WHEN res.position = 1  THEN 1 ELSE 0 END)  AS wins,
			SUM(CASE WHEN res.position <= 3 THEN 1 ELSE 0 END)  AS podiums,
			ROUND(AVG(res.position::float)::numeric, 2)         AS avg_position
		FROM results res
		JOIN races    ra ON ra.race_id    = res.race_id
		JOIN circuits ci ON ci.circuit_id = ra.circuit_id
		JOIN drivers  d  ON d.driver_id   = res.driver_id
		WHERE ci.ref = $1 AND res.position IS NOT NULL
		GROUP BY d.ref, d.forename, d.surname
		HAVING COUNT(*) >= 3
		ORDER BY wins DESC, avg_position ASC
		LIMIT 20
	`, ref)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "query failed")
		return
	}
	defer rows.Close()

	type row struct {
		Ref         string  `json:"ref"`
		Name        string  `json:"name"`
		Races       int     `json:"races"`
		Wins        int     `json:"wins"`
		Podiums     int     `json:"podiums"`
		AvgPosition float64 `json:"avg_position"`
	}
	var results []row
	for rows.Next() {
		var r row
		rows.Scan(&r.Ref, &r.Name, &r.Races, &r.Wins, &r.Podiums, &r.AvgPosition)
		results = append(results, r)
	}

	respond(w, http.StatusOK, map[string]any{
		"circuit": map[string]string{"ref": ref, "name": circuitName},
		"form":    results,
	})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func sortedInts(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// Simple insertion sort — small slices only.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
