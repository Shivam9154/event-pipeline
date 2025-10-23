package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"event-pipeline/internal/config"
	"event-pipeline/internal/logger"
	"event-pipeline/internal/metrics"
	"event-pipeline/internal/models"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/sirupsen/logrus"
)

// DB wraps the SQL database connection
type DB struct {
	conn *sql.DB
}

// New creates a new database connection
func New(cfg *config.MSSQLConfig) (*DB, error) {
	connString := cfg.GetConnectionString()

	conn, err := sql.Open("sqlserver", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	logger.Log.Info("Successfully connected to MS SQL database")

	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// UpsertUser inserts or updates a user (idempotent)
func (db *DB) UpsertUser(ctx context.Context, event models.UserCreated) error {
	start := time.Now()
	defer func() {
		metrics.DBLatency.WithLabelValues("upsert_user").Observe(time.Since(start).Seconds())
	}()

	query := `
		MERGE INTO users AS target
		USING (SELECT @p1 AS user_id) AS source
		ON target.user_id = source.user_id
		WHEN MATCHED THEN
			UPDATE SET email = @p2, first_name = @p3, last_name = @p4, updated_at = @p5
		WHEN NOT MATCHED THEN
			INSERT (user_id, email, first_name, last_name, created_at, updated_at)
			VALUES (@p1, @p2, @p3, @p4, @p6, @p5);
	`

	_, err := db.conn.ExecContext(ctx, query,
		event.UserID,
		event.Email,
		event.FirstName,
		event.LastName,
		time.Now(),
		event.CreatedAt,
	)

	if err != nil {
		logger.WithEventID(event.EventID).WithFields(logrus.Fields{
			"userId": event.UserID,
			"error":  err.Error(),
		}).Error("Failed to upsert user")
		return fmt.Errorf("failed to upsert user: %w", err)
	}

	logger.WithEventID(event.EventID).WithFields(logrus.Fields{
		"userId": event.UserID,
	}).Info("User upserted successfully")

	return nil
}

// UpsertOrder inserts or updates an order (idempotent)
func (db *DB) UpsertOrder(ctx context.Context, event models.OrderPlaced) error {
	start := time.Now()
	defer func() {
		metrics.DBLatency.WithLabelValues("upsert_order").Observe(time.Since(start).Seconds())
	}()

	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Upsert order
	orderQuery := `
		MERGE INTO orders AS target
		USING (SELECT @p1 AS order_id) AS source
		ON target.order_id = source.order_id
		WHEN MATCHED THEN
			UPDATE SET user_id = @p2, total_amount = @p3, currency = @p4, updated_at = @p5
		WHEN NOT MATCHED THEN
			INSERT (order_id, user_id, total_amount, currency, placed_at, updated_at)
			VALUES (@p1, @p2, @p3, @p4, @p6, @p5);
	`

	_, err = tx.ExecContext(ctx, orderQuery,
		event.OrderID,
		event.UserID,
		event.TotalAmount,
		event.Currency,
		time.Now(),
		event.PlacedAt,
	)

	if err != nil {
		logger.WithEventID(event.EventID).Error("Failed to upsert order")
		return fmt.Errorf("failed to upsert order: %w", err)
	}

	// Delete existing order items
	deleteQuery := `DELETE FROM order_items WHERE order_id = @p1`
	_, err = tx.ExecContext(ctx, deleteQuery, event.OrderID)
	if err != nil {
		return fmt.Errorf("failed to delete existing order items: %w", err)
	}

	// Insert order items
	itemQuery := `
		INSERT INTO order_items (order_id, sku, quantity, price)
		VALUES (@p1, @p2, @p3, @p4)
	`

	for _, item := range event.Items {
		_, err = tx.ExecContext(ctx, itemQuery,
			event.OrderID,
			item.SKU,
			item.Quantity,
			item.Price,
		)
		if err != nil {
			return fmt.Errorf("failed to insert order item: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.WithEventID(event.EventID).WithFields(logrus.Fields{
		"orderId": event.OrderID,
	}).Info("Order upserted successfully")

	return nil
}

// UpsertPayment inserts or updates a payment (idempotent)
func (db *DB) UpsertPayment(ctx context.Context, event models.PaymentSettled) error {
	start := time.Now()
	defer func() {
		metrics.DBLatency.WithLabelValues("upsert_payment").Observe(time.Since(start).Seconds())
	}()

	query := `
		MERGE INTO payments AS target
		USING (SELECT @p1 AS payment_id) AS source
		ON target.payment_id = source.payment_id
		WHEN MATCHED THEN
			UPDATE SET order_id = @p2, amount = @p3, currency = @p4, 
			           payment_method = @p5, status = @p6, settled_at = @p7, updated_at = @p8
		WHEN NOT MATCHED THEN
			INSERT (payment_id, order_id, amount, currency, payment_method, status, settled_at, updated_at)
			VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8);
	`

	_, err := db.conn.ExecContext(ctx, query,
		event.PaymentID,
		event.OrderID,
		event.Amount,
		event.Currency,
		event.PaymentMethod,
		event.Status,
		event.SettledAt,
		time.Now(),
	)

	if err != nil {
		logger.WithEventID(event.EventID).Error("Failed to upsert payment")
		return fmt.Errorf("failed to upsert payment: %w", err)
	}

	logger.WithEventID(event.EventID).WithFields(logrus.Fields{
		"paymentId": event.PaymentID,
	}).Info("Payment upserted successfully")

	return nil
}

// UpsertInventory adjusts inventory (idempotent via unique constraint)
func (db *DB) UpsertInventory(ctx context.Context, event models.InventoryAdjusted) error {
	start := time.Now()
	defer func() {
		metrics.DBLatency.WithLabelValues("upsert_inventory").Observe(time.Since(start).Seconds())
	}()

	// Calculate quantity delta
	delta := event.Quantity
	if event.AdjustmentType == "subtract" {
		delta = -delta
	}

	query := `
		MERGE INTO inventory AS target
		USING (SELECT @p1 AS sku) AS source
		ON target.sku = source.sku
		WHEN MATCHED THEN
			UPDATE SET quantity = target.quantity + @p2, updated_at = @p3
		WHEN NOT MATCHED THEN
			INSERT (sku, quantity, updated_at)
			VALUES (@p1, @p2, @p3);
	`

	_, err := db.conn.ExecContext(ctx, query,
		event.SKU,
		delta,
		time.Now(),
	)

	if err != nil {
		logger.WithEventID(event.EventID).Error("Failed to upsert inventory")
		return fmt.Errorf("failed to upsert inventory: %w", err)
	}

	logger.WithEventID(event.EventID).WithFields(logrus.Fields{
		"sku":   event.SKU,
		"delta": delta,
	}).Info("Inventory adjusted successfully")

	return nil
}

// GetUserWithOrders retrieves a user with their last 5 orders
func (db *DB) GetUserWithOrders(ctx context.Context, userID string) (*UserWithOrders, error) {
	start := time.Now()
	defer func() {
		metrics.DBLatency.WithLabelValues("get_user_orders").Observe(time.Since(start).Seconds())
	}()

	// Get user
	userQuery := `
		SELECT user_id, email, first_name, last_name, created_at, updated_at
		FROM users
		WHERE user_id = @p1
	`

	var user UserWithOrders
	err := db.conn.QueryRowContext(ctx, userQuery, userID).Scan(
		&user.UserID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get last 5 orders
	ordersQuery := `
		SELECT TOP 5 order_id, user_id, total_amount, currency, placed_at, updated_at
		FROM orders
		WHERE user_id = @p1
		ORDER BY placed_at DESC
	`

	rows, err := db.conn.QueryContext(ctx, ordersQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}
	defer rows.Close()

	user.Orders = []Order{}
	for rows.Next() {
		var order Order
		err := rows.Scan(
			&order.OrderID,
			&order.UserID,
			&order.TotalAmount,
			&order.Currency,
			&order.PlacedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		user.Orders = append(user.Orders, order)
	}

	return &user, nil
}

// GetOrderWithPayment retrieves an order with payment status
func (db *DB) GetOrderWithPayment(ctx context.Context, orderID string) (*OrderWithPayment, error) {
	start := time.Now()
	defer func() {
		metrics.DBLatency.WithLabelValues("get_order_payment").Observe(time.Since(start).Seconds())
	}()

	query := `
		SELECT 
			o.order_id, o.user_id, o.total_amount, o.currency, o.placed_at, o.updated_at,
			p.payment_id, p.amount, p.payment_method, p.status, p.settled_at
		FROM orders o
		LEFT JOIN payments p ON o.order_id = p.order_id
		WHERE o.order_id = @p1
	`

	var order OrderWithPayment
	var paymentID, paymentMethod, paymentStatus sql.NullString
	var paymentAmount sql.NullFloat64
	var settledAt sql.NullTime

	err := db.conn.QueryRowContext(ctx, query, orderID).Scan(
		&order.OrderID,
		&order.UserID,
		&order.TotalAmount,
		&order.Currency,
		&order.PlacedAt,
		&order.UpdatedAt,
		&paymentID,
		&paymentAmount,
		&paymentMethod,
		&paymentStatus,
		&settledAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("order not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Set payment details if exists
	if paymentID.Valid {
		order.Payment = &Payment{
			PaymentID:     paymentID.String,
			Amount:        paymentAmount.Float64,
			PaymentMethod: paymentMethod.String,
			Status:        paymentStatus.String,
			SettledAt:     settledAt.Time,
		}
	}

	return &order, nil
}

// Response models
type UserWithOrders struct {
	UserID    string    `json:"userId"`
	Email     string    `json:"email"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Orders    []Order   `json:"orders"`
}

// UserSummary is a lightweight view for listing users
type UserSummary struct {
	UserID    string    `json:"userId"`
	Email     string    `json:"email"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// GetRecentUsers returns the latest N users by created_at desc
func (db *DB) GetRecentUsers(ctx context.Context, limit int) ([]UserSummary, error) {
	start := time.Now()
	defer func() {
		metrics.DBLatency.WithLabelValues("get_recent_users").Observe(time.Since(start).Seconds())
	}()

	if limit <= 0 || limit > 100 {
		limit = 5
	}

	query := `
		SELECT TOP (@p1) user_id, email, first_name, last_name, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := db.conn.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []UserSummary
	for rows.Next() {
		var u UserSummary
		if err := rows.Scan(&u.UserID, &u.Email, &u.FirstName, &u.LastName, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}

	return users, nil
}

type Order struct {
	OrderID     string    `json:"orderId"`
	UserID      string    `json:"userId"`
	TotalAmount float64   `json:"totalAmount"`
	Currency    string    `json:"currency"`
	PlacedAt    time.Time `json:"placedAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type OrderWithPayment struct {
	OrderID     string    `json:"orderId"`
	UserID      string    `json:"userId"`
	TotalAmount float64   `json:"totalAmount"`
	Currency    string    `json:"currency"`
	PlacedAt    time.Time `json:"placedAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Payment     *Payment  `json:"payment,omitempty"`
}

type Payment struct {
	PaymentID     string    `json:"paymentId"`
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"paymentMethod"`
	Status        string    `json:"status"`
	SettledAt     time.Time `json:"settledAt"`
}
