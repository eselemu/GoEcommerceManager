package main

import (
	"encoding/json"
	"fmt"
	"inventory-management/shared"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InventoryServer represents the server with its data stores
type InventoryServer struct {
	products     map[string]shared.Product
	orders       map[string]shared.Order
	mu           sync.RWMutex
	productsFile string
	ordersFile   string
}

// NewInventoryServer creates a new server instance
func NewInventoryServer(productsFile, ordersFile string) *InventoryServer {
	server := &InventoryServer{
		products:     make(map[string]shared.Product),
		orders:       make(map[string]shared.Order),
		productsFile: productsFile,
		ordersFile:   ordersFile,
	}

	// Load data from JSON files
	server.loadProducts()
	server.loadOrders()

	return server
}

// loadProducts loads products from JSON file
func (s *InventoryServer) loadProducts() {
	file, err := os.ReadFile(s.productsFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading products file: %v", err)
		}
		return
	}

	var products []shared.Product
	err = json.Unmarshal(file, &products)
	if err != nil {
		log.Printf("Error parsing products file: %v", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range products {
		s.products[p.ID] = p
	}
	fmt.Printf("Loaded %d products from file\n", len(products))
}

// saveProducts saves products to JSON file
func (s *InventoryServer) saveProducts() error {
	s.mu.RLock()
	products := make([]shared.Product, 0, len(s.products))
	for _, p := range s.products {
		products = append(products, p)
	}
	s.mu.RUnlock()

	data, err := json.MarshalIndent(products, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling products: %v", err)
	}

	err = os.WriteFile(s.productsFile, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing products file: %v", err)
	}

	return nil
}

// loadOrders loads orders from JSON file
func (s *InventoryServer) loadOrders() {
	file, err := os.ReadFile(s.ordersFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading orders file: %v", err)
		}
		return
	}

	var orders []shared.Order
	err = json.Unmarshal(file, &orders)
	if err != nil {
		log.Printf("Error parsing orders file: %v", err)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, o := range orders {
		s.orders[o.ID] = o
	}
	fmt.Printf("Loaded %d orders from file\n", len(orders))
}

// saveOrders saves orders to JSON file
func (s *InventoryServer) saveOrders() error {
	s.mu.RLock()
	orders := make([]shared.Order, 0, len(s.orders))
	for _, o := range s.orders {
		orders = append(orders, o)
	}
	s.mu.RUnlock()

	data, err := json.MarshalIndent(orders, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling orders: %v", err)
	}

	err = os.WriteFile(s.ordersFile, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing orders file: %v", err)
	}

	return nil
}

func main() {
	// Load configuration
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	fmt.Printf("Inventory Management Server is listening on %s\n", addr)

	// Create server instance with JSON file storage
	server := NewInventoryServer("products.json", "orders.json")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go server.handleConn(conn)
	}
}

func loadConfig(filename string) (*shared.Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config shared.Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (s *InventoryServer) handleConn(c net.Conn) {
	defer c.Close()
	fmt.Printf("Client connected from %s\n", c.RemoteAddr().String())

	decoder := json.NewDecoder(c)
	encoder := json.NewEncoder(c)

	for {
		// Read JSON request from client
		var req shared.Request
		err := decoder.Decode(&req)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Error decoding request from %s: %v\n", c.RemoteAddr().String(), err)
			}
			break
		}

		fmt.Printf("Received action '%s' from %s\n", req.Action, c.RemoteAddr().String())

		// Process the request and get response
		response := s.processRequest(req)

		// Send response back to client
		err = encoder.Encode(response)
		if err != nil {
			fmt.Printf("Error sending response to %s: %v\n", c.RemoteAddr().String(), err)
			break
		}
	}

	fmt.Printf("Client %s disconnected\n", c.RemoteAddr().String())
}

func (s *InventoryServer) processRequest(req shared.Request) shared.Response {
	switch req.Action {
	case shared.ActionAddProduct:
		return s.handleAddProduct(req.Parameters)
	case shared.ActionUpdateStock:
		return s.handleUpdateStock(req.Parameters)
	case shared.ActionUpdatePrice:
		return s.handleUpdatePrice(req.Parameters)
	case shared.ActionCreateOrder:
		return s.handleCreateOrder(req.Parameters)
	case shared.ActionUpdateOrderStatus:
		return s.handleUpdateOrderStatus(req.Parameters)
	case shared.ActionListProducts:
		return s.handleListProducts()
	case shared.ActionListOrders:
		return s.handleListOrders()
	case shared.ActionGetProduct:
		return s.handleGetProduct(req.Parameters)
	case shared.ActionGetOrder:
		return s.handleGetOrder(req.Parameters)
	case shared.ActionDeleteProduct:
		return s.handleDeleteProduct(req.Parameters)
	default:
		return shared.Response{
			Action:  req.Action,
			Success: false,
			Error:   fmt.Sprintf("Unknown action: %s", req.Action),
		}
	}
}

// handleAddProduct adds a new product to inventory
func (s *InventoryServer) handleAddProduct(params interface{}) shared.Response {
	fmt.Println("Processing add_product request...")

	// Parse parameters
	var p shared.AddProductParams
	data, err := json.Marshal(params)
	if err != nil {
		return shared.Response{
			Action:  shared.ActionAddProduct,
			Success: false,
			Error:   "Invalid parameters format",
		}
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		return shared.Response{
			Action:  shared.ActionAddProduct,
			Success: false,
			Error:   "Invalid parameters: " + err.Error(),
		}
	}

	// Validate input
	if p.Name == "" {
		return shared.Response{
			Action:  shared.ActionAddProduct,
			Success: false,
			Error:   "Product name cannot be empty",
		}
	}
	if p.Price <= 0 {
		return shared.Response{
			Action:  shared.ActionAddProduct,
			Success: false,
			Error:   "Price must be greater than 0",
		}
	}
	if p.Stock < 0 {
		return shared.Response{
			Action:  shared.ActionAddProduct,
			Success: false,
			Error:   "Stock cannot be negative",
		}
	}

	// Create new product
	now := time.Now()
	product := shared.Product{
		ID:        uuid.New().String(),
		Name:      p.Name,
		Price:     p.Price,
		Stock:     p.Stock,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Save to memory
	s.mu.Lock()
	s.products[product.ID] = product
	s.mu.Unlock()

	// Save to file
	err = s.saveProducts()
	if err != nil {
		return shared.Response{
			Action:  shared.ActionAddProduct,
			Success: false,
			Error:   "Error saving to file: " + err.Error(),
		}
	}

	return shared.Response{
		Action:  shared.ActionAddProduct,
		Success: true,
		Data:    product,
	}
}

// handleUpdateStock updates product stock
func (s *InventoryServer) handleUpdateStock(params interface{}) shared.Response {
	fmt.Println("Processing update_stock request...")

	// Parse parameters
	var p shared.UpdateStockParams
	data, err := json.Marshal(params)
	if err != nil {
		return shared.Response{
			Action:  shared.ActionUpdateStock,
			Success: false,
			Error:   "Invalid parameters format",
		}
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		return shared.Response{
			Action:  shared.ActionUpdateStock,
			Success: false,
			Error:   "Invalid parameters: " + err.Error(),
		}
	}

	// Validate input
	if p.NewStock < 0 {
		return shared.Response{
			Action:  shared.ActionUpdateStock,
			Success: false,
			Error:   "Stock cannot be negative",
		}
	}

	// Find product with safe unlock pattern
	s.mu.Lock()
	locked := true
	defer func() {
		if locked {
			s.mu.Unlock()
		}
	}()

	product, exists := s.products[p.ProductID]
	if !exists {
		locked = false
		s.mu.Unlock()
		return shared.Response{
			Action:  shared.ActionUpdateStock,
			Success: false,
			Error:   fmt.Sprintf("Product with ID %s not found", p.ProductID),
		}
	}

	// Update stock
	product.Stock = p.NewStock
	product.UpdatedAt = time.Now()
	s.products[p.ProductID] = product

	locked = false
	s.mu.Unlock()

	// Save to file
	err = s.saveProducts()
	if err != nil {
		return shared.Response{
			Action:  shared.ActionUpdateStock,
			Success: false,
			Error:   "Error saving to file: " + err.Error(),
		}
	}

	return shared.Response{
		Action:  shared.ActionUpdateStock,
		Success: true,
		Data:    product,
	}
}

// handleUpdatePrice updates product price
func (s *InventoryServer) handleUpdatePrice(params interface{}) shared.Response {
	fmt.Println("Processing update_price request...")

	// Parse parameters
	var p shared.UpdatePriceParams
	data, err := json.Marshal(params)
	if err != nil {
		return shared.Response{
			Action:  shared.ActionUpdatePrice,
			Success: false,
			Error:   "Invalid parameters format",
		}
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		return shared.Response{
			Action:  shared.ActionUpdatePrice,
			Success: false,
			Error:   "Invalid parameters: " + err.Error(),
		}
	}

	// Validate input
	if p.NewPrice <= 0 {
		return shared.Response{
			Action:  shared.ActionUpdatePrice,
			Success: false,
			Error:   "Price must be greater than 0",
		}
	}

	// Find product with safe unlock pattern
	s.mu.Lock()
	locked := true
	defer func() {
		if locked {
			s.mu.Unlock()
		}
	}()

	product, exists := s.products[p.ProductID]
	if !exists {
		locked = false
		s.mu.Unlock()
		return shared.Response{
			Action:  shared.ActionUpdatePrice,
			Success: false,
			Error:   fmt.Sprintf("Product with ID %s not found", p.ProductID),
		}
	}

	// Update price
	product.Price = p.NewPrice
	product.UpdatedAt = time.Now()
	s.products[p.ProductID] = product

	locked = false
	s.mu.Unlock()

	// Save to file
	err = s.saveProducts()
	if err != nil {
		return shared.Response{
			Action:  shared.ActionUpdatePrice,
			Success: false,
			Error:   "Error saving to file: " + err.Error(),
		}
	}

	return shared.Response{
		Action:  shared.ActionUpdatePrice,
		Success: true,
		Data:    product,
	}
}

// handleCreateOrder creates a new order
func (s *InventoryServer) handleCreateOrder(params interface{}) shared.Response {
	fmt.Println("Processing create_order request...")

	// Parse parameters
	var p shared.CreateOrderParams
	data, err := json.Marshal(params)
	if err != nil {
		return shared.Response{
			Action:  shared.ActionCreateOrder,
			Success: false,
			Error:   "Invalid parameters format",
		}
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		return shared.Response{
			Action:  shared.ActionCreateOrder,
			Success: false,
			Error:   "Invalid parameters: " + err.Error(),
		}
	}

	// Validate order items
	if len(p.Items) == 0 {
		return shared.Response{
			Action:  shared.ActionCreateOrder,
			Success: false,
			Error:   "Order must contain at least one item",
		}
	}

	// Lock for the entire order creation with safe unlock pattern
	s.mu.Lock()
	locked := true
	defer func() {
		if locked {
			s.mu.Unlock()
		}
	}()

	var orderItems []shared.OrderItem
	var total float64
	var productsToUpdate []string // Track products we modified for potential rollback

	// Check each product and calculate total
	for _, item := range p.Items {
		// Check if product exists
		product, exists := s.products[item.ProductID]
		if !exists {
			locked = false
			s.mu.Unlock()
			return shared.Response{
				Action:  shared.ActionCreateOrder,
				Success: false,
				Error:   fmt.Sprintf("Product with ID %s does not exist", item.ProductID),
			}
		}

		// Check if enough stock
		if product.Stock < item.Quantity {
			locked = false
			s.mu.Unlock()
			return shared.Response{
				Action:  shared.ActionCreateOrder,
				Success: false,
				Error: fmt.Sprintf("Insufficient stock for product %s. Available: %d, Requested: %d",
					product.Name, product.Stock, item.Quantity),
			}
		}

		// Calculate subtotal
		subtotal := float64(item.Quantity) * product.Price
		total += subtotal

		// Create order item
		orderItems = append(orderItems, shared.OrderItem{
			ProductID: item.ProductID,
			Name:      product.Name,
			Quantity:  item.Quantity,
			Price:     product.Price,
			Subtotal:  subtotal,
		})

		// Reduce stock
		product.Stock -= item.Quantity
		product.UpdatedAt = time.Now()
		s.products[item.ProductID] = product
		productsToUpdate = append(productsToUpdate, item.ProductID)
	}

	// Create order
	now := time.Now()
	order := shared.Order{
		ID:        uuid.New().String(),
		Products:  orderItems,
		Total:     total,
		Status:    shared.OrderStatusCreated,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Save order
	s.orders[order.ID] = order

	// Unlock before file operations
	locked = false
	s.mu.Unlock()

	// Save both products and orders to files
	err = s.saveProducts()
	if err != nil {
		return shared.Response{
			Action:  shared.ActionCreateOrder,
			Success: false,
			Error:   "Error saving products to file: " + err.Error(),
		}
	}

	err = s.saveOrders()
	if err != nil {
		return shared.Response{
			Action:  shared.ActionCreateOrder,
			Success: false,
			Error:   "Error saving order to file: " + err.Error(),
		}
	}

	return shared.Response{
		Action:  shared.ActionCreateOrder,
		Success: true,
		Data:    order,
	}
}

// handleUpdateOrderStatus updates order status
func (s *InventoryServer) handleUpdateOrderStatus(params interface{}) shared.Response {
	fmt.Println("Processing update_order_status request...")

	// Parse parameters
	var p shared.UpdateOrderStatusParams
	data, err := json.Marshal(params)
	if err != nil {
		return shared.Response{
			Action:  shared.ActionUpdateOrderStatus,
			Success: false,
			Error:   "Invalid parameters format",
		}
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		return shared.Response{
			Action:  shared.ActionUpdateOrderStatus,
			Success: false,
			Error:   "Invalid parameters: " + err.Error(),
		}
	}

	// Validate status
	validStatus := false
	switch p.Status {
	case shared.OrderStatusCreated, shared.OrderStatusCompleted, shared.OrderStatusCancelled:
		validStatus = true
	}
	if !validStatus {
		return shared.Response{
			Action:  shared.ActionUpdateOrderStatus,
			Success: false,
			Error:   fmt.Sprintf("Invalid status: %s", p.Status),
		}
	}

	// Lock with safe unlock pattern
	s.mu.Lock()
	locked := true
	defer func() {
		if locked {
			s.mu.Unlock()
		}
	}()

	// Find order
	order, exists := s.orders[p.OrderID]
	if !exists {
		locked = false
		s.mu.Unlock()
		return shared.Response{
			Action:  shared.ActionUpdateOrderStatus,
			Success: false,
			Error:   fmt.Sprintf("Order with ID %s not found", p.OrderID),
		}
	}

	// Check if order is already completed
	if order.Status == shared.OrderStatusCompleted && p.Status == shared.OrderStatusCompleted {
		locked = false
		s.mu.Unlock()
		return shared.Response{
			Action:  shared.ActionUpdateOrderStatus,
			Success: false,
			Error:   "Order cannot be completed twice",
		}
	}

	// Handle cancellation - restore stock
	stockRestored := false
	if p.Status == shared.OrderStatusCancelled && order.Status != shared.OrderStatusCancelled {
		for _, item := range order.Products {
			if product, exists := s.products[item.ProductID]; exists {
				product.Stock += item.Quantity
				product.UpdatedAt = time.Now()
				s.products[item.ProductID] = product
			}
		}
		stockRestored = true
	}

	// Update order status
	now := time.Now()
	order.Status = p.Status
	order.UpdatedAt = now

	if p.Status == shared.OrderStatusCompleted {
		order.CompletedAt = &now
	}

	s.orders[p.OrderID] = order

	// Unlock before file operations
	locked = false
	s.mu.Unlock()

	// Save products first if stock was restored
	if stockRestored {
		err = s.saveProducts()
		if err != nil {
			return shared.Response{
				Action:  shared.ActionUpdateOrderStatus,
				Success: false,
				Error:   "Error saving products to file: " + err.Error(),
			}
		}
	}

	// Save orders to file
	err = s.saveOrders()
	if err != nil {
		return shared.Response{
			Action:  shared.ActionUpdateOrderStatus,
			Success: false,
			Error:   "Error saving order to file: " + err.Error(),
		}
	}

	return shared.Response{
		Action:  shared.ActionUpdateOrderStatus,
		Success: true,
		Data:    order,
	}
}

// handleListProducts returns all products
func (s *InventoryServer) handleListProducts() shared.Response {
	fmt.Println("Processing list_products request...")

	s.mu.RLock()
	products := make([]shared.Product, 0, len(s.products))
	for _, p := range s.products {
		products = append(products, p)
	}
	s.mu.RUnlock()

	return shared.Response{
		Action:  shared.ActionListProducts,
		Success: true,
		Data:    products,
	}
}

// handleListOrders returns all orders
func (s *InventoryServer) handleListOrders() shared.Response {
	fmt.Println("Processing list_orders request...")

	s.mu.RLock()
	orders := make([]shared.Order, 0, len(s.orders))
	for _, o := range s.orders {
		orders = append(orders, o)
	}
	s.mu.RUnlock()

	return shared.Response{
		Action:  shared.ActionListOrders,
		Success: true,
		Data:    orders,
	}
}

// handleGetProduct returns a specific product
func (s *InventoryServer) handleGetProduct(params interface{}) shared.Response {
	fmt.Println("Processing get_product request...")

	// Parse parameters
	var p shared.GetProductParams
	data, err := json.Marshal(params)
	if err != nil {
		return shared.Response{
			Action:  shared.ActionGetProduct,
			Success: false,
			Error:   "Invalid parameters format",
		}
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		return shared.Response{
			Action:  shared.ActionGetProduct,
			Success: false,
			Error:   "Invalid parameters: " + err.Error(),
		}
	}

	s.mu.RLock()
	product, exists := s.products[p.ProductID]
	s.mu.RUnlock()

	if !exists {
		return shared.Response{
			Action:  shared.ActionGetProduct,
			Success: false,
			Error:   fmt.Sprintf("Product with ID %s not found", p.ProductID),
		}
	}

	return shared.Response{
		Action:  shared.ActionGetProduct,
		Success: true,
		Data:    product,
	}
}

// handleGetOrder returns a specific order
func (s *InventoryServer) handleGetOrder(params interface{}) shared.Response {
	fmt.Println("Processing get_order request...")

	// Parse parameters
	var p shared.GetOrderParams
	data, err := json.Marshal(params)
	if err != nil {
		return shared.Response{
			Action:  shared.ActionGetOrder,
			Success: false,
			Error:   "Invalid parameters format",
		}
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		return shared.Response{
			Action:  shared.ActionGetOrder,
			Success: false,
			Error:   "Invalid parameters: " + err.Error(),
		}
	}

	s.mu.RLock()
	order, exists := s.orders[p.OrderID]
	s.mu.RUnlock()

	if !exists {
		return shared.Response{
			Action:  shared.ActionGetOrder,
			Success: false,
			Error:   fmt.Sprintf("Order with ID %s not found", p.OrderID),
		}
	}

	return shared.Response{
		Action:  shared.ActionGetOrder,
		Success: true,
		Data:    order,
	}
}

// handleDeleteProduct deletes a product (only if not used in any order)
func (s *InventoryServer) handleDeleteProduct(params interface{}) shared.Response {
	fmt.Println("Processing delete_product request...")

	// Parse parameters
	var p shared.DeleteProductParams
	data, err := json.Marshal(params)
	if err != nil {
		return shared.Response{
			Action:  shared.ActionDeleteProduct,
			Success: false,
			Error:   "Invalid parameters format",
		}
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		return shared.Response{
			Action:  shared.ActionDeleteProduct,
			Success: false,
			Error:   "Invalid parameters: " + err.Error(),
		}
	}

	// Lock with safe unlock pattern
	s.mu.Lock()
	locked := true
	defer func() {
		if locked {
			s.mu.Unlock()
		}
	}()

	// Check if product exists
	_, exists := s.products[p.ProductID]
	if !exists {
		locked = false
		s.mu.Unlock()
		return shared.Response{
			Action:  shared.ActionDeleteProduct,
			Success: false,
			Error:   fmt.Sprintf("Product with ID %s not found", p.ProductID),
		}
	}

	// Check if product is used in any non-cancelled order
	for _, order := range s.orders {
		if order.Status != shared.OrderStatusCancelled {
			for _, item := range order.Products {
				if item.ProductID == p.ProductID {
					locked = false
					s.mu.Unlock()
					return shared.Response{
						Action:  shared.ActionDeleteProduct,
						Success: false,
						Error:   "Cannot delete product that is used in existing orders",
					}
				}
			}
		}
	}

	// Delete product
	delete(s.products, p.ProductID)

	// Unlock before file operation
	locked = false
	s.mu.Unlock()

	// Save to file
	err = s.saveProducts()
	if err != nil {
		return shared.Response{
			Action:  shared.ActionDeleteProduct,
			Success: false,
			Error:   "Error saving to file: " + err.Error(),
		}
	}

	return shared.Response{
		Action:  shared.ActionDeleteProduct,
		Success: true,
		Data:    map[string]string{"message": "Product deleted successfully"},
	}
}
