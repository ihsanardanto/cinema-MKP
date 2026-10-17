package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type app struct{ db *sql.DB }
type showtime struct {
	ID           int64     `json:"id"`
	MovieID      int64     `json:"movie_id"`
	AuditoriumID int64     `json:"auditorium_id"`
	StartsAt     time.Time `json:"starts_at"`
	EndsAt       time.Time `json:"ends_at"`
	Price        int64     `json:"price"`
	Status       string    `json:"status"`
}

func send(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}
func fail(w http.ResponseWriter, code int, msg string) {
	send(w, code, map[string]string{"error": msg})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(v) != nil || d.Decode(&struct{}{}) != io.EOF {
		fail(w, 400, "invalid JSON body")
		return false
	}
	return true
}
func dbError(w http.ResponseWriter, err error) {
	var e *pq.Error
	if errors.As(err, &e) {
		switch e.Code {
		case "23P01", "23505":
			fail(w, 409, "schedule overlaps or resource already exists")
			return
		case "23503", "23514":
			fail(w, 400, "invalid reference or value")
			return
		}
	}
	if errors.Is(err, sql.ErrNoRows) {
		fail(w, 404, "not found")
		return
	}
	log.Print(err)
	fail(w, 500, "internal error")
}
func hash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func (a app) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	var id int64
	var password string
	err := a.db.QueryRowContext(r.Context(), "SELECT id,password_hash FROM users WHERE email=$1", strings.ToLower(strings.TrimSpace(in.Email))).Scan(&id, &password)
	// A fixed bcrypt hash also burns work for unknown users, reducing timing differences.
	if errors.Is(err, sql.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$12$R9h/cIPz0gi.URNNX3kh2OPST9/PgBkqquzi.Ss7KIUgO2t0jWMUW"), []byte(in.Password))
		fail(w, 401, "invalid credentials")
		return
	}
	if err != nil {
		dbError(w, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(password), []byte(in.Password)) != nil {
		fail(w, 401, "invalid credentials")
		return
	}
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		dbError(w, err)
		return
	}
	token := hex.EncodeToString(b)
	_, err = a.db.ExecContext(r.Context(), "INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,now()+interval '1 hour')", hash(token), id)
	if err != nil {
		dbError(w, err)
		return
	}
	send(w, 200, map[string]any{"access_token": token, "token_type": "Bearer", "expires_in": 3600})
}
func (a app) auth(next http.HandlerFunc, admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			fail(w, 401, "bearer token required")
			return
		}
		var role string
		err := a.db.QueryRowContext(r.Context(), "SELECT u.role FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at>now()", hash(strings.TrimPrefix(h, "Bearer "))).Scan(&role)
		if errors.Is(err, sql.ErrNoRows) {
			fail(w, 401, "invalid or expired token")
			return
		}
		if err != nil {
			dbError(w, err)
			return
		}
		if admin && role != "admin" {
			fail(w, 403, "admin required")
			return
		}
		next(w, r)
	}
}

const cols = "id,movie_id,auditorium_id,starts_at,ends_at,price,status"

func scan(s interface{ Scan(...any) error }) (showtime, error) {
	var v showtime
	e := s.Scan(&v.ID, &v.MovieID, &v.AuditoriumID, &v.StartsAt, &v.EndsAt, &v.Price, &v.Status)
	return v, e
}
func id(r *http.Request) (int64, error) {
	n, e := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if n <= 0 {
		return 0, errors.New("invalid id")
	}
	return n, e
}
func valid(v showtime) bool {
	return v.MovieID > 0 && v.AuditoriumID > 0 && v.Price > 0 && v.StartsAt.After(time.Now()) && v.EndsAt.After(v.StartsAt) && (v.Status == "" || v.Status == "scheduled")
}
func (a app) list(w http.ResponseWriter, r *http.Request) {
	limit, offset := 50, 0
	for k, dst := range map[string]*int{"limit": &limit, "offset": &offset} {
		if s := r.URL.Query().Get(k); s != "" {
			n, e := strconv.Atoi(s)
			if e != nil || n < 0 {
				fail(w, 400, "invalid pagination")
				return
			}
			*dst = n
		}
	}
	if limit < 1 || limit > 100 {
		fail(w, 400, "limit must be 1..100")
		return
	}
	rows, e := a.db.QueryContext(r.Context(), "SELECT "+cols+" FROM showtimes ORDER BY starts_at,id LIMIT $1 OFFSET $2", limit, offset)
	if e != nil {
		dbError(w, e)
		return
	}
	defer rows.Close()
	out := []showtime{}
	for rows.Next() {
		v, e := scan(rows)
		if e != nil {
			dbError(w, e)
			return
		}
		out = append(out, v)
	}
	if rows.Err() != nil {
		dbError(w, rows.Err())
		return
	}
	send(w, 200, out)
}
func (a app) get(w http.ResponseWriter, r *http.Request) {
	n, e := id(r)
	if e != nil {
		fail(w, 400, "invalid id")
		return
	}
	v, e := scan(a.db.QueryRowContext(r.Context(), "SELECT "+cols+" FROM showtimes WHERE id=$1", n))
	if e != nil {
		dbError(w, e)
		return
	}
	send(w, 200, v)
}
func (a app) write(w http.ResponseWriter, r *http.Request) {
	var in showtime
	if !decode(w, r, &in) {
		return
	}
	if !valid(in) {
		fail(w, 400, "positive IDs/price and future, ordered timestamps required; status must be scheduled")
		return
	}
	tx, e := a.db.BeginTx(r.Context(), nil)
	if e != nil {
		dbError(w, e)
		return
	}
	defer tx.Rollback()
	var duration int
	e = tx.QueryRowContext(r.Context(), "SELECT duration_minutes FROM movies WHERE id=$1", in.MovieID).Scan(&duration)
	if errors.Is(e, sql.ErrNoRows) {
		fail(w, 400, "unknown movie")
		return
	}
	if e != nil {
		dbError(w, e)
		return
	}
	if in.EndsAt.Sub(in.StartsAt) < time.Duration(duration)*time.Minute {
		fail(w, 400, "schedule shorter than movie")
		return
	}
	n := int64(0)
	code := 201
	if r.Method == "PUT" {
		n, e = id(r)
		if e != nil {
			fail(w, 400, "invalid id")
			return
		}
		var status string
		e = tx.QueryRowContext(r.Context(), "SELECT status FROM showtimes WHERE id=$1 FOR UPDATE", n).Scan(&status)
		if e != nil {
			dbError(w, e)
			return
		}
		var exists bool
		e = tx.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM bookings WHERE showtime_id=$1)", n).Scan(&exists)
		if e != nil {
			dbError(w, e)
			return
		}
		if exists || status != "scheduled" {
			fail(w, 409, "booked or cancelled showtime cannot be edited")
			return
		}
		_, e = tx.ExecContext(r.Context(), "DELETE FROM showtime_seats WHERE showtime_id=$1", n)
		if e != nil {
			dbError(w, e)
			return
		}
		_, e = tx.ExecContext(r.Context(), "UPDATE showtimes SET movie_id=$1,auditorium_id=$2,starts_at=$3,ends_at=$4,price=$5 WHERE id=$6", in.MovieID, in.AuditoriumID, in.StartsAt, in.EndsAt, in.Price, n)
		code = 200
	} else {
		e = tx.QueryRowContext(r.Context(), "INSERT INTO showtimes(movie_id,auditorium_id,starts_at,ends_at,price) VALUES($1,$2,$3,$4,$5) RETURNING id", in.MovieID, in.AuditoriumID, in.StartsAt, in.EndsAt, in.Price).Scan(&n)
	}
	if e != nil {
		dbError(w, e)
		return
	}
	_, e = tx.ExecContext(r.Context(), "INSERT INTO showtime_seats(showtime_id,auditorium_id,seat_id) SELECT $1,auditorium_id,id FROM seats WHERE auditorium_id=$2", n, in.AuditoriumID)
	if e != nil {
		dbError(w, e)
		return
	}
	if e = tx.Commit(); e != nil {
		dbError(w, e)
		return
	}
	in.ID = n
	in.Status = "scheduled"
	send(w, code, in)
}
func (a app) remove(w http.ResponseWriter, r *http.Request) {
	n, e := id(r)
	if e != nil {
		fail(w, 400, "invalid id")
		return
	}
	tx, e := a.db.BeginTx(r.Context(), nil)
	if e != nil {
		dbError(w, e)
		return
	}
	defer tx.Rollback()
	var status string
	e = tx.QueryRowContext(r.Context(), "SELECT status FROM showtimes WHERE id=$1 FOR UPDATE", n).Scan(&status)
	if e != nil {
		dbError(w, e)
		return
	}
	// Cancellation with bookings belongs to the refund workflow, not destructive CRUD.
	var exists bool
	e = tx.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM bookings WHERE showtime_id=$1)", n).Scan(&exists)
	if e != nil {
		dbError(w, e)
		return
	}
	if exists {
		fail(w, 409, "booked showtime requires cinema cancellation/refund workflow")
		return
	}
	_, e = tx.ExecContext(r.Context(), "DELETE FROM showtime_seats WHERE showtime_id=$1", n)
	if e != nil {
		dbError(w, e)
		return
	}
	_, e = tx.ExecContext(r.Context(), "DELETE FROM showtimes WHERE id=$1", n)
	if e != nil {
		dbError(w, e)
		return
	}
	if e = tx.Commit(); e != nil {
		dbError(w, e)
		return
	}
	send(w, 204, nil)
}
func (a app) routes() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("POST /api/login", a.login)
	m.HandleFunc("GET /api/showtimes", a.auth(a.list, false))
	m.HandleFunc("GET /api/showtimes/{id}", a.auth(a.get, false))
	m.HandleFunc("POST /api/showtimes", a.auth(a.write, true))
	m.HandleFunc("PUT /api/showtimes/{id}", a.auth(a.write, true))
	m.HandleFunc("DELETE /api/showtimes/{id}", a.auth(a.remove, true))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		m.ServeHTTP(w, r.WithContext(ctx))
	})
}
func main() {
	db, e := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if e != nil {
		log.Fatal(e)
	}
	defer db.Close()
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if e = db.PingContext(ctx); e != nil {
		log.Fatal(e)
	}
	s := http.Server{Addr: ":8080", Handler: (app{db}).routes(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	log.Fatal(s.ListenAndServe())
}
