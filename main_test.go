package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStrictJSON(t *testing.T) {
	for _, body := range []string{`{"x":1}`, `{} {}`, `{`, strings.Repeat(" ", 9000) + `{}`} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		var v struct{}
		if decode(w, r, &v) || w.Code != 400 {
			t.Errorf("accepted invalid body")
		}
	}
}
func TestValidation(t *testing.T) {
	s := showtime{MovieID: 1, AuditoriumID: 1, Price: 1, StartsAt: time.Now().Add(time.Hour), EndsAt: time.Now().Add(3 * time.Hour)}
	if !valid(s) {
		t.Fatal("valid rejected")
	}
	s.Price = 0
	if valid(s) {
		t.Fatal("zero price accepted")
	}
	s.Price = 1
	s.EndsAt = s.StartsAt
	if valid(s) {
		t.Fatal("empty interval accepted")
	}
}
func TestMissingAuthorization(t *testing.T) {
	w := httptest.NewRecorder()
	called := false
	(app{}).auth(func(wr http.ResponseWriter, r *http.Request) { called = true }, true)(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 401 || called {
		t.Fatal("auth bypass")
	}
}
