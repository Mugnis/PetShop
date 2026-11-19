package user

import (
	"encoding/json"
	"errors"
	"go-pet-shop/internal/handlers/user/mocks"
	"go-pet-shop/internal/models"
	"go-pet-shop/internal/storage/postgres"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// =======================
// Get All Users
// =======================

func TestGetAllUsers_Success(t *testing.T) {
	UsersGetterMock := mocks.NewUsers(t)

	UsersGetterMock.On("GetAllUsers", mock.Anything).
		Return([]models.User{{ID: 1, Name: "Mugnis", Email: "someone@example.com"}}, nil).
		Once()

	// Создаем запрос
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	handler := New(slog.Default(), UsersGetterMock)
	handler.GetAllUsers(w, req)

	// Ожидаем HTTP 200
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestGetAllUsers_Error(t *testing.T) {
	UsersGetterMock := mocks.NewUsers(t)

	UsersGetterMock.On("GetAllUsers", mock.Anything).
		Return(nil, errors.New("DB Error")).
		Once()

	// Создаем запрос
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	handler := New(slog.Default(), UsersGetterMock)
	handler.GetAllUsers(w, req)

	// Ожидаем HTTP 500
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// =======================
// Get User By Email
// =======================

func TestGetUserByEmail_Success(t *testing.T) {
	UserGetterMock := mocks.NewUsers(t)

	UserGetterMock.On("GetUserByEmail", mock.Anything, "someone@example.com").
		Return(models.User{ID: 1, Name: "Mugnis", Email: "someone@example.com"}, nil).
		Once()

	// Создаем запрос
	req := httptest.NewRequest(http.MethodGet, "/users/by-email?email=Someone@Example.com", nil)
	w := httptest.NewRecorder()

	handler := New(slog.Default(), UserGetterMock)
	handler.GetUserByEmail(w, req)

	// Ожидаем HTTP 200
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var got models.User
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, models.User{ID: 1, Name: "Mugnis", Email: "someone@example.com"}, got)
}

func TestGetUserByEmail_EmptyEmail(t *testing.T) {
	UserGetterMock := mocks.NewUsers(t)

	// Создаем запрос
	req := httptest.NewRequest(http.MethodGet, "/users/by-email", nil)
	w := httptest.NewRecorder()

	handler := New(slog.Default(), UserGetterMock)
	handler.GetUserByEmail(w, req)

	// Ожидаем HTTP 400
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	UserGetterMock := mocks.NewUsers(t)

	UserGetterMock.On("GetUserByEmail", mock.Anything, "someone@example.com").
		Return(models.User{}, postgres.ErrNotFound).
		Once()

	// Создаем запрос
	req := httptest.NewRequest(http.MethodGet, "/users/by-email?email=Someone@Example.com", nil)
	w := httptest.NewRecorder()

	handler := New(slog.Default(), UserGetterMock)
	handler.GetUserByEmail(w, req)

	// Ожидаем HTTP 404
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetUserByEmail_Error(t *testing.T) {
	UserGetterMock := mocks.NewUsers(t)

	UserGetterMock.On("GetUserByEmail", mock.Anything, "someone@example.com").
		Return(models.User{}, errors.New("DB Error")).
		Once()

	// Создаем запрос
	req := httptest.NewRequest(http.MethodGet, "/users/by-email?email=Someone@Example.com", nil)
	w := httptest.NewRecorder()

	handler := New(slog.Default(), UserGetterMock)
	handler.GetUserByEmail(w, req)

	// Ожидаем HTTP 500
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// =======================
// Create User
// =======================

func TestCreateUser_Success(t *testing.T) {
	UserCreatorMock := mocks.NewUsers(t)

	UserCreatorMock.On("CreateUser", mock.Anything, models.User{Name: "Mugnis", Email: "someone@example.com"}).
		Return(1, nil).
		Once()

	// Создаем запрос
	body := `{"name": "Mugnis", "email": "someone@Example.com"}`

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler := New(slog.Default(), UserCreatorMock)
	handler.CreateUser(w, req)

	// Ожидаем HTTP 201
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var resp struct {
		Status string      `json:"status"`
		ID     int         `json:"id"`
		User   models.User `json:"user"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Equal(t, models.User{ID: 1, Name: "Mugnis", Email: "someone@example.com"}, resp.User)
}

func TestCreateUser_InvalidJSON(t *testing.T) {
	UserCreatorMock := mocks.NewUsers(t)

	// Создаем запрос
	body := `{"nameugns", "ema omeone@Exme.com"`

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler := New(slog.Default(), UserCreatorMock)
	handler.CreateUser(w, req)

	// Ожидаем HTTP 400
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateUser_EmptyName(t *testing.T) {
	UserCreatorMock := mocks.NewUsers(t)

	// Создаем запрос
	body := `{"name": "", "email": "someone@Example.com"}`

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler := New(slog.Default(), UserCreatorMock)
	handler.CreateUser(w, req)

	// Ожидаем HTTP 400
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateUser_InvalidEmail(t *testing.T) {
	UserCreatorMock := mocks.NewUsers(t)

	// Создаем запрос
	body := `{"name": "Mugnis", "email": "someoneExample.com"}`

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler := New(slog.Default(), UserCreatorMock)
	handler.CreateUser(w, req)

	// Ожидаем HTTP 400
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateUser_Error(t *testing.T) {
	UserCreatorMock := mocks.NewUsers(t)

	UserCreatorMock.On("CreateUser", mock.Anything, models.User{Name: "Mugnis", Email: "someone@example.com"}).
		Return(0, errors.New("DB Error")).
		Once()

	// Создаем запрос
	body := `{"name": "Mugnis", "email": "someone@Example.com"}`

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	w := httptest.NewRecorder()

	handler := New(slog.Default(), UserCreatorMock)
	handler.CreateUser(w, req)

	// Ожидаем HTTP 500
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
