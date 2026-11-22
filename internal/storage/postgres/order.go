package postgres

import (
	"context"
	"errors"
	"fmt"
	"go-pet-shop/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Storage) CreateOrder(ctx context.Context, o models.Order) (models.Order, error) {
	const fn = "storage.postgres.order.CreateOrder"

	err := s.db.QueryRow(ctx,
		`INSERT INTO orders (user_id, total_price) VALUES ($1, $2) RETURNING id, created_at`,
		o.UserID, o.TotalPrice).
		Scan(&o.ID, &o.CreatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return models.Order{}, fmt.Errorf("%s: %w", fn, ErrNotFound)
		}
		return models.Order{}, fmt.Errorf("%s: %w", fn, err)
	}

	return o, nil
}

func (s *Storage) AddItemToOrder(ctx context.Context, o models.OrderItem) (int, error) {
	const fn = "storage.postgres.order.AddItemToOrder"

	var id int
	err := s.db.QueryRow(ctx,
		`INSERT INTO order_items (order_id, product_id, quantity) VALUES ($1, $2, $3) RETURNING id`,
		o.OrderID, o.ProductID, o.Quantity).Scan(&id)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return 0, fmt.Errorf("%s: %w", fn, ErrNotFound)
		}
		return 0, fmt.Errorf("%s: %w", fn, err)
	}

	return id, nil
}

func (s *Storage) GetOrderById(ctx context.Context, id int) (models.Order, error) {
	const fn = "storage.postgres.order.GetOrderById"

	var order models.Order
	err := s.db.QueryRow(ctx,
		`SELECT id, user_id, total_price, created_at FROM orders WHERE id = $1`,
		id).Scan(&order.ID, &order.UserID, &order.TotalPrice, &order.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Order{}, fmt.Errorf("%s: %w", fn, ErrNotFound)
		}
		return models.Order{}, fmt.Errorf("%s: %w", fn, err)
	}

	return order, nil
}

func (s *Storage) GetOrdersByEmail(ctx context.Context, email string) ([]models.Order, error) {
	const fn = "storage.postgres.order.GetOrdersByEmail"

	orders := make([]models.Order, 0)
	rows, err := s.db.Query(ctx,
		`SELECT o.id, o.user_id, o.total_price, o.created_at FROM orders o JOIN users u ON o.user_id = u.id WHERE u.email = $1 ORDER BY o.created_at DESC`,
		email)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", fn, err)
	}
	defer rows.Close()

	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.TotalPrice, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("%s: %w", fn, err)
		}
		orders = append(orders, o)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", fn, err)
	}

	return orders, nil
}

func (s *Storage) GetOrderItemsByOrderId(ctx context.Context, orderId int) ([]models.OrderItem, error) {
	const fn = "storage.postgres.order.GetOrderItemsByOrderId"

	orderItems := make([]models.OrderItem, 0)

	rows, err := s.db.Query(ctx, `SELECT id, order_id, product_id, quantity FROM order_items WHERE order_id = $1`, orderId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", fn, err)
	}
	defer rows.Close()

	for rows.Next() {
		var orderItem models.OrderItem
		if err := rows.Scan(&orderItem.ID, &orderItem.OrderID, &orderItem.ProductID, &orderItem.Quantity); err != nil {
			return nil, fmt.Errorf("%s: %w", fn, err)
		}
		orderItems = append(orderItems, orderItem)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", fn, err)
	}

	return orderItems, nil
}
