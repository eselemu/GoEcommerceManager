package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"inventory-management/shared" // Import the shared package
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

var (
	mainLogger         *log.Logger
	parseCommandLogger *log.Logger
)

func main() {
	//Create log file
	file, err := os.OpenFile("info.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Panic("Failed to open log file")
	}
	defer file.Close()

	//Temporary prefixes for logs to log handle connection
	mainLogger = log.New(file, "[Client] main(): ", log.LstdFlags)
	parseCommandLogger = log.New(file, "[Client] parseCommand(): ", log.LstdFlags)
	// Load configuration
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	// Connect to the server
	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal("Error connecting to server:", err)
	}
	//Change prefix to add address
	mainLogger.SetPrefix(fmt.Sprintf("[Client %s] main(): ", addr))
	parseCommandLogger.SetPrefix(fmt.Sprintf("[Client %s] parseCommand(): ", addr))

	mainLogger.Print("Succesfully connected to server")
	defer conn.Close()

	fmt.Println("Available commands:")
	fmt.Println("  add_product <name> <price> <stock>")
	fmt.Println("  update_stock <product_id> <new_stock>")
	fmt.Println("  update_price <product_id> <new_price>")
	fmt.Println("  get_product <product_id>")
	fmt.Println("  delete_product <product_id>")
	fmt.Println("  list_products")
	fmt.Println("  create_order <product_id1:quantity1,product_id2:quantity2,...>")
	fmt.Println("  update_order_status <order_id> <status>")
	fmt.Println("  get_order <order_id>")
	fmt.Println("  list_orders")
	fmt.Println("  exit")

	var mu sync.Mutex
	connected := true

	// Goroutine to read responses from server
	go func() {
		decoder := json.NewDecoder(conn)
		for {
			var resp shared.Response // Use shared.Response
			err := decoder.Decode(&resp)
			if err != nil {
				fmt.Println("\nDisconnected from server")
				mu.Lock()
				connected = false
				mu.Unlock()
				return
			}
			// Pretty print the response
			fmt.Println("\n=== Server Response ===")
			fmt.Printf("Action: %s\n", resp.Action)
			fmt.Printf("Success: %v\n", resp.Success)
			if resp.Error != "" {
				fmt.Printf("Error: %s\n", resp.Error)
			}
			if resp.Data != nil {
				fmt.Printf("Data: %+v\n", resp.Data)
			}
			fmt.Println("=======================\n")
			fmt.Print("> ")
			mainLogger.Print("Server response logged")
		}
	}()

	// Main goroutine to read user input
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")

	for scanner.Scan() {
		mu.Lock()
		if !connected {
			mu.Unlock()
			fmt.Println("Server disconnected. Exiting...")
			break
		}
		mu.Unlock()

		text := scanner.Text()
		if text == "exit" {
			break
		}

		// Parse and send command
		request, err := parseCommand(text, addr)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Print("> ")
			continue
		}
		mainLogger.Print("Sending command to server")

		// Send request as JSON
		encoder := json.NewEncoder(conn)
		err = encoder.Encode(request)
		if err != nil {
			mainLogger.Fatal("Error sending request:", err)
		}

		fmt.Print("> ")
		mainLogger.Print("Sent command to server succesfully")
	}
	mainLogger.Print("Connection closing")
}

func loadConfig(filename string) (*shared.Config, error) { // Use shared.Config
	mainLogger.Print("Client LoadConfig():")
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	mainLogger.Print("Config file opened")
	var config shared.Config // Use shared.Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	mainLogger.Print("Config file loaded succesfully")
	return &config, nil
}

func parseCommand(input string, addr string) (shared.Request, error) { // Use shared.Request
	parseCommandLogger.SetPrefix(fmt.Sprintf("Client in %s ParseCommand():", addr))
	var request shared.Request // Use shared.Request
	var args []string
	var inQuotes bool
	var current string

	// Simple argument parsing (handles quoted strings)
	for _, r := range input {
		if r == '"' {
			inQuotes = !inQuotes
		} else if r == ' ' && !inQuotes {
			if current != "" {
				args = append(args, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		args = append(args, current)
	}

	if len(args) == 0 {
		parseCommandLogger.Print("Command empty")
		return request, fmt.Errorf("empty command")
	}

	command := args[0]

	switch command {
	case "add_product":
		parseCommandLogger.Print("Attempting to add item")
		if len(args) != 4 {
			parseCommandLogger.Print("Add product parameters incorrect, returning")
			return request, fmt.Errorf("usage: add_product <name> <price> <stock>")
		}
		var price float64
		var stock int
		fmt.Sscanf(args[2], "%f", &price)
		fmt.Sscanf(args[3], "%d", &stock)

		request.Action = "add_product"
		request.Parameters = shared.AddProductParams{ // Use shared.AddProductParams
			Name:  args[1],
			Price: price,
			Stock: stock,
		}
		parseCommandLogger.Print("Added product succesfully")

	case "update_stock":
		parseCommandLogger.Print("Attempting to update stock")
		if len(args) != 3 {
			parseCommandLogger.Print("Update product stock parameters incorrect, returning")
			return request, fmt.Errorf("usage: update_stock <product_id> <new_stock>")
		}
		var stock int
		fmt.Sscanf(args[2], "%d", &stock)

		request.Action = "update_stock"
		request.Parameters = shared.UpdateStockParams{ // Use shared.UpdateStockParams
			ProductID: args[1],
			NewStock:  stock,
		}
		parseCommandLogger.Print("Updated product stock succesfully")

	case "update_price":
		parseCommandLogger.Print("Attempting to update price")
		if len(args) != 3 {
			parseCommandLogger.Print("Price update parameters incorrect, returning")
			return request, fmt.Errorf("usage: update_price <product_id> <new_price>")
		}
		var price float64
		fmt.Sscanf(args[2], "%f", &price)

		request.Action = "update_price"
		request.Parameters = shared.UpdatePriceParams{ // Use shared.UpdatePriceParams
			ProductID: args[1],
			NewPrice:  price,
		}
		parseCommandLogger.Print("Price updated succesfully")

	case "create_order":
		parseCommandLogger.Print("Attempting to create order")
		if len(args) != 2 {
			parseCommandLogger.Print("Order parameters incorrect, returning")
			return request, fmt.Errorf("usage: create_order <product_id1:quantity1,product_id2:quantity2,...>")
		}
		// Parse items with format "product_id:quantity"
		items := strings.Split(args[1], ",")
		var orderItems []shared.OrderItemRequest

		for _, item := range items {
			parts := strings.Split(item, ":")
			if len(parts) != 2 {
				parseCommandLogger.Print("Items in order misplaced, returning")
				return request, fmt.Errorf("invalid item format: %s (should be product_id:quantity)", item)
			}

			quantity, err := strconv.Atoi(parts[1])
			if err != nil {
				parseCommandLogger.Print("Invalid item quantity, returning")
				return request, fmt.Errorf("invalid quantity for item %s: %v", parts[0], err)
			}

			orderItems = append(orderItems, shared.OrderItemRequest{
				ProductID: parts[0],
				Quantity:  quantity,
			})
		}

		request.Action = shared.ActionCreateOrder
		request.Parameters = shared.CreateOrderParams{
			Items: orderItems,
		}
		parseCommandLogger.Print("Order placed succesfully")

	case "update_order_status":
		parseCommandLogger.Print("Attempting to update order stauts")
		if len(args) != 3 {
			parseCommandLogger.Print("Update order parameters incorrect, returning")
			return request, fmt.Errorf("usage: update_order_status <order_id> <status>")
		}
		request.Action = "update_order_status"
		request.Parameters = shared.UpdateOrderStatusParams{ // Use shared.UpdateOrderStatusParams
			OrderID: args[1],
			Status:  args[2],
		}
		parseCommandLogger.Print("Order status updated succesfully")

	case "list_products":
		parseCommandLogger.Print("Listing products")
		request.Action = "list_products"
		request.Parameters = nil

	case "list_orders":
		parseCommandLogger.Print("Listing orders")
		request.Action = "list_orders"
		request.Parameters = nil

	case "get_product":
		parseCommandLogger.Print("Attempting to get product")
		if len(args) != 2 {
			parseCommandLogger.Print("Get product parameters incorrect, returning")
			return request, fmt.Errorf("usage: get_product <product_id>")
		}
		request.Action = shared.ActionGetProduct
		request.Parameters = shared.GetProductParams{
			ProductID: args[1],
		}
		parseCommandLogger.Print("Get product ran succesfully")

	case "get_order":
		parseCommandLogger.Print("Attempting to get order")
		if len(args) != 2 {
			parseCommandLogger.Print("Get order parameters incorrect, returning")
			return request, fmt.Errorf("usage: get_order <order_id>")
		}
		request.Action = shared.ActionGetOrder
		request.Parameters = shared.GetOrderParams{
			OrderID: args[1],
		}
		parseCommandLogger.Print("Get order ran succesfully")

	case "delete_product":
		parseCommandLogger.Print("Attempting to delete product")
		if len(args) != 2 {
			parseCommandLogger.Print("Delete product parameters incorrect, returning")
			return request, fmt.Errorf("usage: delete_product <product_id>")
		}
		request.Action = shared.ActionDeleteProduct
		request.Parameters = shared.DeleteProductParams{
			ProductID: args[1],
		}
		parseCommandLogger.Print("Deleted product successfully")

	default:
		parseCommandLogger.Print("Unkown command, returning")
		return request, fmt.Errorf("unknown command: %s", command)
	}

	return request, nil
}
