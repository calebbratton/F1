package models

import "time"

type Season struct {
	Year int `json:"year"`
}

type Circuit struct {
	CircuitID   int     `json:"circuit_id"`
	Ref         string  `json:"ref"`
	Name        string  `json:"name"`
	Location    string  `json:"location"`
	Country     string  `json:"country"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	URL         string  `json:"url,omitempty"`
}

type Constructor struct {
	ConstructorID int    `json:"constructor_id"`
	Ref           string `json:"ref"`
	Name          string `json:"name"`
	Nationality   string `json:"nationality"`
	URL           string `json:"url,omitempty"`
}

type Driver struct {
	DriverID    int        `json:"driver_id"`
	Ref         string     `json:"ref"`
	Number      *int       `json:"number,omitempty"`
	Code        string     `json:"code,omitempty"`
	Forename    string     `json:"forename"`
	Surname     string     `json:"surname"`
	DateOfBirth *time.Time `json:"date_of_birth,omitempty"`
	Nationality string     `json:"nationality"`
	URL         string     `json:"url,omitempty"`
}

type Race struct {
	RaceID    int        `json:"race_id"`
	Season    int        `json:"season"`
	Round     int        `json:"round"`
	Circuit   Circuit    `json:"circuit"`
	Name      string     `json:"name"`
	Date      *time.Time `json:"date,omitempty"`
	Time      string     `json:"time,omitempty"`
	URL       string     `json:"url,omitempty"`
}

type Result struct {
	ResultID        int          `json:"result_id"`
	Race            RaceRef      `json:"race"`
	Driver          DriverRef    `json:"driver"`
	Constructor     ConstructorRef `json:"constructor"`
	Grid            int          `json:"grid"`
	Position        *int         `json:"position,omitempty"`
	PositionText    string       `json:"position_text"`
	Points          float64      `json:"points"`
	Laps            int          `json:"laps"`
	Status          string       `json:"status"`
	Time            string       `json:"time,omitempty"`
	FastestLapRank  *int         `json:"fastest_lap_rank,omitempty"`
	FastestLapTime  string       `json:"fastest_lap_time,omitempty"`
	FastestLapSpeed string       `json:"fastest_lap_speed,omitempty"`
}

type DriverStanding struct {
	Driver   DriverRef `json:"driver"`
	Points   float64   `json:"points"`
	Position int       `json:"position"`
	Wins     int       `json:"wins"`
}

type ConstructorStanding struct {
	Constructor ConstructorRef `json:"constructor"`
	Points      float64        `json:"points"`
	Position    int            `json:"position"`
	Wins        int            `json:"wins"`
}

// Lightweight reference types used in nested responses
type RaceRef struct {
	RaceID int `json:"race_id"`
	Season int `json:"season"`
	Round  int `json:"round"`
	Name   string `json:"name"`
}

type DriverRef struct {
	DriverID int    `json:"driver_id"`
	Ref      string `json:"ref"`
	Forename string `json:"forename"`
	Surname  string `json:"surname"`
	Code     string `json:"code,omitempty"`
}

type ConstructorRef struct {
	ConstructorID int    `json:"constructor_id"`
	Ref           string `json:"ref"`
	Name          string `json:"name"`
}
