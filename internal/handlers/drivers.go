package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/calebbratton/f1-api/internal/models"
)

type DriversHandler struct {
	db *pgxpool.Pool
}

func NewDriversHandler(db *pgxpool.Pool) *DriversHandler {
	return &DriversHandler{db: db}
}

func (h *DriversHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	rows, err := h.db.Query(context.Background(), `
		SELECT driver_id, ref, number, code, forename, surname, date_of_birth, nationality, url
		FROM drivers
		ORDER BY surname, forename
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to query drivers")
		return
	}
	defer rows.Close()

	var total int
	h.db.QueryRow(context.Background(), `SELECT COUNT(*) FROM drivers`).Scan(&total)

	drivers := []models.Driver{}
	for rows.Next() {
		var d models.Driver
		if err := rows.Scan(&d.DriverID, &d.Ref, &d.Number, &d.Code, &d.Forename, &d.Surname, &d.DateOfBirth, &d.Nationality, &d.URL); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to scan driver")
			return
		}
		drivers = append(drivers, d)
	}

	respondWithMeta(w, http.StatusOK, drivers, Meta{Total: total, Limit: limit, Offset: offset})
}

func (h *DriversHandler) Get(w http.ResponseWriter, r *http.Request) {
	ref := chi.URLParam(r, "ref")

	var d models.Driver
	err := h.db.QueryRow(context.Background(), `
		SELECT driver_id, ref, number, code, forename, surname, date_of_birth, nationality, url
		FROM drivers WHERE ref = $1
	`, ref).Scan(&d.DriverID, &d.Ref, &d.Number, &d.Code, &d.Forename, &d.Surname, &d.DateOfBirth, &d.Nationality, &d.URL)
	if err != nil {
		respondError(w, http.StatusNotFound, "driver not found")
		return
	}

	respond(w, http.StatusOK, d)
}

func parsePagination(r *http.Request) (limit, offset int) {
	limit = 30
	offset = 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	return
}
