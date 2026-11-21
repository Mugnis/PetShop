package product

import (
	"encoding/json"
	"errors"
	"go-pet-shop/internal/handlers/product/mocks"
	"go-pet-shop/internal/models"
	"go-pet-shop/internal/storage/postgres"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Get Product - Ready
func TestGetAllProducts_Success(t *testing.T) {
	// Мокаем storage — он вернёт один продукт.
	ProductGetterMock := mocks.NewProducts(t)

	ProductGetterMock.On("GetAllProducts", mock.Anything).
		Return([]models.Product{{ID: 1, Name: "Dog Food"}}, nil).
		Once()

	// Создаем HTTP-запрос GET /products
	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	w := httptest.NewRecorder()

	// Создаем хендлер с мок-хранилищем
	handler := New(slog.Default(), ProductGetterMock)

	// Вызываем метод GetAllProducts, который является http.HandlerFunc
	handler.GetAllProducts(w, req)

	// Проверяем HTTP-код
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}
func TestGetAllProducts_Error(t *testing.T) {
	// Мокаем storage — он будет возвращать ошибку
	ProductGetterMock := mocks.NewProducts(t)

	ProductGetterMock.On("GetAllProducts", mock.Anything).
		Return(nil, errors.New("DB Error")).
		Once()

	// Создаем запрос
	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	w := httptest.NewRecorder()

	handler := New(slog.Default(), ProductGetterMock)
	handler.GetAllProducts(w, req)

	// Ожидаем HTTP 500
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// =======================
// Get Product By Id
// =======================

func TestGetProductById_Success(t *testing.T) {
	// Мокаем storage — он вернёт один продукт.
	ProductGetterMock := mocks.NewProducts(t)

	ProductGetterMock.On("GetProductById", mock.Anything, 1).
		Return(models.Product{ID: 1, Name: "Dog Food"}, nil).
		Once()

	// Создаем HTTP-запрос GET /products
	req := httptest.NewRequest(http.MethodGet, "/products/1", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), ProductGetterMock)

	router.Get("/products/{id}", handler.GetProductById)

	router.ServeHTTP(w, req)

	// Проверяем HTTP-код
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var got models.Product
	require.NoError(t, json.NewDecoder(w.Body).Decode(&got))
	require.Equal(t, models.Product{ID: 1, Name: "Dog Food"}, got)
}

func TestGetProductById_BadRequest(t *testing.T) {
	// Мокаем storage — он вернёт один продукт.
	ProductGetterMock := mocks.NewProducts(t)

	// Создаем HTTP-запрос GET /products
	req := httptest.NewRequest(http.MethodGet, "/products/abc", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), ProductGetterMock)

	router.Get("/products/{id}", handler.GetProductById)

	router.ServeHTTP(w, req)

	// Проверяем HTTP-код
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
}

func TestGetProductById_NotFound(t *testing.T) {
	// Мокаем storage — он вернёт один продукт.
	ProductGetterMock := mocks.NewProducts(t)

	ProductGetterMock.On("GetProductById", mock.Anything, 1).
		Return(models.Product{}, postgres.ErrNotFound).
		Once()

	// Создаем HTTP-запрос GET /products
	req := httptest.NewRequest(http.MethodGet, "/products/1", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), ProductGetterMock)

	router.Get("/products/{id}", handler.GetProductById)

	router.ServeHTTP(w, req)

	// Проверяем HTTP-код
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestGetProductById_Error(t *testing.T) {
	// Мокаем storage — он вернёт один продукт.
	ProductGetterMock := mocks.NewProducts(t)

	ProductGetterMock.On("GetProductById", mock.Anything, 1).
		Return(models.Product{}, errors.New("DB Error")).
		Once()

	// Создаем HTTP-запрос GET /products
	req := httptest.NewRequest(http.MethodGet, "/products/1", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), ProductGetterMock)

	router.Get("/products/{id}", handler.GetProductById)

	router.ServeHTTP(w, req)

	// Проверяем HTTP-код
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

// =======================
// Create Product
// =======================

func TestCreateProduct_Success(t *testing.T) {
	ProductCreatingMock := mocks.NewProducts(t)

	product := &models.Product{}
	product.Name = "Корм для собак"
	product.Price = 499.99
	product.Stock = 10

	ProductCreatingMock.On("CreateProduct", mock.Anything, *product).
		Return(1, nil).
		Once()

	requestBody := `{"name": "Корм для собак", "price": 499.99, "stock": 10}`

	req := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	handler := New(slog.Default(), ProductCreatingMock)
	handler.CreateProduct(w, req)

	// Ожидаем HTTP 201
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if response["status"] != "Product created successfully" {
		t.Errorf("unexpected status string: %v", response["status"])
	}

	if response["id"] != float64(1) {
		t.Errorf("unexpected id: %v", response["id"])
	}
}

func TestCreateProduct_BadRequest(t *testing.T) {
	ProductCreatingMock := mocks.NewProducts(t)

	requestBody := `"na: "Корля соак", "price 499.99, "sto"0}`

	req := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	handler := New(slog.Default(), ProductCreatingMock)
	handler.CreateProduct(w, req)

	// Ожидаем HTTP 400
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if response["error"] != "Bad request" {
		t.Errorf("unexpected status string: %v", response["status"])
	}
}

func TestCreateProduct_Fail(t *testing.T) {
	ProductCreatingMock := mocks.NewProducts(t)

	product := &models.Product{}
	product.Name = "Корм для собак"
	product.Price = 499.99
	product.Stock = 10

	ProductCreatingMock.On("CreateProduct", mock.Anything, *product).
		Return(0, errors.New("DB Error")).
		Once()

	requestBody := `{"name": "Корм для собак", "price": 499.99, "stock": 10}`

	req := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	handler := New(slog.Default(), ProductCreatingMock)
	handler.CreateProduct(w, req)

	// Ожидаем HTTP 500
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if response["error"] != "Internal server error" {
		t.Errorf("unexpected error: %v", response["status"])
	}
}

// =======================
// Update Product
// =======================

func TestUpdateProduct_Success(t *testing.T) {
	ProductCreatingMock := mocks.NewProducts(t)

	product := &models.Product{}
	product.ID = 1
	product.Name = "Корм для собак"
	product.Price = 499.99
	product.Stock = 10

	ProductCreatingMock.On("UpdateProduct", mock.Anything, *product).
		Return(nil).
		Once()

	requestBody := `{"id": 10, "name": "Корм для собак", "price": 499.99, "stock": 10}`

	req := httptest.NewRequest(http.MethodPost, "/products/1", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), ProductCreatingMock)

	router.Post("/products/{id}", handler.UpdateProduct)

	router.ServeHTTP(w, req)

	// Ожидаем HTTP 200
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if response["status"] != "Product updated successfully" {
		t.Errorf("unexpected status string: %v", response["status"])
	}

	if response["id"] != float64(1) {
		t.Errorf("unexpected id: %v", response["id"])
	}
}

func TestUpdateProduct_BadRequest(t *testing.T) {
	ProductCreatingMock := mocks.NewProducts(t)

	requestBody := `d": 10, "name": "Корм для собак", "ice": 499.99, "stoc: 10}`

	req := httptest.NewRequest(http.MethodPost, "/products/1", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), ProductCreatingMock)

	router.Post("/products/{id}", handler.UpdateProduct)

	router.ServeHTTP(w, req)

	// Ожидаем HTTP 400
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if response["error"] != "Bad request" {
		t.Errorf("unexpected error: %v", response["status"])
	}
}

func TestUpdateProduct_Fail(t *testing.T) {
	ProductCreatingMock := mocks.NewProducts(t)

	product := &models.Product{}
	product.ID = 1
	product.Name = "Корм для собак"
	product.Price = 499.99
	product.Stock = 10

	ProductCreatingMock.On("UpdateProduct", mock.Anything, *product).
		Return(errors.New("DB Error")).
		Once()

	requestBody := `{"id": 10, "name": "Корм для собак", "price": 499.99, "stock": 10}`

	req := httptest.NewRequest(http.MethodPost, "/products/1", strings.NewReader(requestBody))
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), ProductCreatingMock)

	router.Post("/products/{id}", handler.UpdateProduct)

	router.ServeHTTP(w, req)

	// Ожидаем HTTP 500
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if response["error"] != "Internal server error" {
		t.Errorf("unexpected error: %v", response["status"])
	}
}

// =======================
// Delete Product
// =======================

func TestDeleteProduct_Success(t *testing.T) {
	ProductDeleterMock := mocks.NewProducts(t)

	ProductDeleterMock.On("DeleteProduct", mock.Anything, 1).
		Return(nil).
		Once()

	req := httptest.NewRequest(http.MethodDelete, "/products/1", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), ProductDeleterMock)

	router.Delete("/products/{id}", handler.DeleteProduct)

	router.ServeHTTP(w, req)

	// Ожидаем HTTP 200
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if response["status"] != "Product deleted successfully" {
		t.Errorf("unexpected status string: %v", response["status"])
	}

	if response["id"] != float64(1) {
		t.Errorf("unexpected id: %v", response["id"])
	}
}

func TestDeleteProduct_BadRequest(t *testing.T) {
	ProductDeleterMock := mocks.NewProducts(t)

	req := httptest.NewRequest(http.MethodDelete, "/products/", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), ProductDeleterMock)

	router.Delete("/products/{id}", handler.DeleteProduct)
	router.Delete("/products/", handler.DeleteProduct)

	router.ServeHTTP(w, req)

	// Ожидаем HTTP 400
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if response["error"] != "Bad request" {
		t.Errorf("unexpected error: %v", response["status"])
	}
}

func TestDeleteProduct_Fail(t *testing.T) {
	ProductDeleterMock := mocks.NewProducts(t)

	ProductDeleterMock.On("DeleteProduct", mock.Anything, 1).
		Return(errors.New("DB Error")).
		Once()

	req := httptest.NewRequest(http.MethodDelete, "/products/1", nil)
	w := httptest.NewRecorder()

	router := chi.NewRouter()
	handler := New(slog.Default(), ProductDeleterMock)

	router.Delete("/products/{id}", handler.DeleteProduct)

	router.ServeHTTP(w, req)

	// Ожидаем HTTP 500
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if response["error"] != "Internal server error" {
		t.Errorf("unexpected error: %v", response["status"])
	}
}

// =======================
// Get Popular Products
// =======================

func TestGetPopularProducts_Success(t *testing.T) {
	PopularProductsGetter := mocks.NewProducts(t)

	popularProducts := []models.PopularProduct{
		{
			ID:         1,
			Name:       "name",
			Price:      1,
			TotalSales: 1,
		},
		{
			ID:         2,
			Name:       "name",
			Price:      2,
			TotalSales: 2,
		},
	}

	PopularProductsGetter.On("GetPopularProducts", mock.Anything).
		Return(popularProducts, nil).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/products/popular", nil)
	w := httptest.NewRecorder()

	handler := New(slog.Default(), PopularProductsGetter)
	handler.GetPopularProducts(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var gotPopularProducts []models.PopularProduct
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &gotPopularProducts))
	require.Equal(t, popularProducts, gotPopularProducts)
}

func TestGetPopularProducts_Error(t *testing.T) {
	PopularProductsGetter := mocks.NewProducts(t)

	PopularProductsGetter.On("GetPopularProducts", mock.Anything).
		Return(nil, errors.New("DB Error")).
		Once()

	req := httptest.NewRequest(http.MethodGet, "/products/popular", nil)
	w := httptest.NewRecorder()

	handler := New(slog.Default(), PopularProductsGetter)
	handler.GetPopularProducts(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var response struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Equal(t, "Internal server error", response.Error)
	require.Equal(t, "Failed to get popular products", response.Message)
}