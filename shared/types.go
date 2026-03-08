package shared

import (
	"time"
)

// Config represents the application configuration
type Config struct {
	Server ServerConfig `json:"server"`
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// Request represents a JSON request from client to server
type Request struct {
	Action     string      `json:"action"`
	Parameters interface{} `json:"parameters"`
}

// Response represents a JSON response from server to client
type Response struct {
	Action  string      `json:"action"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Action constants
const (
	ActionAddProduct        = "add_product"
	ActionUpdateStock       = "update_stock"
	ActionUpdatePrice       = "update_price"
	ActionCreateOrder       = "create_order"
	ActionUpdateOrderStatus = "update_order_status"
	ActionListProducts      = "list_products"
	ActionListOrders        = "list_orders"
	ActionGetProduct        = "get_product"
	ActionGetOrder          = "get_order"
	ActionDeleteProduct     = "delete_product"
	ActionAddUser           = "add_user"
	ActionLogin             = "login"
)

// Order status constants
const (
	OrderStatusCreated   = "created"
	OrderStatusCompleted = "completed"
	OrderStatusCancelled = "cancelled"
)

// Product represents a product in the inventory
type Product struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Price     float64   `json:"price"`
	Stock     int       `json:"stock"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Order represents an order
type Order struct {
	ID          string      `json:"id"`
	Products    []OrderItem `json:"products"`
	Total       float64     `json:"total"`
	Status      string      `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
}

// User represents a user
type User struct {
	Username     string    `json:"username"`
	Password     []byte    `json:"password"`
	Admin        bool      `json:"admin"`
	RegisteredAt time.Time `json:"registered_at"`
}

// OrderItem represents a product in an order
type OrderItem struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
	Subtotal  float64 `json:"subtotal"`
}

// LoginParamas represents parameters for login
type LoginParams struct {
	Username     string    `json:"username"`
	Password     []byte    `json:"password"`
	RegisteredAt time.Time `json:"registered_at"`
}

// AddProductParams represents parameters for adding a product
type AddProductParams struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

// UpdateStockParams represents parameters for updating stock
type UpdateStockParams struct {
	ProductID string `json:"product_id"`
	NewStock  int    `json:"new_stock"`
}

// UpdatePriceParams represents parameters for updating price
type UpdatePriceParams struct {
	ProductID string  `json:"product_id"`
	NewPrice  float64 `json:"new_price"`
}

// CreateOrderParams represents parameters for creating an order
type CreateOrderParams struct {
	Items []OrderItemRequest `json:"items"`
}

// OrderItemRequest represents a product request in an order
type OrderItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// UpdateOrderStatusParams represents parameters for updating order status
type UpdateOrderStatusParams struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}

// GetProductParams represents parameters for getting a product
type GetProductParams struct {
	ProductID string `json:"product_id"`
}

// GetOrderParams represents parameters for getting an order
type GetOrderParams struct {
	OrderID string `json:"order_id"`
}

// DeleteProductParams represents parameters for deleting a product
type DeleteProductParams struct {
	ProductID string `json:"product_id"`
}

type Session struct {
	Username string `json:"username"`
	Admin    bool   `json:"admin"`
}
