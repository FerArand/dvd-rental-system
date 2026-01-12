package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type App struct {
	DB *sql.DB
}

type apiError struct {
	Error string `json:"error"`
}

func respondJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if v != nil {
		json.NewEncoder(w).Encode(v)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func connectDB() (*sql.DB, error) {
	host := getenv("PGHOST", "db")
	port := getenv("PGPORT", "5432")
	user := getenv("PGUSER", "postgres")
	pass := getenv("PGPASSWORD", "postgres")
	name := getenv("PGDATABASE", "dvdrental")
	ssl := getenv("PGSSLMODE", "disable")
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", host, port, user, pass, name, ssl)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	return db, nil
}

// ---- Auth (very simple for demo): accept email + role check exists ----

type loginRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"` // "staff" or "customer"
}

type loginResponse struct {
	Token string `json:"token"`
	Role  string `json:"role"`
	Id    int    `json:"id"`
	Name  string `json:"name"`
}

func (a *App) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	switch req.Role {
	case "staff":
		var id int
		var name string
		err := a.DB.QueryRow(`SELECT staff_id, first_name||' '||last_name FROM staff WHERE lower(email)=lower($1)`, req.Email).Scan(&id, &name)
		if err != nil {
			respondJSON(w, http.StatusUnauthorized, apiError{Error: "staff not found"})
			return
		}
		respondJSON(w, http.StatusOK, loginResponse{Token: fmt.Sprintf("staff-%d", id), Role: "staff", Id: id, Name: name})
	case "customer":
		var id int
		var name string
		err := a.DB.QueryRow(`SELECT customer_id, first_name||' '||last_name FROM customer WHERE lower(email)=lower($1)`, req.Email).Scan(&id, &name)
		if err != nil {
			respondJSON(w, http.StatusUnauthorized, apiError{Error: "customer not found"})
			return
		}
		respondJSON(w, http.StatusOK, loginResponse{Token: fmt.Sprintf("customer-%d", id), Role: "customer", Id: id, Name: name})
	default:
		respondJSON(w, http.StatusBadRequest, apiError{Error: "role must be 'staff' or 'customer'"})
	}
}

// ---- Rentals ----

type rentRequest struct {
	CustomerID  int `json:"customer_id"`
	InventoryID int `json:"inventory_id"`
	StaffID     int `json:"staff_id"`
}

type rentResponse struct {
	RentalID int `json:"rental_id"`
}

func (a *App) Rent(w http.ResponseWriter, r *http.Request) {
	var req rentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	// Ensure inventory item is not currently rented out (open rental)
	var openCount int
	if err := a.DB.QueryRow(`SELECT COUNT(*) FROM rental WHERE inventory_id=$1 AND return_date IS NULL`, req.InventoryID).Scan(&openCount); err != nil {
		respondJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	if openCount > 0 {
		respondJSON(w, http.StatusConflict, apiError{Error: "inventory already rented"})
		return
	}
	// Create rental
	var rentalID int
	err := a.DB.QueryRow(
		`INSERT INTO rental (rental_date, inventory_id, customer_id, staff_id) VALUES (NOW(), $1, $2, $3) RETURNING rental_id`,
		req.InventoryID, req.CustomerID, req.StaffID,
	).Scan(&rentalID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	respondJSON(w, http.StatusCreated, rentResponse{RentalID: rentalID})
}

func (a *App) Return(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["rental_id"]
	id, _ := strconv.Atoi(idStr)
	res, err := a.DB.Exec(`UPDATE rental SET return_date=NOW() WHERE rental_id=$1 AND return_date IS NULL`, id)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		respondJSON(w, http.StatusNotFound, apiError{Error: "rental not found or already returned"})
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"returned": id})
}

// Cancel: if rental not returned yet, delete it (simple policy)
func (a *App) Cancel(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["rental_id"]
	id, _ := strconv.Atoi(idStr)
	var exists int
	if err := a.DB.QueryRow(`SELECT COUNT(*) FROM rental WHERE rental_id=$1 AND return_date IS NULL`, id).Scan(&exists); err != nil {
		respondJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	if exists == 0 {
		respondJSON(w, http.StatusConflict, apiError{Error: "cannot cancel: rental already returned or not found"})
		return
	}
	if _, err := a.DB.Exec(`DELETE FROM rental WHERE rental_id=$1`, id); err != nil {
		respondJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"canceled": id})
}

// Helper: find an available inventory_id for a film
func (a *App) AvailableInventory(w http.ResponseWriter, r *http.Request) {
	filmIDStr := r.URL.Query().Get("film_id")
	if filmIDStr == "" {
		respondJSON(w, http.StatusBadRequest, apiError{Error: "film_id required"})
		return
	}
	filmID, _ := strconv.Atoi(filmIDStr)
	rows, err := a.DB.Query(`
		SELECT i.inventory_id
		FROM inventory i
		LEFT JOIN rental r ON i.inventory_id = r.inventory_id AND r.return_date IS NULL
		WHERE i.film_id=$1 AND r.rental_id IS NULL
		LIMIT 10`, filmID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		rows.Scan(&id)
		ids = append(ids, id)
	}
	respondJSON(w, http.StatusOK, map[string]any{"inventory_ids": ids})
}

// ---- Reports ----

// 1) Lista de todas las rentas de un cliente
func (a *App) ReportCustomerRentals(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["customer_id"]
	id, _ := strconv.Atoi(idStr)
	rows, err := a.DB.Query(`
		SELECT r.rental_id, r.rental_date, r.return_date, f.title, i.inventory_id
		FROM rental r
		JOIN inventory i ON r.inventory_id=i.inventory_id
		JOIN film f ON i.film_id=f.film_id
		WHERE r.customer_id=$1
		ORDER BY r.rental_date DESC`, id)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	defer rows.Close()
	type row struct {
		RentalID    int        `json:"rental_id"`
		RentalDate  time.Time  `json:"rental_date"`
		ReturnDate  *time.Time `json:"return_date"`
		Title       string     `json:"title"`
		InventoryID int        `json:"inventory_id"`
	}
	var out []row
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.RentalID, &x.RentalDate, &x.ReturnDate, &x.Title, &x.InventoryID); err == nil {
			out = append(out, x)
		}
	}
	respondJSON(w, http.StatusOK, out)
}

// 2) Identificar los DVD que no se han devuelto
func (a *App) ReportNotReturned(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(`
		SELECT r.rental_id, c.first_name||' '||c.last_name AS customer, f.title, r.rental_date, i.inventory_id
		FROM rental r
		JOIN customer c ON r.customer_id=c.customer_id
		JOIN inventory i ON r.inventory_id=i.inventory_id
		JOIN film f ON i.film_id=f.film_id
		WHERE r.return_date IS NULL
		ORDER BY r.rental_date ASC`)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	defer rows.Close()
	type row struct {
		RentalID    int       `json:"rental_id"`
		Customer    string    `json:"customer"`
		Title       string    `json:"title"`
		RentalDate  time.Time `json:"rental_date"`
		InventoryID int       `json:"inventory_id"`
	}
	var out []row
	for rows.Next() {
		var x row
		rows.Scan(&x.RentalID, &x.Customer, &x.Title, &x.RentalDate, &x.InventoryID)
		out = append(out, x)
	}
	respondJSON(w, http.StatusOK, out)
}

// 3) Determinar los DVD más rentados (top N)
func (a *App) ReportTopRented(w http.ResponseWriter, r *http.Request) {
	nStr := r.URL.Query().Get("limit")
	if nStr == "" {
		nStr = "10"
	}
	n, _ := strconv.Atoi(nStr)
	rows, err := a.DB.Query(`
		SELECT f.title, COUNT(*) as total
		FROM rental r
		JOIN inventory i ON r.inventory_id=i.inventory_id
		JOIN film f ON i.film_id=f.film_id
		GROUP BY f.title
		ORDER BY total DESC
		LIMIT $1`, n)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	defer rows.Close()
	type row struct {
		Title string `json:"title"`
		Total int    `json:"total"`
	}
	var out []row
	for rows.Next() {
		var x row
		rows.Scan(&x.Title, &x.Total)
		out = append(out, x)
	}
	respondJSON(w, http.StatusOK, out)
}

// 4) Calcular el total de ganancia generada por cada miembro del staff
// In dvdrental, payments link to staff via staff_id
func (a *App) ReportRevenueByStaff(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(`
		SELECT s.staff_id, s.first_name||' '||s.last_name as staff, COALESCE(SUM(p.amount),0) as revenue
		FROM staff s
		LEFT JOIN payment p ON s.staff_id = p.staff_id
		GROUP BY s.staff_id, staff
		ORDER BY revenue DESC`)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	defer rows.Close()
	type row struct {
		StaffID int     `json:"staff_id"`
		Staff   string  `json:"staff"`
		Revenue float64 `json:"revenue"`
	}
	var out []row
	for rows.Next() {
		var x row
		rows.Scan(&x.StaffID, &x.Staff, &x.Revenue)
		out = append(out, x)
	}
	respondJSON(w, http.StatusOK, out)
}

func (a *App) Health(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"ok": "true"})
}

func main() {
	db, err := connectDB()
	if err != nil {
		log.Fatalf("DB error: %v", err)
	}
	app := &App{DB: db}

	r := mux.NewRouter()
	// auth
	r.HandleFunc("/api/auth/login", app.Login).Methods("POST")
	// rentals
	r.HandleFunc("/api/rentals", app.Rent).Methods("POST")
	r.HandleFunc("/api/returns/{rental_id}", app.Return).Methods("POST")
	r.HandleFunc("/api/rentals/{rental_id}/cancel", app.Cancel).Methods("POST")
	r.HandleFunc("/api/inventory/available", app.AvailableInventory).Methods("GET")

	// reports
	r.HandleFunc("/api/reports/customer/{customer_id}/rentals", app.ReportCustomerRentals).Methods("GET")
	r.HandleFunc("/api/reports/not-returned", app.ReportNotReturned).Methods("GET")
	r.HandleFunc("/api/reports/top-rented", app.ReportTopRented).Methods("GET")
	r.HandleFunc("/api/reports/revenue-by-staff", app.ReportRevenueByStaff).Methods("GET")

	// health
	r.HandleFunc("/health", app.Health).Methods("GET")

	addr := getenv("ADDR", ":8080")
	log.Printf("Listening on %s", addr)
	http.ListenAndServe(addr, r)
}
