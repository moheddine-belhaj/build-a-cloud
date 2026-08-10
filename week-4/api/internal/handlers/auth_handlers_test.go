package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/types"
)

const validRegisterBody = `{"email":"a@example.com","password":"hunter22","firstName":"Ada","lastName":"Lovelace"}`

func TestRegister(t *testing.T) {
	srv, _, _ := newFakeServer()

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(validRegisterBody))
	w := httptest.NewRecorder()

	srv.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusCreated, w.Body.String())
	}
	got := decode[types.AuthResponse](t, w.Body)
	if got.Token == "" {
		t.Error("expected a non-empty token")
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	srv, _, _ := newFakeServer()

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(`{"email":"a@example.com","password":"short","firstName":"Ada","lastName":"Lovelace"}`))
	w := httptest.NewRecorder()

	srv.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRegister_MissingName(t *testing.T) {
	srv, _, _ := newFakeServer()

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(`{"email":"a@example.com","password":"hunter22"}`))
	w := httptest.NewRecorder()

	srv.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	srv, _, _ := newFakeServer()

	srv.Register(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(validRegisterBody)))

	w := httptest.NewRecorder()
	srv.Register(w, httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(validRegisterBody)))

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestLogin(t *testing.T) {
	srv, _, _ := newFakeServer()
	srv.Register(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(validRegisterBody)))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(`{"email":"a@example.com","password":"hunter22"}`))
	srv.Login(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	got := decode[types.AuthResponse](t, w.Body)
	if got.Token == "" {
		t.Error("expected a non-empty token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	srv, _, _ := newFakeServer()
	srv.Register(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/auth/register", strings.NewReader(validRegisterBody)))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(`{"email":"a@example.com","password":"wrong-password"}`))
	srv.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	srv, _, _ := newFakeServer()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(`{"email":"nobody@example.com","password":"hunter22"}`))
	srv.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
