package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/auth"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/store"
	"github.com/moheddine-belhaj/build-a-cloud/week-4/api/internal/types"
)

func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	var req types.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"message":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)
	if email == "" || req.Password == "" || firstName == "" || lastName == "" {
		http.Error(w, `{"message":"email, password, firstName and lastName are required"}`, http.StatusBadRequest)
		return
	}
	if len(req.Password) < 8 {
		http.Error(w, `{"message":"password must be at least 8 characters"}`, http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, `{"message":"failed to process password"}`, http.StatusInternalServerError)
		return
	}

	user, err := s.store.CreateUser(r.Context(), email, hash, firstName, lastName)
	if errors.Is(err, store.ErrAlreadyExists) {
		http.Error(w, `{"message":"email already registered"}`, http.StatusConflict)
		return
	}
	if err != nil {
		http.Error(w, `{"message":"failed to create user"}`, http.StatusInternalServerError)
		return
	}

	s.issueToken(w, user.ID, user.Email, http.StatusCreated)
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	var req types.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"message":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	user, err := s.store.GetUserByEmail(r.Context(), email)
	if err != nil || !auth.CheckPassword(user.PasswordHash, req.Password) {
		http.Error(w, `{"message":"invalid email or password"}`, http.StatusUnauthorized)
		return
	}

	s.issueToken(w, user.ID, user.Email, http.StatusOK)
}

func (s *Server) issueToken(w http.ResponseWriter, userID int64, email string, status int) {
	token, expiresAt, err := auth.GenerateToken(s.jwtSecret, s.jwtTTL, userID, email)
	if err != nil {
		http.Error(w, `{"message":"failed to issue token"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(types.AuthResponse{Token: token, ExpiresAt: expiresAt})
}
