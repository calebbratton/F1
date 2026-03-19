// cmd/seed imports F1 historical data from the Jolpica API (jolpi.ca/ergast)
// into your PostgreSQL database.
//
// Usage:
//
//	./f1-seed                   # imports 2020–current season
//	./f1-seed -from 2000        # imports 2000–current season
//	./f1-seed -from 1950        # all history (slow — ~1100 races)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"github.com/calebbratton/f1-api/internal/config"
	"github.com/calebbratton/f1-api/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

const jolpicaBase = "https://api.jolpi.ca/ergast/f1"

var httpClient = &http.Client{Timeout: 15 * time.Second}

// ── Jolpica response types ────────────────────────────────────────────────────

type mrData struct {
	Total          string          `json:"total"`
	Limit          string          `json:"limit"`
	Offset         string          `json:"offset"`
	CircuitTable   *circuitTable   `json:"CircuitTable"`
	ConstructorTable *constructorTable `json:"ConstructorTable"`
	DriverTable    *driverTable    `json:"DriverTable"`
	RaceTable      *raceTable      `json:"RaceTable"`
}

type apiResponse struct {
	MRData mrData `json:"MRData"`
}

type circuitTable struct {
	Circuits []jolpicaCircuit `json:"Circuits"`
}
type jolpicaCircuit struct {
	CircuitID   string   `json:"circuitId"`
	URL         string   `json:"url"`
	CircuitName string   `json:"circuitName"`
	Location    location `json:"Location"`
}
type location struct {
	Lat      string `json:"lat"`
	Long     string `json:"long"`
	Locality string `json:"locality"`
	Country  string `json:"country"`
}

type constructorTable struct {
	Constructors []jolpicaConstructor `json:"Constructors"`
}
type jolpicaConstructor struct {
	ConstructorID string `json:"constructorId"`
	URL           string `json:"url"`
	Name          string `json:"name"`
	Nationality   string `json:"nationality"`
}

type driverTable struct {
	Drivers []jolpicaDriver `json:"Drivers"`
}
type jolpicaDriver struct {
	DriverID        string `json:"driverId"`
	PermanentNumber string `json:"permanentNumber"`
	Code            string `json:"code"`
	URL             string `json:"url"`
	GivenName       string `json:"givenName"`
	FamilyName      string `json:"familyName"`
	DateOfBirth     string `json:"dateOfBirth"`
	Nationality     string `json:"nationality"`
}

type raceTable struct {
	Races []jolpicaRace `json:"Races"`
}
type jolpicaRace struct {
	Season   string         `json:"season"`
	Round    string         `json:"round"`
	URL      string         `json:"url"`
	RaceName string         `json:"raceName"`
	Circuit  jolpicaCircuit `json:"Circuit"`
	Date     string         `json:"date"`
	Time     string         `json:"time"`
	Results  []jolpicaResult `json:"Results"`
}
type jolpicaResult struct {
	Number       string             `json:"number"`
	Position     string             `json:"position"`
	PositionText string             `json:"positionText"`
	Points       string             `json:"points"`
	Driver       jolpicaDriver      `json:"Driver"`
	Constructor  jolpicaConstructor `json:"Constructor"`
	Grid         string             `json:"grid"`
	Laps         string             `json:"laps"`
	Status       string             `json:"status"`
	Time         *raceTime          `json:"Time"`
	FastestLap   *fastestLap        `json:"FastestLap"`
}
type raceTime struct {
	Time string `json:"time"`
}
type fastestLap struct {
	Rank         string    `json:"rank"`
	Time         raceTime  `json:"Time"`
	AverageSpeed avgSpeed  `json:"AverageSpeed"`
}
type avgSpeed struct {
	Speed string `json:"speed"`
}


// ── main ─────────────────────────────────────────────────────────────────────

func main() {
	fromYear := flag.Int("from", 2020, "first season to import (e.g. 1950 for all history)")
	flag.Parse()

	_ = godotenv.Load()
	cfg := config.Load()

	pool, err := db.Connect(cfg.DSN())
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	currentYear := time.Now().Year()
	log.Printf("seeding seasons %d–%d", *fromYear, currentYear)

	s := &seeder{db: pool}

	log.Println("importing circuits...")
	if err := s.importCircuits(); err != nil {
		log.Fatalf("circuits: %v", err)
	}

	log.Println("importing constructors...")
	if err := s.importConstructors(); err != nil {
		log.Fatalf("constructors: %v", err)
	}

	log.Println("importing drivers...")
	if err := s.importDrivers(); err != nil {
		log.Fatalf("drivers: %v", err)
	}

	for year := *fromYear; year <= currentYear; year++ {
		log.Printf("importing season %d...", year)
		if err := s.importSeason(year); err != nil {
			log.Printf("  season %d error: %v (skipping)", year, err)
		}
		time.Sleep(1 * time.Second)
	}

	log.Println("done.")
}

// ── seeder ────────────────────────────────────────────────────────────────────

type seeder struct {
	db *pgxpool.Pool
}

// get fetches a paginated Jolpica endpoint, retrying on rate-limit responses.
func get(path string, limit, offset int) (*mrData, error) {
	url := fmt.Sprintf("%s%s?limit=%d&offset=%d", jolpicaBase, path, limit, offset)
	delays := []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second}
	for attempt, maxAttempts := 0, len(delays)+1; attempt < maxAttempts; attempt++ {
		resp, err := httpClient.Get(url)
		if err != nil {
			return nil, err
		}
		var r apiResponse
		decErr := json.NewDecoder(resp.Body).Decode(&r)
		resp.Body.Close()
		if decErr == nil {
			return &r.MRData, nil
		}
		// Non-JSON response (HTML rate-limit page). Back off and retry.
		if attempt < len(delays) {
			log.Printf("  rate limited on %s, retrying in %v...", path, delays[attempt])
			time.Sleep(delays[attempt])
		} else {
			return nil, decErr
		}
	}
	return nil, fmt.Errorf("unreachable")
}

// getAll handles pagination automatically, calling visit for each page.
func getAll(path string, visit func(*mrData) error) error {
	const pageSize = 100
	offset := 0
	for {
		data, err := get(path, pageSize, offset)
		if err != nil {
			return err
		}
		if err := visit(data); err != nil {
			return err
		}
		total, _ := strconv.Atoi(data.Total)
		offset += pageSize
		if offset >= total {
			break
		}
		time.Sleep(200 * time.Millisecond) // be polite to the API
	}
	return nil
}

// ── import functions ──────────────────────────────────────────────────────────

func (s *seeder) importCircuits() error {
	return getAll("/circuits", func(d *mrData) error {
		if d.CircuitTable == nil {
			return nil
		}
		for _, c := range d.CircuitTable.Circuits {
			lat, _ := strconv.ParseFloat(c.Location.Lat, 64)
			lng, _ := strconv.ParseFloat(c.Location.Long, 64)
			_, err := s.db.Exec(context.Background(), `
				INSERT INTO circuits (ref, name, location, country, lat, lng, url)
				VALUES ($1,$2,$3,$4,$5,$6,$7)
				ON CONFLICT (ref) DO UPDATE
				  SET name=$2, location=$3, country=$4, lat=$5, lng=$6, url=$7
			`, c.CircuitID, c.CircuitName, c.Location.Locality, c.Location.Country, lat, lng, c.URL)
			if err != nil {
				return fmt.Errorf("insert circuit %s: %w", c.CircuitID, err)
			}
		}
		return nil
	})
}

func (s *seeder) importConstructors() error {
	return getAll("/constructors", func(d *mrData) error {
		if d.ConstructorTable == nil {
			return nil
		}
		for _, c := range d.ConstructorTable.Constructors {
			_, err := s.db.Exec(context.Background(), `
				INSERT INTO constructors (ref, name, nationality, url)
				VALUES ($1,$2,$3,$4)
				ON CONFLICT (ref) DO UPDATE
				  SET name=$2, nationality=$3, url=$4
			`, c.ConstructorID, c.Name, c.Nationality, c.URL)
			if err != nil {
				return fmt.Errorf("insert constructor %s: %w", c.ConstructorID, err)
			}
		}
		return nil
	})
}

func (s *seeder) importDrivers() error {
	return getAll("/drivers", func(d *mrData) error {
		if d.DriverTable == nil {
			return nil
		}
		for _, dr := range d.DriverTable.Drivers {
			num := parseNullInt(dr.PermanentNumber)
			dob := parseNullDate(dr.DateOfBirth)
			_, err := s.db.Exec(context.Background(), `
				INSERT INTO drivers (ref, number, code, forename, surname, date_of_birth, nationality, url)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
				ON CONFLICT (ref) DO UPDATE
				  SET number=$2, code=$3, forename=$4, surname=$5,
				      date_of_birth=$6, nationality=$7, url=$8
			`, dr.DriverID, num, nullStr(dr.Code), dr.GivenName, dr.FamilyName, dob, dr.Nationality, dr.URL)
			if err != nil {
				return fmt.Errorf("insert driver %s: %w", dr.DriverID, err)
			}
		}
		return nil
	})
}

func (s *seeder) importSeason(year int) error {
	// Fetch the race schedule for the season (small, always fits in one page).
	var allRaces []jolpicaRace
	if err := getAll(fmt.Sprintf("/%d/races", year), func(d *mrData) error {
		if d.RaceTable != nil {
			allRaces = append(allRaces, d.RaceTable.Races...)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("fetch races: %w", err)
	}

	for _, race := range allRaces {
		raceID, err := s.upsertRace(year, race)
		if err != nil {
			log.Printf("  round %s: %v", race.Round, err)
			continue
		}

		// Fetch results for this specific round (avoids pagination issues).
		time.Sleep(500 * time.Millisecond)
		rdata, err := get(fmt.Sprintf("/%d/%s/results", year, race.Round), 100, 0)
		if err != nil {
			log.Printf("  round %s results fetch: %v", race.Round, err)
			continue
		}
		if rdata.RaceTable != nil && len(rdata.RaceTable.Races) > 0 {
			if err := s.importResults(raceID, rdata.RaceTable.Races[0].Results); err != nil {
				log.Printf("  round %s results: %v", race.Round, err)
			}
		}
	}

	return nil
}

func (s *seeder) upsertRace(year int, race jolpicaRace) (int, error) {
	var circuitID int
	err := s.db.QueryRow(context.Background(),
		`SELECT circuit_id FROM circuits WHERE ref = $1`, race.Circuit.CircuitID,
	).Scan(&circuitID)
	if err != nil {
		return 0, fmt.Errorf("circuit %s not found: %w", race.Circuit.CircuitID, err)
	}

	round, _ := strconv.Atoi(race.Round)
	raceDate := parseNullDate(race.Date)
	raceTime := nullStr(race.Time)

	var raceID int
	err = s.db.QueryRow(context.Background(), `
		INSERT INTO races (season, round, circuit_id, name, date, time, url)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (season, round) DO UPDATE
		  SET circuit_id=$3, name=$4, date=$5, time=$6, url=$7
		RETURNING race_id
	`, year, round, circuitID, race.RaceName, raceDate, raceTime, race.URL).Scan(&raceID)
	return raceID, err
}

func (s *seeder) importResults(raceID int, results []jolpicaResult) error {
	for i, res := range results {
		var driverID int
		if err := s.db.QueryRow(context.Background(),
			`SELECT driver_id FROM drivers WHERE ref = $1`, res.Driver.DriverID,
		).Scan(&driverID); err != nil {
			continue // driver not in DB yet — skip
		}

		var constructorID int
		if err := s.db.QueryRow(context.Background(),
			`SELECT constructor_id FROM constructors WHERE ref = $1`, res.Constructor.ConstructorID,
		).Scan(&constructorID); err != nil {
			continue
		}

		position := parseNullInt(res.Position)
		grid, _ := strconv.Atoi(res.Grid)
		laps, _ := strconv.Atoi(res.Laps)
		points, _ := strconv.ParseFloat(res.Points, 64)

		var lapTime string
		if res.Time != nil {
			lapTime = res.Time.Time
		}
		var flRank *int
		var flTime, flSpeed string
		if res.FastestLap != nil {
			r, _ := strconv.Atoi(res.FastestLap.Rank)
			flRank = &r
			flTime = res.FastestLap.Time.Time
			flSpeed = res.FastestLap.AverageSpeed.Speed
		}

		_, err := s.db.Exec(context.Background(), `
			INSERT INTO results
			  (race_id, driver_id, constructor_id, grid, position, position_text,
			   position_order, points, laps, time, status,
			   fastest_lap_rank, fastest_lap_time, fastest_lap_speed)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
			ON CONFLICT DO NOTHING
		`, raceID, driverID, constructorID, grid, position, res.PositionText,
			i+1, points, laps, nullStr(lapTime), res.Status,
			flRank, nullStr(flTime), nullStr(flSpeed))
		if err != nil {
			log.Printf("    result driver %s: %v", res.Driver.DriverID, err)
		}
	}
	return nil
}


// ── helpers ───────────────────────────────────────────────────────────────────

func parseNullInt(s string) *int {
	if s == "" {
		return nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &v
}

func parseNullDate(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
