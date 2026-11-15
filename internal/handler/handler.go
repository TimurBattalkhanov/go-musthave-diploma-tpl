package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/auth"
	mw "github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/middleware"
	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/models"
	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func NewRouter(store repository.Storage, secret []byte, log *zap.SugaredLogger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.StripSlashes)
	r.Use(mw.HTTPLogging(log))
	r.Post("/api/user/register", registerHandler(store, secret, log))
	r.Post("/api/user/login", loginHandler(store, secret, log))
	r.Group(func(r chi.Router) {
		r.Use(mw.Auth(secret))
	})
	return r
}

func registerHandler(store repository.Storage, secret []byte, log *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.UserRequest
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			log.Debugw("Failed to decode request", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if req.Password == "" || req.Login == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Errorw("Failed to generate bcrypt hash", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = store.RegisterUser(r.Context(), req.Login, string(hash))
		if errors.Is(err, repository.ErrLoginTaken) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		if err != nil {
			log.Errorw("Failed to register user", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = auth.IssueToken(w, req.Login, secret)
		if err != nil {
			log.Errorw("Failed to issue token", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func loginHandler(store repository.Storage, secret []byte, log *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.UserRequest
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			log.Debugw("Failed to decode request", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if req.Password == "" || req.Login == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		passwordHash, err := store.GetPasswordHash(r.Context(), req.Login)
		if errors.Is(err, repository.ErrUserNotFound) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err != nil {
			log.Errorw("Failed to login user", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		err = auth.IssueToken(w, req.Login, secret)
		if err != nil {
			log.Errorw("Failed to issue token", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
