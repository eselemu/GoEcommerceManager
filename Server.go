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
	"golang.org/x/crypto/bcrypt"
)

var (
	serverLogger     *log.Logger
	handleConnLogger *log.Logger
)

// InventoryServer represents the server with its data stores
type InventoryServer struct {
	products     map[string]shared.Product
	orders       map[string]shared.Order
	users        map[string]shared.User
	mu           sync.RWMutex
	productsFile string
	ordersFile   string
	usersFile    string
}

// NewInventoryServer creates a new server instance
func NewInventoryServer(productsFile, ordersFile, usersFile string) *InventoryServer {
	server := &InventoryServer{
		products:     make(map[string]shared.Product),
		orders:       make(map[string]shared.Order),
		users:        make(map[string]shared.User),
		productsFile: productsFile,
		ordersFile:   ordersFile,
		usersFile:    usersFile,
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

func (s *InventoryServer) promptAdmin() bool {
	file, err := os.ReadFile(s.usersFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading products file: %v", err)
			fmt.Printf("No user file exists.")
		}
		return false
	}

	var users []shared.User
	err = json.Unmarshal(file, &users)
	if err != nil {
		log.Printf("Error parsing clients file %v", err)
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range users {
		s.users[u.Username] = u
	}

	for _, u := range users {
		if u.Admin {
			fmt.Printf("Loaded %d users from file\n", len(users))
			return true
		}
	}
	return false
}

func (s *InventoryServer) bootstrapAdmin() shared.Response {
	now := time.Now()
	hash, err := bcrypt.GenerateFromPassword([]byte("12345"), bcrypt.DefaultCost)

	if err == nil {
		admin := shared.User{
			Username:     "Admin1",
			Password:     hash,
			Admin:        true,
			RegisteredAt: now,
		}

		// Save to memory
		s.mu.Lock()
		s.users[admin.Username] = admin
		s.mu.Unlock()

		err = s.saveUsers()
		if err != nil {
			log.Print("Error saving to file in bootstrapAdmin, returning")
			return shared.Response{
				Action:  shared.ActionAddUser,
				Success: false,
				Error:   "Error adding user:" + err.Error(),
			}
		}
		log.Print("Admin added succesfully, returning to main.")

		return shared.Response{
			Action:  shared.ActionAddProduct,
			Success: true,
			Data:    admin,
		}

	} else {
		log.Print("Error creating a new admin")
		return shared.Response{Action: shared.ActionAddUser, Success: false, Error: err.Error()}
	}

}

func (s *InventoryServer) saveUsers() error {
	s.mu.RLock()
	users := make([]shared.User, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u)
	}
	s.mu.RUnlock()

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling users: %v", err)
	}

	err = os.WriteFile(s.usersFile, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing users file: %v", err)
	}

	return nil
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
	file, err := os.OpenFile("info.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Failed to open log file")
	}
	defer file.Close()

	serverLogger = log.New(file, "[Server] main(): ", log.LstdFlags)
	handleConnLogger = log.New(file, "[Server] handleConn():", log.LstdFlags)
	// Load configuration
	config, err := loadConfig("config.json")
	if err != nil {
		serverLogger.Fatal("Error loading config:", err)
	}
	serverLogger.Print("Loaded config file")

	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		serverLogger.Fatal(err)
	}
	serverLogger.Print(fmt.Sprintf("Listening on %s", addr))
	defer listener.Close()

	fmt.Printf("Inventory Management Server is listening on %s\n", addr)

	// Create server instance with JSON file storage
	server := NewInventoryServer("products.json", "orders.json", "users.json")

	if !server.promptAdmin() {
		server.bootstrapAdmin()
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			serverLogger.Printf("Error accepting connection: %v", err)
			continue
		}
		serverLogger.Print("Connection succesful. Creating goroutine for new client")

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
	handleConnLogger := log.New(handleConnLogger.Writer(), fmt.Sprintf("[Server] handleConn() for client %s: ", c.RemoteAddr().String()), log.LstdFlags)
	fmt.Printf("Client connected from %s\n", c.RemoteAddr().String())
	handleConnLogger.Printf("Client connected, handling connection")

	decoder := json.NewDecoder(c)
	encoder := json.NewEncoder(c)

	var session *shared.Session
	for session == nil {
		var req shared.Request
		err := decoder.Decode(&req)
		if err != nil {
			return
		}

		if req.Action != shared.ActionLogin {
			encoder.Encode(shared.Response{Action: req.Action, Success: false, Error: "You must login first"})
			continue
		}

		resp := s.handleLogin(req.Parameters, handleConnLogger)
		if !resp.Success {
			encoder.Encode(resp)
			continue
		}

		sessionData := resp.Data.(shared.Session)
		session = &sessionData
		encoder.Encode(shared.Response{Action: req.Action, Success: true, Data: session})
	}

	for {
		// Read JSON request from client
		var req shared.Request
		err := decoder.Decode(&req)
		if err != nil {
			if err != io.EOF {
				handleConnLogger.Print("Error decoding request from %s: %v", c.RemoteAddr().String(), err)
				fmt.Printf("Error decoding request from %s: %v\n", c.RemoteAddr().String(), err)
			}
			break
		}
		handleConnLogger.Print("Action received valid, processing")
		fmt.Printf("Received action '%s' from %s\n", req.Action, c.RemoteAddr().String())

		// Process the request and get response
		response := s.processRequest(req, session, handleConnLogger)

		// Send response back to client
		err = encoder.Encode(response)
		if err != nil {
			handleConnLogger.Print("Failure to send response to %s: %v", c.RemoteAddr().String(), err)
			fmt.Printf("Error sending response to %s: %v\n", c.RemoteAddr().String(), err)
			break
		}
	}

	handleConnLogger.Print("Connection with client %s terminated", c.RemoteAddr().String())
	fmt.Printf("Client %s disconnected\n", c.RemoteAddr().String())
}

func (s *InventoryServer) processRequest(req shared.Request, session *shared.Session, logger *log.Logger) shared.Response {
	handleConnLogger.SetPrefix("[Server] processRequest(): ")

	switch req.Action {
	case shared.ActionAddProduct,
		shared.ActionUpdateStock,
		shared.ActionUpdatePrice,
		shared.ActionUpdateOrderStatus,
		shared.ActionDeleteProduct:
		if !session.Admin {
			logger.Print("Unauthorized action attempted. Not an admin user")
			return shared.Response{
				Action:  req.Action,
				Success: false,
				Error:   fmt.Sprintf("Unauthorized action. Only admins can %s", req.Action),
			}
		}
	}

	switch req.Action {
	case shared.ActionAddProduct:
		return s.handleAddProduct(req.Parameters, logger)
	case shared.ActionUpdateStock:
		return s.handleUpdateStock(req.Parameters, logger)
	case shared.ActionUpdatePrice:
		return s.handleUpdatePrice(req.Parameters, logger)
	case shared.ActionCreateOrder:
		return s.handleCreateOrder(req.Parameters, logger)
	case shared.ActionUpdateOrderStatus:
		return s.handleUpdateOrderStatus(req.Parameters, logger)
	case shared.ActionListProducts:
		return s.handleListProducts(logger)
	case shared.ActionListOrders:
		return s.handleListOrders(logger)
	case shared.ActionGetProduct:
		return s.handleGetProduct(req.Parameters, logger)
	case shared.ActionGetOrder:
		return s.handleGetOrder(req.Parameters, logger)
	case shared.ActionDeleteProduct:
		return s.handleDeleteProduct(req.Parameters, logger)
	default:
		serverLogger.Print("Action unknown, returning to handle conn")
		return shared.Response{
			Action:  req.Action,
			Success: false,
			Error:   fmt.Sprintf("Unknown action: %s", req.Action),
		}
	}
}

// Handle login checks if login is valid
func (s *InventoryServer) handleLogin(params interface{}, logger *log.Logger) shared.Response {
	fmt.Println("Processing login request...")
	logger.Print("Entering handleLogin")

	var l shared.LoginParams
	data, err := json.Marshal(params)
	if err != nil {
		logger.Print("Invalid parameters in handleLogin, returning")
		return shared.Response{
			Action:  shared.ActionLogin,
			Success: false,
			Error:   "Invalid parameters format",
		}
	}
	err = json.Unmarshal(data, &l)
	if err != nil {
		logger.Print("Error unmarshalling in handleLogin, returning")
		return shared.Response{
			Action:  shared.ActionLogin,
			Success: false,
			Error:   "Invalid parameters: " + err.Error(),
		}
	}

	// Validate input
	if l.Username == "" {
		logger.Print("Invalid parameters in handleLogin; username empty, returning")
		return shared.Response{
			Action:  shared.ActionLogin,
			Success: false,
			Error:   "Username cannot be empty",
		}
	}

	if len(l.Password) == 0 {
		logger.Print("Invalid parameters in handleLogin; password empty, returning")
		return shared.Response{
			Action:  shared.ActionAddProduct,
			Success: false,
			Error:   "Password cannot be empty",
		}
	}

	// Check users
	s.mu.Lock()
	user, exists := s.users[l.Username]
	s.mu.Unlock()

	if !exists {
		logger.Print("Login failed, user not found")
		return shared.Response{
			Action:  shared.ActionLogin,
			Success: false,
			Error:   "Invalid username or password",
		}
	}

	//Checks if password matches the password registered for the user
	err = bcrypt.CompareHashAndPassword(user.Password, []byte(l.Password))
	if err != nil {
		logger.Print("Login failed, password incorrect")
		return shared.Response{
			Action:  shared.ActionLogin,
			Success: false,
			Error:   "Invalid password",
		}
	}

	logger.Print("Success. Creating session")
	return shared.Response{
		Action:  shared.ActionLogin,
		Success: true,
		Data: shared.Session{
			Username: user.Username,
			Admin:    user.Admin,
		},
	}
}

// handleAddProduct adds a new product to inventory
func (s *InventoryServer) handleAddProduct(params interface{}, logger *log.Logger) shared.Response {
	fmt.Println("Processing add_product request...")
	logger.Print("Entering handleAddProduct")

	// Parse parameters
	var p shared.AddProductParams
	data, err := json.Marshal(params)
	if err != nil {
		logger.Print("Invalid parameters in handleAddProduct, returning")
		return shared.Response{
			Action:  shared.ActionAddProduct,
			Success: false,
			Error:   "Invalid parameters format",
		}
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		logger.Print("Error unmarshaling in handleAddProduct, returning")
		return shared.Response{
			Action:  shared.ActionAddProduct,
			Success: false,
			Error:   "Invalid parameters: " + err.Error(),
		}
	}

	// Validate input
	if p.Name == "" {
		logger.Print("Invalid parameters in handleAddProduct; name empty, returning")
		return shared.Response{
			Action:  shared.ActionAddProduct,
			Success: false,
			Error:   "Product name cannot be empty",
		}
	}
	if p.Price <= 0 {
		logger.Print("Invalid parameters in handleAddProduct; price less than 0, returning")
		return shared.Response{
			Action:  shared.ActionAddProduct,
			Success: false,
			Error:   "Price must be greater than 0",
		}
	}
	if p.Stock < 0 {
		logger.Print("Invalid parameters in handleAddProduct; stock less than 0, returning")
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
		logger.Print("Error saving to file in handleAddProduct, returning")
		return shared.Response{
			Action:  shared.ActionAddProduct,
			Success: false,
			Error:   "Error saving to file: " + err.Error(),
		}
	}

	logger.Print("Product added succesfully, returning to main")

	return shared.Response{
		Action:  shared.ActionAddProduct,
		Success: true,
		Data:    product,
	}
}

// handleUpdateStock updates product stock
func (s *InventoryServer) handleUpdateStock(params interface{}, logger *log.Logger) shared.Response {
	fmt.Println("Processing update_stock request...")
	logger.Print("Entering handleUpdateStock")
	// Parse parameters
	var p shared.UpdateStockParams
	data, err := json.Marshal(params)
	if err != nil {
		logger.Print("Invalid parameters in handleUpdateStock, returning")
		return shared.Response{
			Action:  shared.ActionUpdateStock,
			Success: false,
			Error:   "Invalid parameters format",
		}
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		logger.Print("Error unmarshaling in handleUpdateStock, returning")
		return shared.Response{
			Action:  shared.ActionUpdateStock,
			Success: false,
			Error:   "Invalid parameters: " + err.Error(),
		}
	}

	// Validate input
	if p.NewStock < 0 {
		logger.Print("Invalid parameters in handleUpdateStock; stock less than 0, returning")
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
		logger.Print("Invalid parameters in handleUpdateStock; invalid product id, returning")
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
		logger.Print("Error saving to file  in handleUpdateStock, returning")
		return shared.Response{
			Action:  shared.ActionUpdateStock,
			Success: false,
			Error:   "Error saving to file: " + err.Error(),
		}
	}

	logger.Print("Updated stock succesfully, returning to main")
	return shared.Response{
		Action:  shared.ActionUpdateStock,
		Success: true,
		Data:    product,
	}
}

// handleUpdatePrice updates product price
func (s *InventoryServer) handleUpdatePrice(params interface{}, logger *log.Logger) shared.Response {
	fmt.Println("Processing update_price request...")
	logger.Print("Entered handleUpdatePrice")

	// Parse parameters
	var p shared.UpdatePriceParams
	data, err := json.Marshal(params)
	if err != nil {
		logger.Print("Invalid parameters in handleUpdatePrice, returning")
		return shared.Response{
			Action:  shared.ActionUpdatePrice,
			Success: false,
			Error:   "Invalid parameters format",
		}
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		logger.Print("Error unmarshaling in handleUpdatePrice returning")
		return shared.Response{
			Action:  shared.ActionUpdatePrice,
			Success: false,
			Error:   "Invalid parameters: " + err.Error(),
		}
	}

	// Validate input
	if p.NewPrice <= 0 {
		logger.Print("Invalid parameters in handleUpdatePrice; price less than 0, returning")
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
		logger.Print("Invalid parameters in handleUpdatePrice; invalid product id, returning")
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
		logger.Print("Error saving to file in handleUpdatePrice, returning")
		return shared.Response{
			Action:  shared.ActionUpdatePrice,
			Success: false,
			Error:   "Error saving to file: " + err.Error(),
		}
	}
	logger.Print("Price updated succesfully, returning to main")
	return shared.Response{
		Action:  shared.ActionUpdatePrice,
		Success: true,
		Data:    product,
	}
}

// handleCreateOrder creates a new order
func (s *InventoryServer) handleCreateOrder(params interface{}, logger *log.Logger) shared.Response {
	fmt.Println("Processing create_order request...")
	logger.Print("Entered handleCreateOrder")

	// Parse parameters
	var p shared.CreateOrderParams
	data, err := json.Marshal(params)
	if err != nil {
		logger.Print("Invalid parameters in handleCreateOrder, returning")
		return shared.Response{
			Action:  shared.ActionCreateOrder,
			Success: false,
			Error:   "Invalid parameters format",
		}
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		logger.Print("Error unmarshaling in handleCreateOrder, returning")
		return shared.Response{
			Action:  shared.ActionCreateOrder,
			Success: false,
			Error:   "Invalid parameters: " + err.Error(),
		}
	}

	// Validate order items
	if len(p.Items) == 0 {
		logger.Print("Invalid parameters in handleCreateOrder; item less than zero, returning")
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
			logger.Print("Invalid parameters in handleCreateOrder; unidentified product id, returning")
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
			logger.Print("Invalid parameters in handleCreateOrder; stock not sufficient for request, returning")
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
		logger.Print("Succesfully created order item")
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
		logger.Print("Error saving products to file after creating order, returning")
		return shared.Response{
			Action:  shared.ActionCreateOrder,
			Success: false,
			Error:   "Error saving products to file: " + err.Error(),
		}
	}

	err = s.saveOrders()
	if err != nil {
		logger.Print("Error saving order to file, returning")
		return shared.Response{
			Action:  shared.ActionCreateOrder,
			Success: false,
			Error:   "Error saving order to file: " + err.Error(),
		}
	}

	logger.Print("Order succesfully created and saved, returning")
	return shared.Response{
		Action:  shared.ActionCreateOrder,
		Success: true,
		Data:    order,
	}
}

// handleUpdateOrderStatus updates order status
func (s *InventoryServer) handleUpdateOrderStatus(params interface{}, logger *log.Logger) shared.Response {
	fmt.Println("Processing update_order_status request...")
	logger.Print("Entered handleUpdateOrderStatus")

	// Parse parameters
	var p shared.UpdateOrderStatusParams
	data, err := json.Marshal(params)
	if err != nil {
		logger.Print("Error marshaling in handleUpdateOrderStatus, returning")
		return shared.Response{
			Action:  shared.ActionUpdateOrderStatus,
			Success: false,
			Error:   "Invalid parameters format",
		}
	}

	err = json.Unmarshal(data, &p)
	logger.Print("Error unmarshaling in handleUpdateOrderStatus, returning")
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
		logger.Print("Invalid status in handleCreateOrder, returning")
		return shared.Response{
			Action:  shared.ActionUpdateOrderStatus,
			Success: false,
			Error:   fmt.Sprintf("Invalid status in handleCreateOrder: %s", p.Status),
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
		logger.Print("Invalid parameters in handleCreateOrder; order id not found, returning")
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
		logger.Print("Invalid parameters in handleCreateOrder; Order status marked as completed, returning")
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

	logger.Print("Restored stock at handleUpdateOrderStatus")

	// Update order status
	now := time.Now()
	order.Status = p.Status
	order.UpdatedAt = now

	if p.Status == shared.OrderStatusCompleted {
		order.CompletedAt = &now
	}

	s.orders[p.OrderID] = order

	logger.Print("Order status changed in handleCreateOrder")
	// Unlock before file operations
	locked = false
	s.mu.Unlock()

	// Save products first if stock was restored
	if stockRestored {
		err = s.saveProducts()
		if err != nil {
			logger.Print("Error saving products to file in handleCreateOrder, returning")
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
		logger.Print("Error saving order in handleCreateOrder, returning")
		return shared.Response{
			Action:  shared.ActionUpdateOrderStatus,
			Success: false,
			Error:   "Error saving order to file: " + err.Error(),
		}
	}

	logger.Print("Order status changed and saved succesfully in handleCreateOrder, returning")

	return shared.Response{
		Action:  shared.ActionUpdateOrderStatus,
		Success: true,
		Data:    order,
	}
}

// handleListProducts returns all products
func (s *InventoryServer) handleListProducts(logger *log.Logger) shared.Response {
	fmt.Println("Processing list_products request...")
	logger.Print("Entered handleListProducts")

	s.mu.RLock()
	products := make([]shared.Product, 0, len(s.products))
	for _, p := range s.products {
		products = append(products, p)
	}
	s.mu.RUnlock()

	logger.Print("Sharing list of products, returning")

	return shared.Response{
		Action:  shared.ActionListProducts,
		Success: true,
		Data:    products,
	}
}

// handleListOrders returns all orders
func (s *InventoryServer) handleListOrders(logger *log.Logger) shared.Response {
	fmt.Println("Processing list_orders request...")
	logger.Print("Entered handleListOrders")

	s.mu.RLock()
	orders := make([]shared.Order, 0, len(s.orders))
	for _, o := range s.orders {
		orders = append(orders, o)
	}
	s.mu.RUnlock()

	logger.Print("Sharing list of orders, returning")
	return shared.Response{
		Action:  shared.ActionListOrders,
		Success: true,
		Data:    orders,
	}
}

// handleGetProduct returns a specific product
func (s *InventoryServer) handleGetProduct(params interface{}, logger *log.Logger) shared.Response {
	fmt.Println("Processing get_product request...")
	logger.Print("Entered handleGetProduct")

	// Parse parameters
	var p shared.GetProductParams
	data, err := json.Marshal(params)
	if err != nil {
		logger.Print("Error marshaling at handleGetProduct, returning")
		return shared.Response{
			Action:  shared.ActionGetProduct,
			Success: false,
			Error:   "Invalid parameters format",
		}
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		logger.Print("Error unmarshaling at handleGetProduct, returning")
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
		logger.Print("Invalid parameters at handleGetProduct; product id not found, returning")
		return shared.Response{
			Action:  shared.ActionGetProduct,
			Success: false,
			Error:   fmt.Sprintf("Product with ID %s not found", p.ProductID),
		}
	}

	logger.Print("Product get ran succesfully, returning")

	return shared.Response{
		Action:  shared.ActionGetProduct,
		Success: true,
		Data:    product,
	}
}

// handleGetOrder returns a specific order
func (s *InventoryServer) handleGetOrder(params interface{}, logger *log.Logger) shared.Response {
	fmt.Println("Processing get_order request...")
	logger.Print("Entered handleGetOrder")

	// Parse parameters
	var p shared.GetOrderParams
	data, err := json.Marshal(params)
	if err != nil {
		logger.Print("Error marshaling at handleGetOrder, returning")
		return shared.Response{
			Action:  shared.ActionGetOrder,
			Success: false,
			Error:   "Invalid parameters format",
		}
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		logger.Print("Error unmarshaling at handleGetOrder, returning")
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
		logger.Print("Invalid parameters at handleGetOrder; order id not found, returning")
		return shared.Response{
			Action:  shared.ActionGetOrder,
			Success: false,
			Error:   fmt.Sprintf("Order with ID %s not found", p.OrderID),
		}
	}

	logger.Print("handleGetProduct ran succesfully, returning")
	return shared.Response{
		Action:  shared.ActionGetOrder,
		Success: true,
		Data:    order,
	}
}

// handleDeleteProduct deletes a product (only if not used in any order)
func (s *InventoryServer) handleDeleteProduct(params interface{}, logger *log.Logger) shared.Response {
	fmt.Println("Processing delete_product request...")
	logger.Print("Entered handleDeleteProduct")

	// Parse parameters
	var p shared.DeleteProductParams
	data, err := json.Marshal(params)
	if err != nil {
		logger.Print("Error marshaling at handleDeleteProduct, returning")
		return shared.Response{
			Action:  shared.ActionDeleteProduct,
			Success: false,
			Error:   "Invalid parameters format",
		}
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		logger.Print("Error unmarshaling at handleDeleteProduct, returning")
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
		logger.Print("Invalid parameter at handleDeleteProduct; product id not found, returning")
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
					logger.Print("Error deleting product; product in use on existing orders, returning")
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
		logger.Print("Error saving to file, returning")
		return shared.Response{
			Action:  shared.ActionDeleteProduct,
			Success: false,
			Error:   "Error saving to file: " + err.Error(),
		}
	}

	logger.Print("Product deleted succesfully, returning")
	return shared.Response{
		Action:  shared.ActionDeleteProduct,
		Success: true,
		Data:    map[string]string{"message": "Product deleted successfully"},
	}
}
