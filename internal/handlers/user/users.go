package user

import (
	"context"
	"errors"
	"go-pet-shop/internal/models"
	"go-pet-shop/internal/storage/postgres"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

//go:generate go run github.com/vektra/mockery/v2 --name=Users
type Users interface {
	CreateUser(ctx context.Context, u models.User) (int, error)
	GetUserByEmail(ctx context.Context, email string) (models.User, error)
	GetAllUsers(ctx context.Context) ([]models.User, error)
}

type Handler struct {
	log     *slog.Logger
	storage Users
}

func New(log *slog.Logger, storage Users) *Handler {
	return &Handler{
		log:     log,
		storage: storage,
	}
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	const fn = "handlers.users.CreateUser"

	log := h.log.With(
		slog.String("fn", fn),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	var user models.User
	if err := render.DecodeJSON(r.Body, &user); err != nil {
		log.Error("failed to decode request body", slog.Any("error", err))
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Bad request",
			"message": "Invalid JSON payload",
		})
		return
	}

	if user.Name == "" {
		log.Error("user name is empty")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Bad request",
			"message": "User name is required",
		})
		return
	}

	user.Email = strings.ToLower(strings.TrimSpace(user.Email))

	addr, err := mail.ParseAddress(user.Email)
	if err != nil || addr.Address != user.Email {
		log.Error("email is invalid")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Bad request",
			"message": "Email must be valid",
		})
		return
	}

	userID, err := h.storage.CreateUser(r.Context(), user)
	if err != nil {
		log.Error("failed to create user", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error":   "Internal server error",
			"message": "Failed to create user",
		})
		return
	}

	log.Info("User created successfully",
		slog.Int("id", userID),
		slog.String("name", user.Name),
		slog.String("email", user.Email),
	)

	user.ID = userID
	w.WriteHeader(http.StatusCreated)
	render.JSON(w, r, map[string]interface{}{
		"status": "User created successfully",
		"id":     userID,
		"user":   user,
	})
}

func (h *Handler) GetUserByEmail(w http.ResponseWriter, r *http.Request) {
	const fn = "handlers.users.GetUserByEmail"

	log := h.log.With(
		slog.String("fn", fn),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	email := r.URL.Query().Get("email")
	email = strings.ToLower(strings.TrimSpace(email))

	if email == "" {
		log.Error("empty email")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Bad request",
			"message": "User email is required",
		})
		return
	}

	user, err := h.storage.GetUserByEmail(r.Context(), email)

	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			log.Warn("user not found", slog.String("email", email))
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{
				"error":   "Not found",
				"message": "User not found",
			})
			return
		}
		log.Error("failed to get user", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error":   "Internal server error",
			"message": "Failed to get user",
		})
		return
	}

	log.Info("Retrieved user successfully",
		slog.String("url", r.URL.String()),
	)

	render.JSON(w, r, user)
}

func (h *Handler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	const fn = "handlers.users.GetAllUsers"

	log := h.log.With(
		slog.String("fn", fn),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	users, err := h.storage.GetAllUsers(r.Context())

	if err != nil {
		log.Error("failed to get users", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error":   "Internal server error",
			"message": "Failed to get users",
		})
		return
	}

	log.Info("Retrieved users successfully",
		slog.String("url", r.URL.String()),
	)

	render.JSON(w, r, users)
}
