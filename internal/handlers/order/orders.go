package order

import (
	"context"
	"errors"
	"go-pet-shop/internal/models"
	"go-pet-shop/internal/storage/postgres"
	"log/slog"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
)

//go:generate go run github.com/vektra/mockery/v2 --name=Orders
type Orders interface {
	CreateOrder(ctx context.Context, order models.Order) (models.Order, error)
	AddItemToOrder(ctx context.Context, orderItem models.OrderItem) (int, error)
	GetOrderById(ctx context.Context, id int) (models.Order, error)
	GetOrdersByEmail(ctx context.Context, email string) ([]models.Order, error)
	GetOrderItemsByOrderId(ctx context.Context, orderId int) ([]models.OrderItem, error)
}

type Handler struct {
	log     *slog.Logger
	storage Orders
}

func New(log *slog.Logger, storage Orders) *Handler {
	return &Handler{
		log:     log,
		storage: storage,
	}
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	const fn = "handler.orders.CreateOrder"
	log := h.log.With(
		slog.String("fn", fn),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	var order models.Order
	if err := render.DecodeJSON(r.Body, &order); err != nil {
		log.Error("failed to decode request body", slog.Any("error", err))
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Bad request",
			"message": "Invalid JSON payload",
		})
		return
	}

	if order.UserID <= 0 {
		log.Error("order user_id is not valid", slog.Int("user_id", order.UserID))
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Bad request",
			"message": "UserId cannot be negative",
		})
		return
	}

	if order.TotalPrice <= 0 {
		log.Error("order total price is negative", slog.Int("stock", int(order.TotalPrice)))
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Bad request",
			"message": "Total price cannot be negative",
		})
		return
	}

	order, err := h.storage.CreateOrder(r.Context(), order)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			log.Warn("user not found", slog.Int("user_id", order.UserID))
			w.WriteHeader(http.StatusNotFound)
			render.JSON(w, r, map[string]string{
				"error":   "Not found",
				"message": "User not found",
			})
			return
		}
		log.Error("failed to create order", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error":   "Internal server error",
			"message": "Failed to create order",
		})
		return
	}

	log.Info("Order created successfully",
		slog.Int("id", order.ID),
		slog.Int("total_price", int(order.TotalPrice)),
		slog.String("url", r.URL.String()),
	)

	w.WriteHeader(http.StatusCreated)
	render.JSON(w, r, map[string]interface{}{
		"status": "Order created successfully",
		"id":     order.ID,
		"order":  order,
	})
}

func (h *Handler) AddItemToOrder(w http.ResponseWriter, r *http.Request) {
	const fn = "handler.orders.AddItemToOrder"
	log := h.log.With(
		slog.String("fn", fn),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		log.Error("empty id")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Bad request",
			"message": "Order ID is required",
		})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Error("invalid id format", slog.Any("error", err), slog.String("id", idStr))
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Bad request",
			"message": "Order ID must be a number",
		})
		return
	}

	var orderItem models.OrderItem
	if err := render.DecodeJSON(r.Body, &orderItem); err != nil {
		log.Error("failed to decode request body", slog.Any("error", err))
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Bad request",
			"message": "Invalid JSON payload",
		})
		return
	}

	if id <= 0 {
		log.Error("order_item order_id is not valid", slog.Int("order_id", orderItem.OrderID))
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Bad request",
			"message": "OrderID cannot be negative",
		})
		return
	}

	if orderItem.ProductID <= 0 {
		log.Error("order_item product_id is not valid", slog.Int("product_id", orderItem.ProductID))
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Bad request",
			"message": "ProductID cannot be negative",
		})
		return
	}

	if orderItem.Quantity <= 0 {
		log.Error("order_item quantity is negative", slog.Int("quantity", orderItem.Quantity))
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Bad request",
			"message": "Quantity cannot be negative",
		})
		return
	}

	orderItem.OrderID = id

	itemID, err := h.storage.AddItemToOrder(r.Context(), orderItem)

	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			log.Warn("order or product not found", slog.Int("order_id", orderItem.OrderID), slog.Int("product_id", orderItem.ProductID))
			w.WriteHeader(http.StatusNotFound)
			render.JSON(w, r, map[string]string{
				"error":   "Not found",
				"message": "Order or product not found",
			})
			return
		}

		log.Error("failed to create order_item", slog.Any("error", err))
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error":   "Internal server error",
			"message": "Failed to create orderItem",
		})
		return
	}

	orderItem.ID = itemID

	log.Info("OrderItem created successfully",
		slog.Int("id", orderItem.ID),
		slog.Int("order_id", orderItem.OrderID),
		slog.Int("product_id", int(orderItem.ProductID)),
		slog.Int("quantity", int(orderItem.Quantity)),
		slog.String("url", r.URL.String()),
	)

	w.WriteHeader(http.StatusCreated)
	render.JSON(w, r, map[string]interface{}{
		"status":     "OrderItem created successfully",
		"order_item": orderItem,
	})
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	const fn = "handler.orders.GetOrder"
	log := h.log.With(
		slog.String("fn", fn),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		log.Error("empty id")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Bad request",
			"message": "Order ID is required",
		})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		log.Error("invalid id format", slog.Any("error", err), slog.String("id", idStr))
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Bad request",
			"message": "Order ID must be a number",
		})
		return
	}

	order, err := h.storage.GetOrderById(r.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			log.Warn("order not found", slog.Int("id", id))
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{
				"error":   "Not found",
				"message": "Order not found",
			})
			return
		}
		log.Error("failed to get order", slog.Any("error", err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error":   "Internal server error",
			"message": "Failed to get order",
		})
		return
	}

	items, err := h.storage.GetOrderItemsByOrderId(r.Context(), id)
	if err != nil {
		log.Error("failed to get order items", slog.Any("error", err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error":   "Internal server error",
			"message": "Failed to get order items",
		})
		return
	}
	response := struct {
		Order models.Order       `json:"order"`
		Items []models.OrderItem `json:"items"`
	}{
		Order: order,
		Items: items,
	}

	render.JSON(w, r, response)
}

func (h *Handler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	const fn = "handler.orders.GetUserOrders"
	log := h.log.With(
		slog.String("fn", fn),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	email := chi.URLParam(r, "email")
	email = strings.ToLower(strings.TrimSpace(email))

	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		log.Error("email is invalid")
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error":   "Bad request",
			"message": "Email must be valid",
		})
		return
	}

	orders, err := h.storage.GetOrdersByEmail(r.Context(), email)
	if err != nil {
		log.Error("failed to get orders", slog.Any("error", err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"error":   "Internal server error",
			"message": "Failed to get orders",
		})
		return
	}

	render.JSON(w, r, orders)
}
