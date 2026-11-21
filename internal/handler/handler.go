package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/auth"
	mw "github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/middleware"
	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/models"
	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/repository"
	"github.com/TimurBattalkhanov/go-musthave-diploma-tpl/internal/utils"
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
		r.Post("/api/user/orders", createOrder(store, log))
		r.Get("/api/user/orders", getOrders(store, log))
		r.Get("/api/user/balance", getBalance(store, log))
		r.Post("/api/user/balance/withdraw", withdraw(store, log))
		r.Get("/api/user/withdrawals", getWithdrawals(store, log))
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

func createOrder(store repository.Storage, log *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Debugw("Failed to read body", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		number := strings.TrimSpace(string(body))
		if number == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if !utils.ValidLuhn(number) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		login, ok := mw.LoginFrom(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		err = store.CreateOrder(r.Context(), number, login)
		if errors.Is(err, repository.ErrOrderIsAlreadyExists) {
			w.WriteHeader(http.StatusOK)
			return
		}
		if errors.Is(err, repository.ErrOrderCreatedByAnotherUser) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		if err != nil {
			log.Errorw("Failed to create order", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

func getOrders(store repository.Storage, log *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		login, ok := mw.LoginFrom(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		orders, err := store.GetOrdersByLogin(r.Context(), login)
		if err != nil {
			log.Errorw("Failed to get orders by login", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(orders) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err = json.NewEncoder(w).Encode(orders); err != nil {
			log.Errorw("Failed to encode response", "error", err)
		}
	}
}

func getBalance(store repository.Storage, log *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		login, ok := mw.LoginFrom(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		balance, err := store.GetBalance(r.Context(), login)
		if err != nil {
			log.Errorw("Failed to get balance", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err = json.NewEncoder(w).Encode(balance); err != nil {
			log.Errorw("Failed to encode response", "error", err)
		}
	}
}

func withdraw(store repository.Storage, log *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		login, ok := mw.LoginFrom(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var req models.Withdraw
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&req); err != nil {
			log.Debugw("Failed to decode request", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if !utils.ValidLuhn(req.OrderID) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		if req.Sum <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		err := store.Withdraw(r.Context(), login, req)
		if errors.Is(err, repository.ErrInsufficientFunds) {
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}
		if err != nil {
			log.Errorw("Failed to withdraw", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func getWithdrawals(store repository.Storage, log *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		login, ok := mw.LoginFrom(r.Context())
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		rows, err := store.GetWithdrawHistory(r.Context(), login)
		if err != nil {
			log.Errorw("Failed to get withdrawals", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if len(rows) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err = json.NewEncoder(w).Encode(rows); err != nil {
			log.Errorw("Failed to encode response", "error", err)
		}
	}
}
