package order

import (
	"encoding/json"
	"errors"
	"go-pet-shop/internal/handlers/order/mocks"
	"go-pet-shop/internal/models"
	"go-pet-shop/internal/storage/postgres"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// =======================
// Create Order
// =======================

func TestCreateOrder_Success(t *testing.T) {
	OrderCreatingMock := mocks.NewOrders(t)

	inputOrder := models.Order{
		UserID:     1,
		TotalPrice: 100,
	}

	createdOrder := models.Order{
		ID:         1,
		UserID:     1,
		TotalPrice: 100,
		CreatedAt:  time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC),
	}

	OrderCreatingMock.On("CreateOrder", mock.Anything, inputOrder).
		Return(createdOrder, nil).
		Once()

	requestBody := `{"userId": 1, "totalPrice": 100}`

	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	handler := New(slog.Default(), OrderCreatingMock)
	handler.CreateOrder(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var response struct {
		Status string       `json:"status"`
		ID     int          `json:"id"`
		Order  models.Order `json:"order"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Order created successfully", response.Status)
	require.Equal(t, createdOrder.ID, response.ID)
	require.Equal(t, createdOrder, response.Order)
}

func TestCreateOrder_InvalidJSON(t *testing.T) {
	OrderCreatingMock := mocks.NewOrders(t)

	requestBody := `{usId": "talPrice":1`

	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	handler := New(slog.Default(), OrderCreatingMock)
	handler.CreateOrder(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Bad request", response.Error)
	require.Equal(t, "Invalid JSON payload", response.Message)
}

func TestCreateOrder_InvalidUserID(t *testing.T) {
	OrderCreatingMock := mocks.NewOrders(t)

	requestBody := `{"userId": 0, "totalPrice": 100}`

	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	handler := New(slog.Default(), OrderCreatingMock)
	handler.CreateOrder(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Bad request", response.Error)
	require.Equal(t, "UserId cannot be negative", response.Message)
}

func TestCreateOrder_InvalidTotalPrice(t *testing.T) {
	OrderCreatingMock := mocks.NewOrders(t)

	requestBody := `{"userId": 1, "totalPrice": 0}`

	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	handler := New(slog.Default(), OrderCreatingMock)
	handler.CreateOrder(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Bad request", response.Error)
	require.Equal(t, "Total price cannot be negative", response.Message)
}

func TestCreateOrder_NotFound(t *testing.T) {
	OrderCreatingMock := mocks.NewOrders(t)

	inputOrder := models.Order{
		UserID:     1,
		TotalPrice: 100,
	}

	OrderCreatingMock.On("CreateOrder", mock.Anything, inputOrder).
		Return(models.Order{}, postgres.ErrNotFound).
		Once()

	requestBody := `{"userId": 1, "totalPrice": 100}`

	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	handler := New(slog.Default(), OrderCreatingMock)
	handler.CreateOrder(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Not found", response.Error)
	require.Equal(t, "User not found", response.Message)
}

func TestCreateOrder_Error(t *testing.T) {
	OrderCreatingMock := mocks.NewOrders(t)

	inputOrder := models.Order{
		UserID:     1,
		TotalPrice: 100,
	}

	OrderCreatingMock.On("CreateOrder", mock.Anything, inputOrder).
		Return(models.Order{}, errors.New("DB Error")).
		Once()

	requestBody := `{"userId": 1, "totalPrice": 100}`

	req := httptest.NewRequest(http.MethodPost, "/orders", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	handler := New(slog.Default(), OrderCreatingMock)
	handler.CreateOrder(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Internal server error", response.Error)
	require.Equal(t, "Failed to create order", response.Message)
}

// =======================
// Add Item To Order
// =======================

func TestAddItemToOrder_Success(t *testing.T) {
	ItemAdderMock := mocks.NewOrders(t)

	OrderItem := models.OrderItem{
		OrderID:   1,
		ProductID: 1,
		Quantity:  1,
	}

	ItemAdderMock.On("AddItemToOrder", mock.Anything, OrderItem).
		Return(1, nil).
		Once()

	requestBody := `{"orderID": 1, "productID": 1, "quantity": 1}`

	req := httptest.NewRequest(http.MethodPost, "/orders/1/items", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), ItemAdderMock)

	router.Post("/orders/{id}/items", handler.AddItemToOrder)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	OrderItem.ID = 1
	var response struct {
		Status    string           `json:"status"`
		OrderItem models.OrderItem `json:"order_item"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "OrderItem created successfully", response.Status)
	require.Equal(t, OrderItem, response.OrderItem)
}

func TestAddItemToOrder_InvalidOrderID(t *testing.T) {
	ItemAdderMock := mocks.NewOrders(t)

	requestBody := `{"orderID": 0, "productID": 1, "quantity": 1}`

	req := httptest.NewRequest(http.MethodPost, "/orders/0/items", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), ItemAdderMock)

	router.Post("/orders/{id}/items", handler.AddItemToOrder)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Bad request", response.Error)
	require.Equal(t, "OrderID cannot be negative", response.Message)
}

func TestAddItemToOrder_InvalidJSON(t *testing.T) {
	ItemAdderMock := mocks.NewOrders(t)

	requestBody := `{"orerD": pductI: 1, "quant: `

	req := httptest.NewRequest(http.MethodPost, "/orders/0/items", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), ItemAdderMock)

	router.Post("/orders/{id}/items", handler.AddItemToOrder)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Bad request", response.Error)
	require.Equal(t, "Invalid JSON payload", response.Message)
}

func TestAddItemToOrder_InvalidProductID(t *testing.T) {
	ItemAdderMock := mocks.NewOrders(t)

	requestBody := `{"orderID": 1, "productID": 0, "quantity": 1}`

	req := httptest.NewRequest(http.MethodPost, "/orders/1/items", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), ItemAdderMock)

	router.Post("/orders/{id}/items", handler.AddItemToOrder)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Bad request", response.Error)
	require.Equal(t, "ProductID cannot be negative", response.Message)
}

func TestAddItemToOrder_InvalidQuantity(t *testing.T) {
	ItemAdderMock := mocks.NewOrders(t)

	requestBody := `{"orderID": 1, "productID": 1, "quantity": 0}`

	req := httptest.NewRequest(http.MethodPost, "/orders/1/items", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), ItemAdderMock)

	router.Post("/orders/{id}/items", handler.AddItemToOrder)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Bad request", response.Error)
	require.Equal(t, "Quantity cannot be negative", response.Message)
}

func TestAddItemToOrder_NotFound(t *testing.T) {
	ItemAdderMock := mocks.NewOrders(t)

	OrderItem := models.OrderItem{
		OrderID:   1,
		ProductID: 1,
		Quantity:  1,
	}

	ItemAdderMock.On("AddItemToOrder", mock.Anything, OrderItem).
		Return(0, postgres.ErrNotFound).
		Once()

	requestBody := `{"orderID": 1, "productID": 1, "quantity": 1}`

	req := httptest.NewRequest(http.MethodPost, "/orders/1/items", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), ItemAdderMock)

	router.Post("/orders/{id}/items", handler.AddItemToOrder)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Not found", response.Error)
	require.Equal(t, "Order or product not found", response.Message)
}

func TestAddItemToOrder_Error(t *testing.T) {
	ItemAdderMock := mocks.NewOrders(t)

	OrderItem := models.OrderItem{
		OrderID:   1,
		ProductID: 1,
		Quantity:  1,
	}

	ItemAdderMock.On("AddItemToOrder", mock.Anything, OrderItem).
		Return(0, errors.New("DB Error")).
		Once()

	requestBody := `{"orderID": 1, "productID": 1, "quantity": 1}`

	req := httptest.NewRequest(http.MethodPost, "/orders/1/items", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), ItemAdderMock)

	router.Post("/orders/{id}/items", handler.AddItemToOrder)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Internal server error", response.Error)
	require.Equal(t, "Failed to create orderItem", response.Message)
}

// =======================
// Get Order
// =======================

func TestGetOrder_Success(t *testing.T) {
	OrderGetterMock := mocks.NewOrders(t)

	order := models.Order{
		ID:         1,
		UserID:     1,
		TotalPrice: 1,
		CreatedAt:  time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC),
	}

	orderItems := []models.OrderItem{
		{
			ID:        1,
			OrderID:   1,
			ProductID: 1,
			Quantity:  1,
		},
	}

	OrderGetterMock.On("GetOrderById", mock.Anything, 1).
		Return(order, nil).
		Once()

	OrderGetterMock.On("GetOrderItemsByOrderId", mock.Anything, 1).
		Return(orderItems, nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/orders/1", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), OrderGetterMock)

	router.Get("/orders/{id}", handler.GetOrder)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Order models.Order       `json:"order"`
		Items []models.OrderItem `json:"items"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, order, response.Order)
	require.Equal(t, orderItems, response.Items)
}

func TestGetOrder_InvalidID(t *testing.T) {
	OrderGetterMock := mocks.NewOrders(t)

	req := httptest.NewRequest(http.MethodGet, "/orders/abc", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), OrderGetterMock)

	router.Get("/orders/{id}", handler.GetOrder)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Bad request", response.Error)
	require.Equal(t, "Order ID must be a number", response.Message)
}

func TestGetOrder_NotFound(t *testing.T) {
	OrderGetterMock := mocks.NewOrders(t)

	OrderGetterMock.On("GetOrderById", mock.Anything, 1).
		Return(models.Order{}, postgres.ErrNotFound).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/orders/1", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), OrderGetterMock)

	router.Get("/orders/{id}", handler.GetOrder)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Not found", response.Error)
	require.Equal(t, "Order not found", response.Message)
}

func TestGetOrder_Error1(t *testing.T) {
	OrderGetterMock := mocks.NewOrders(t)

	OrderGetterMock.On("GetOrderById", mock.Anything, 1).
		Return(models.Order{}, errors.New("DB Error")).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/orders/1", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), OrderGetterMock)

	router.Get("/orders/{id}", handler.GetOrder)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Internal server error", response.Error)
	require.Equal(t, "Failed to get order", response.Message)
}

func TestGetOrder_Error2(t *testing.T) {
	OrderGetterMock := mocks.NewOrders(t)

	order := models.Order{
		ID:         1,
		UserID:     1,
		TotalPrice: 1,
		CreatedAt:  time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC),
	}

	OrderGetterMock.On("GetOrderById", mock.Anything, 1).
		Return(order, nil).
		Once()

	OrderGetterMock.On("GetOrderItemsByOrderId", mock.Anything, 1).
		Return([]models.OrderItem{}, errors.New("DB Error")).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/orders/1", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), OrderGetterMock)

	router.Get("/orders/{id}", handler.GetOrder)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Internal server error", response.Error)
	require.Equal(t, "Failed to get order items", response.Message)
}

// =======================
// Get User Orders
// =======================

func TestGetUserOrders_Success(t *testing.T) {
	OrderGetterMock := mocks.NewOrders(t)

	orders := []models.Order{
		{
			ID:         1,
			UserID:     1,
			TotalPrice: 1,
			CreatedAt:  time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC),
		},
	}

	OrderGetterMock.On("GetOrdersByEmail", mock.Anything, "someone@example.com").
		Return(orders, nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/users/someone@example.com/orders", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), OrderGetterMock)

	router.Get("/users/{email}/orders", handler.GetUserOrders)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var gotOrders []models.Order
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &gotOrders))
	require.Equal(t, orders, gotOrders)
}

func TestGetUserOrders_InvalidEmail(t *testing.T) {
	OrderGetterMock := mocks.NewOrders(t)

	req := httptest.NewRequest(http.MethodGet, "/users/someoneexample.com/orders", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), OrderGetterMock)

	router.Get("/users/{email}/orders", handler.GetUserOrders)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Bad request", response.Error)
	require.Equal(t, "Email must be valid", response.Message)
}

func TestGetUserOrders_Error(t *testing.T) {
	OrderGetterMock := mocks.NewOrders(t)

	OrderGetterMock.On("GetOrdersByEmail", mock.Anything, "someone@example.com").
		Return(nil, errors.New("DB Error")).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/users/someone@example.com/orders", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), OrderGetterMock)

	router.Get("/users/{email}/orders", handler.GetUserOrders)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Internal server error", response.Error)
	require.Equal(t, "Failed to get orders", response.Message)
}
