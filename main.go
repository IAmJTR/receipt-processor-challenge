package main

import (
	"fmt"
	"log"
	"net/http"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/google/uuid"
	"strconv"
	"strings"
	"unicode"
)

// In-memory storage for receipts
var receiptStore = make(map[string]Receipt)

// Receipt represents the structure of the receipt.
type Receipt struct {
	Retailer     string  `json:"retailer"`
	PurchaseDate string  `json:"purchaseDate"`
	PurchaseTime string  `json:"purchaseTime"`
	Total        string  `json:"total"`
	Items        []Item  `json:"items"`
}

// Item represents the structure of items in a receipt.
type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

// ProcessReceipt handles POST /receipts/process
func ProcessReceipt(w http.ResponseWriter, r *http.Request) {
	var receipt Receipt
	if err := json.NewDecoder(r.Body).Decode(&receipt); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Generate a unique receipt ID
	id := uuid.New().String()

	// Store the receipt data in memory (temporary)
	receiptStore[id] = receipt

	// Send the response with the receipt ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

// GetPoints handles GET /receipts/{id}/points
func GetPoints(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Retrieve the receipt data for the given ID
	receipt, exists := receiptStore[id]
	if !exists {
		http.Error(w, "Receipt not found", http.StatusNotFound)
		return
	}

	// Calculate points for the receipt
	points := calculatePoints(receipt)

	// Send the response with the points
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"points": points})
}

// calculatePoints calculates the points based on the receipt data
func calculatePoints(receipt Receipt) int {
	points := 0

	// 1. One point for every alphanumeric character in the retailer name
	points += alphaNumericCount(receipt.Retailer)

	// 2. 50 points if the total is a round dollar amount with no cents
	total, err := strconv.ParseFloat(receipt.Total, 64)
	if err == nil && total == float64(int(total)) {
		points += 50
	}

	// 3. 25 points if the total is a multiple of 0.25
	if int(total * 100) % 25 == 0 {
		points += 25
	}

	// 4. 5 points for every two items on the receipt
	points += int(len(receipt.Items) / 2) * 5

	// 5. If the trimmed length of the item description is a multiple of 3, multiply the price by 0.2 and round up
	for _, item := range receipt.Items {
		itemDescLength := len(strings.TrimSpace(item.ShortDescription))
		if itemDescLength % 3 == 0 {
			price, err := strconv.ParseFloat(item.Price, 64)
			if err == nil {
				points += int(price * 0.2) + 1 // Round up
			}
		}
	}

	// 6. 6 points if the day in the purchase date is odd
	day, err := strconv.Atoi(strings.Split(receipt.PurchaseDate, "-")[2])
	if err == nil && day % 2 == 1 {
		points += 6
	}

	// 7. 10 points if the time of purchase is after 2:00pm and before 4:00pm
	hour, err := parseTime(receipt.PurchaseTime)
	if err == nil {
		// Check if the hour is between 14 (2:00 PM) and 16 (4:00 PM)
		if hour >= 14 && hour < 16 {
			points += 10
		}
	}

	return points
}

// alphaNumericCount counts the alpha numeric characters retailer name
func alphaNumericCount(s string) int {
	count := 0
	for _, char := range s {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			count++
		}
	}
	return count
}

// parseTime parses a time in 24-hour format (HH:MM) and returns the hour
func parseTime(timeStr string) (int, error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		// Return error if the time format is invalid
		return 0, fmt.Errorf("invalid time format")
	}
	
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hour")
	}
	
	return hour, nil
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/receipts/process", ProcessReceipt).Methods("POST")
	r.HandleFunc("/receipts/{id}/points", GetPoints).Methods("GET")

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
