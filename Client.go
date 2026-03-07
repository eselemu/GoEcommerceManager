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

func main() {
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
		request, err := parseCommand(text)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Print("> ")
			continue
		}

		// Send request as JSON
		encoder := json.NewEncoder(conn)
		err = encoder.Encode(request)
		if err != nil {
			log.Fatal("Error sending request:", err)
		}

		fmt.Print("> ")
	}
}

func loadConfig(filename string) (*shared.Config, error) { // Use shared.Config
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config shared.Config // Use shared.Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func parseCommand(input string) (shared.Request, error) { // Use shared.Request
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
		return request, fmt.Errorf("empty command")
	}

	command := args[0]

	switch command {
	case "add_product":
		if len(args) != 4 {
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

	case "update_stock":
		if len(args) != 3 {
			return request, fmt.Errorf("usage: update_stock <product_id> <new_stock>")
		}
		var stock int
		fmt.Sscanf(args[2], "%d", &stock)

		request.Action = "update_stock"
		request.Parameters = shared.UpdateStockParams{ // Use shared.UpdateStockParams
			ProductID: args[1],
			NewStock:  stock,
		}

	case "update_price":
		if len(args) != 3 {
			return request, fmt.Errorf("usage: update_price <product_id> <new_price>")
		}
		var price float64
		fmt.Sscanf(args[2], "%f", &price)

		request.Action = "update_price"
		request.Parameters = shared.UpdatePriceParams{ // Use shared.UpdatePriceParams
			ProductID: args[1],
			NewPrice:  price,
		}

	case "create_order":
		if len(args) != 2 {
			return request, fmt.Errorf("usage: create_order <product_id1:quantity1,product_id2:quantity2,...>")
		}
		// Parse items with format "product_id:quantity"
		items := strings.Split(args[1], ",")
		var orderItems []shared.OrderItemRequest

		for _, item := range items {
			parts := strings.Split(item, ":")
			if len(parts) != 2 {
				return request, fmt.Errorf("invalid item format: %s (should be product_id:quantity)", item)
			}

			quantity, err := strconv.Atoi(parts[1])
			if err != nil {
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

	case "update_order_status":
		if len(args) != 3 {
			return request, fmt.Errorf("usage: update_order_status <order_id> <status>")
		}
		request.Action = "update_order_status"
		request.Parameters = shared.UpdateOrderStatusParams{ // Use shared.UpdateOrderStatusParams
			OrderID: args[1],
			Status:  args[2],
		}

	case "list_products":
		request.Action = "list_products"
		request.Parameters = nil

	case "list_orders":
		request.Action = "list_orders"
		request.Parameters = nil

	case "get_product":
		if len(args) != 2 {
			return request, fmt.Errorf("usage: get_product <product_id>")
		}
		request.Action = shared.ActionGetProduct
		request.Parameters = shared.GetProductParams{
			ProductID: args[1],
		}

	case "get_order":
		if len(args) != 2 {
			return request, fmt.Errorf("usage: get_order <order_id>")
		}
		request.Action = shared.ActionGetOrder
		request.Parameters = shared.GetOrderParams{
			OrderID: args[1],
		}

	case "delete_product":
		if len(args) != 2 {
			return request, fmt.Errorf("usage: delete_product <product_id>")
		}
		request.Action = shared.ActionDeleteProduct
		request.Parameters = shared.DeleteProductParams{
			ProductID: args[1],
		}

	default:
		return request, fmt.Errorf("unknown command: %s", command)
	}

	return request, nil
}
