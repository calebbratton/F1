package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LiveHandler serves the latest state for each F1 live timing topic.
// Data is written by the ingestor and served directly from the live_state table.
type LiveHandler struct {
	db *pgxpool.Pool
}

func NewLiveHandler(db *pgxpool.Pool) *LiveHandler {
	return &LiveHandler{db: db}
}

// GET /live/timing
// Returns the full TimingData snapshot (positions, gaps, lap times for all drivers).
func (h *LiveHandler) Timing(w http.ResponseWriter, r *http.Request) {
	h.serveTopic(w, "TimingData")
}

// GET /live/car-data
// Returns the latest CarData snapshot (speed, throttle, brake, gear, DRS per driver).
func (h *LiveHandler) CarData(w http.ResponseWriter, r *http.Request) {
	h.serveTopic(w, "CarData.z")
}

// GET /live/positions
// Returns the latest GPS position snapshot for all cars (useful for a track map).
func (h *LiveHandler) Positions(w http.ResponseWriter, r *http.Request) {
	h.serveTopic(w, "Position.z")
}

// GET /live/weather
// Returns the current weather data (air temp, track temp, rain, wind, humidity).
func (h *LiveHandler) Weather(w http.ResponseWriter, r *http.Request) {
	h.serveTopic(w, "WeatherData")
}

// GET /live/race-control
// Returns race control messages (safety car, VSC, flags, penalties, etc.).
func (h *LiveHandler) RaceControl(w http.ResponseWriter, r *http.Request) {
	h.serveTopic(w, "RaceControlMessages")
}

// GET /live/session
// Returns current session metadata (circuit, session type, start time, etc.).
func (h *LiveHandler) Session(w http.ResponseWriter, r *http.Request) {
	h.serveTopic(w, "SessionInfo")
}

// GET /live/track-status
// Returns the current track status (green, yellow, safety car, red flag, etc.).
func (h *LiveHandler) TrackStatus(w http.ResponseWriter, r *http.Request) {
	h.serveTopic(w, "TrackStatus")
}

// GET /live/drivers
// Returns the current driver list (number, name, team, abbreviation).
func (h *LiveHandler) Drivers(w http.ResponseWriter, r *http.Request) {
	h.serveTopic(w, "DriverList")
}

// GET /live/laps
// Returns the current lap count.
func (h *LiveHandler) LapCount(w http.ResponseWriter, r *http.Request) {
	h.serveTopic(w, "LapCount")
}

// GET /live/topic/{name}
// Generic escape hatch — fetch any raw topic by name.
func (h *LiveHandler) RawTopic(w http.ResponseWriter, r *http.Request) {
	h.serveTopic(w, chi.URLParam(r, "name"))
}

// GET /live
// Returns a summary: which topics have data and when they were last updated.
func (h *LiveHandler) Index(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(context.Background(), `
		SELECT topic, updated_at FROM live_state ORDER BY topic
	`)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query live state")
		return
	}
	defer rows.Close()

	type entry struct {
		Topic     string `json:"topic"`
		UpdatedAt string `json:"updated_at"`
	}
	var topics []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.Topic, &e.UpdatedAt); err != nil {
			continue
		}
		topics = append(topics, e)
	}
	respond(w, http.StatusOK, topics)
}

func (h *LiveHandler) serveTopic(w http.ResponseWriter, topic string) {
	var data []byte
	err := h.db.QueryRow(context.Background(), `
		SELECT data FROM live_state WHERE topic = $1
	`, topic).Scan(&data)
	if err != nil {
		respondError(w, http.StatusNotFound, "no live data for topic: "+topic)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
