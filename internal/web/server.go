package web

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/NumeroQuadro/goofy-ahh-expenses-tracker/internal/data"
	"github.com/gin-gonic/gin"
)

type Server struct {
	router *gin.Engine
	data   *data.Data
	bot    BotHandler
}

type BotHandler interface {
	HandleWebAppData(chatID int64, data string) error
}

type TransactionRequest struct {
	Date        string  `json:"date"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	ChatID      int64   `json:"chat_id"`
}

func New(data *data.Data, bot BotHandler) *Server {
	r := gin.Default()

	// Load HTML templates
	r.LoadHTMLGlob("static/*.html")

	// Serve static files under the app prefix so it works behind reverse proxy
	r.Static("/expenses/static", "./static")

	s := &Server{
		router: r,
		data:   data,
		bot:    bot,
	}

	// Routes
	expenses := r.Group("/expenses")
	{
		expenses.GET("/", s.handleIndex)
		expenses.POST("/transaction", s.handleTransaction)
		expenses.POST("/upload-csv", s.handleCSVUpload)
		expenses.GET("/transactions", s.handleGetTransactions)
	}

	return s
}

func (s *Server) handleIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "Expense Tracker",
	})
}

func (s *Server) handleTransaction(c *gin.Context) {
	var req TransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate required fields
	if req.Date == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Date is required"})
		return
	}
	if req.Category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Category is required"})
		return
	}
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount must be positive"})
		return
	}

	// If chat ID is provided, let the bot handle persistence + confirmation to avoid duplicate saves
	if req.ChatID != 0 {
		transactionData := map[string]interface{}{
			"date":        req.Date,
			"category":    req.Category,
			"description": req.Description,
			"amount":      req.Amount,
		}

		jsonData, _ := json.Marshal(transactionData)
		if err := s.bot.HandleWebAppData(req.ChatID, string(jsonData)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Transaction added via Telegram",
		})
		return
	}

	// Otherwise, persist directly
	tx := data.Transaction{
		Date:        req.Date,
		Category:    req.Category,
		Description: req.Description,
		Amount:      req.Amount,
	}
	if err := s.data.AddTransaction(tx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaction added successfully", "transaction": tx})
}

func (s *Server) handleCSVUpload(c *gin.Context) {
	file, err := c.FormFile("csv")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No CSV file provided"})
		return
	}

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open uploaded file"})
		return
	}
	defer src.Close()

	// Read and parse CSV
	reader := csv.NewReader(src)
	records, err := reader.ReadAll()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid CSV format"})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file is empty"})
		return
	}

	// Validate header
	expectedHeader := []string{"Date", "Category", "Description", "Amount"}
	if len(records[0]) != 4 || !compareStringSlices(records[0], expectedHeader) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV header must be: Date,Category,Description,Amount"})
		return
	}

	// Process transactions
	var transactions []data.Transaction
	var errors []string

	for i, record := range records[1:] {
		if len(record) != 4 {
			errors = append(errors, fmt.Sprintf("Line %d: Invalid number of fields", i+2))
			continue
		}

		amount, err := strconv.ParseFloat(record[3], 64)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Line %d: Invalid amount '%s'", i+2, record[3]))
			continue
		}

		if amount <= 0 {
			errors = append(errors, fmt.Sprintf("Line %d: Amount must be positive", i+2))
			continue
		}

		tx := data.Transaction{
			Date:        record[0],
			Category:    record[1],
			Description: record[2],
			Amount:      amount,
		}

		transactions = append(transactions, tx)
	}

	// If there are validation errors, return them
	if len(errors) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "CSV validation failed",
			"errors": errors,
		})
		return
	}

	// Add all valid transactions
	for _, tx := range transactions {
		if err := s.data.AddTransaction(tx); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save transactions"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Successfully imported %d transactions", len(transactions)),
		"count":   len(transactions),
	})
}

func (s *Server) handleGetTransactions(c *gin.Context) {
	date := c.Query("date")

	var transactions []data.Transaction
	if date != "" {
		transactions = s.data.GetTransactionsByDate(date)
	} else {
		transactions = s.data.GetAllTransactions()
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"count":        len(transactions),
	})
}

func (s *Server) Start(address string, certPath string, keyPath string) error {
	// Check if we're running in Docker with mounted certificates
	if certPath == "" && keyPath == "" {
		// Try to find certificates in the standard Docker location
		dockerCertPath := "/app/certs/fullchain.pem"
		dockerKeyPath := "/app/certs/privkey.pem"

		if _, err := os.Stat(dockerCertPath); err == nil {
			if _, err := os.Stat(dockerKeyPath); err == nil {
				certPath = dockerCertPath
				keyPath = dockerKeyPath
			}
		}
	}

	if certPath != "" && keyPath != "" {
		// Verify certificate files exist
		if _, err := os.Stat(certPath); err != nil {
			return fmt.Errorf("certificate file not found: %s", certPath)
		}
		if _, err := os.Stat(keyPath); err != nil {
			return fmt.Errorf("private key file not found: %s", keyPath)
		}

		log.Printf("Starting HTTPS server on %s with certificates: %s, %s", address, certPath, keyPath)
		return s.router.RunTLS(address, certPath, keyPath)
	}

	log.Printf("Starting HTTP server on %s", address)
	return s.router.Run(address)
}

func compareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
