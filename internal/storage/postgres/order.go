package postgres

import (
	"context"
	"errors"
	"fmt"
	"go-pet-shop/internal/models"

	"github.com/jackc/pgx/v5"
)

func (s *Storage) CreateOrder(ctx context.Context, o models.Order) (models.Order, error) {
	const fn = "storage.postgres.order.CreateOrder"

	err := s.db.QueryRow(ctx,
		`INSERT INTO orders (user_id, total_price) VALUES ($1, $2) RETURNING id, created_at`,
		o.UserID, o.TotalPrice).
		Scan(&o.ID, &o.CreatedAt)

	if err != nil {
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

func (s *Storage) PlaceOrder(ctx context.Context, userEmail string, items []models.OrderItem) (int, error) {
	const fn = "storage.postgres.order.PlaceOrder"

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", fn, err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var userID int
	err = tx.QueryRow(ctx, `SELECT id FROM users WHERE email = $1`, userEmail).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("%s: %w", fn, ErrNotFound)
		}
		return 0, fmt.Errorf("%s: %w", fn, err)
	}

	var totalPrice float64
	for _, item := range items {
		var price float64
		err := tx.QueryRow(ctx,
			`UPDATE products 
			 SET stock = stock - $1 
			 WHERE id = $2 AND stock >= $1 
			 RETURNING price`,
			item.Quantity, item.ProductID,
		).Scan(&price)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return 0, fmt.Errorf("%s: product %d: %w", fn, item.ProductID, ErrOutOfStock)
			}
			return 0, fmt.Errorf("%s: %w", fn, err)
		}

		totalPrice += price * float64(item.Quantity)
	}

	var orderID int
	err = tx.QueryRow(ctx,
		`INSERT INTO orders (user_id, total_price) VALUES ($1, $2) RETURNING id`,
		userID, totalPrice,
	).Scan(&orderID)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", fn, err)
	}

	for _, item := range items {
		_, err := tx.Exec(ctx,
			`INSERT INTO order_items (order_id, product_id, quantity) VALUES ($1, $2, $3)`,
			orderID, item.ProductID, item.Quantity,
		)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", fn, err)
		}
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO transactions (order_id, amount, status) VALUES ($1, $2, $3)`,
		orderID, totalPrice, models.TransactionStatusPending,
	)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", fn, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("%s: %w", fn, err)
	}

	return orderID, nil
}
