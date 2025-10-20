-- Event Pipeline Database Schema
-- MS SQL Server

-- Drop tables if they exist (for clean setup)
IF OBJECT_ID('order_items', 'U') IS NOT NULL DROP TABLE order_items;
IF OBJECT_ID('payments', 'U') IS NOT NULL DROP TABLE payments;
IF OBJECT_ID('orders', 'U') IS NOT NULL DROP TABLE orders;
IF OBJECT_ID('users', 'U') IS NOT NULL DROP TABLE users;
IF OBJECT_ID('inventory', 'U') IS NOT NULL DROP TABLE inventory;

-- Users table
CREATE TABLE users (
    user_id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    created_at DATETIME2 NOT NULL DEFAULT GETDATE(),
    updated_at DATETIME2 NOT NULL DEFAULT GETDATE()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_created_at ON users(created_at);

-- Orders table
CREATE TABLE orders (
    order_id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    total_amount DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    placed_at DATETIME2 NOT NULL,
    updated_at DATETIME2 NOT NULL DEFAULT GETDATE(),
    FOREIGN KEY (user_id) REFERENCES users(user_id)
);

CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_placed_at ON orders(placed_at DESC);

-- Order items table
CREATE TABLE order_items (
    id INT IDENTITY(1,1) PRIMARY KEY,
    order_id VARCHAR(36) NOT NULL,
    sku VARCHAR(50) NOT NULL,
    quantity INT NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    FOREIGN KEY (order_id) REFERENCES orders(order_id) ON DELETE CASCADE
);

CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_sku ON order_items(sku);

-- Payments table
CREATE TABLE payments (
    payment_id VARCHAR(36) PRIMARY KEY,
    order_id VARCHAR(36) NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    payment_method VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    settled_at DATETIME2 NOT NULL,
    updated_at DATETIME2 NOT NULL DEFAULT GETDATE(),
    FOREIGN KEY (order_id) REFERENCES orders(order_id)
);

CREATE INDEX idx_payments_order_id ON payments(order_id);
CREATE INDEX idx_payments_status ON payments(status);

-- Inventory table
CREATE TABLE inventory (
    sku VARCHAR(50) PRIMARY KEY,
    quantity INT NOT NULL DEFAULT 0,
    updated_at DATETIME2 NOT NULL DEFAULT GETDATE()
);

CREATE INDEX idx_inventory_updated_at ON inventory(updated_at);

-- Print success message
PRINT 'Database schema created successfully!';
