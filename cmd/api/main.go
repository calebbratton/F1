package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/calebbratton/f1-api/internal/config"
	"github.com/calebbratton/f1-api/internal/db"
	"github.com/calebbratton/f1-api/internal/handlers"
	"github.com/calebbratton/f1-api/internal/ingestor"
	"github.com/calebbratton/f1-api/internal/middleware"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	pool, err := db.Connect(cfg.DSN())
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()
	log.Println("connected to database")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the live timing ingestor only when explicitly enabled.
	// Outside of race weekends the stream is idle, so this is opt-in.
	if cfg.EnableIngestor {
		ing := ingestor.New(pool)
		go ing.Run(ctx)
		log.Println("live timing ingestor started")
	} else {
		log.Println("live timing ingestor disabled (set ENABLE_INGESTOR=true to enable)")
	}

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("shutting down...")
		cancel()
	}()

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.SetHeader("Content-Type", "application/json"))
	r.Use(middleware.RateLimit(cfg.RateLimitRPS, cfg.RateLimitBurst))
	r.Use(middleware.APIKey(cfg.APIKey))

	// Handlers
	seasons := handlers.NewSeasonsHandler(pool)
	circuits := handlers.NewCircuitsHandler(pool)
	drivers := handlers.NewDriversHandler(pool)
	constructors := handlers.NewConstructorsHandler(pool)
	races := handlers.NewRacesHandler(pool)
	results := handlers.NewResultsHandler(pool)
	standings := handlers.NewStandingsHandler(pool)
	live := handlers.NewLiveHandler(pool)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"message":"F1 API","version":"1.0.0"}`))
	})

	r.Get("/seasons", seasons.List)

	r.Get("/circuits", circuits.List)
	r.Get("/circuits/{ref}", circuits.Get)

	r.Get("/drivers", drivers.List)
	r.Get("/drivers/{ref}", drivers.Get)
	r.Get("/drivers/{ref}/results", results.GetByDriver)

	r.Get("/constructors", constructors.List)
	r.Get("/constructors/{ref}", constructors.Get)

	r.Route("/seasons/{season}", func(r chi.Router) {
		r.Get("/races", races.ListBySeason)
		r.Get("/races/{round}", races.GetRace)
		r.Get("/races/{round}/results", results.GetByRace)
		r.Get("/driver-standings", standings.DriverStandings)
		r.Get("/constructor-standings", standings.ConstructorStandings)
	})

	r.Route("/live", func(r chi.Router) {
		r.Get("/", live.Index)
		r.Get("/session", live.Session)
		r.Get("/timing", live.Timing)
		r.Get("/car-data", live.CarData)
		r.Get("/positions", live.Positions)
		r.Get("/weather", live.Weather)
		r.Get("/race-control", live.RaceControl)
		r.Get("/track-status", live.TrackStatus)
		r.Get("/drivers", live.Drivers)
		r.Get("/laps", live.LapCount)
		r.Get("/topic/{name}", live.RawTopic)
	})

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
