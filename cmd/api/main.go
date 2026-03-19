package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/calebbratton/f1-api/internal/config"
	"github.com/calebbratton/f1-api/internal/db"
	"github.com/calebbratton/f1-api/internal/handlers"
)

func main() {
	// Load .env if present (ignored if missing)
	_ = godotenv.Load()

	cfg := config.Load()

	pool, err := db.Connect(cfg.DSN())
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()

	log.Println("connected to database")

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	// Handlers
	seasons := handlers.NewSeasonsHandler(pool)
	circuits := handlers.NewCircuitsHandler(pool)
	drivers := handlers.NewDriversHandler(pool)
	constructors := handlers.NewConstructorsHandler(pool)
	races := handlers.NewRacesHandler(pool)
	results := handlers.NewResultsHandler(pool)
	standings := handlers.NewStandingsHandler(pool)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"message":"F1 API","version":"1.0.0"}`))
	})

	// Seasons
	r.Get("/seasons", seasons.List)

	// Circuits
	r.Get("/circuits", circuits.List)
	r.Get("/circuits/{ref}", circuits.Get)

	// Drivers
	r.Get("/drivers", drivers.List)
	r.Get("/drivers/{ref}", drivers.Get)
	r.Get("/drivers/{ref}/results", results.GetByDriver)

	// Constructors
	r.Get("/constructors", constructors.List)
	r.Get("/constructors/{ref}", constructors.Get)

	// Races + results + standings (nested under season)
	r.Route("/seasons/{season}", func(r chi.Router) {
		r.Get("/races", races.ListBySeason)
		r.Get("/races/{round}", races.GetRace)
		r.Get("/races/{round}/results", results.GetByRace)
		r.Get("/driver-standings", standings.DriverStandings)
		r.Get("/constructor-standings", standings.ConstructorStandings)
	})

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
