package web

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

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
		expenses.GET("/graph", s.handleGraph)
		expenses.GET("/graph-data", s.handleGraphData)
		expenses.POST("/transaction", s.handleTransaction)
		expenses.POST("/upload-csv", s.handleCSVUpload)
		expenses.GET("/transactions", s.handleGetTransactions)
	}

	return s
}

// --- Graph pages & data ---

func (s *Server) handleGraph(c *gin.Context) {
	c.HTML(http.StatusOK, "graph.html", gin.H{
		"title": "Expense Graph",
	})
}

func (s *Server) handleGraphData(c *gin.Context) {
	type point struct {
		Date       string  `json:"date"`
		Spend      float64 `json:"spend"`
		Cumulative float64 `json:"cumulative"`
		BudgetCum  float64 `json:"budget_cum"`
		Saldo      float64 `json:"saldo"`
	}

	fromStr := c.Query("from")
	toStr := c.Query("to")

	// Read budget from env; default 12000 (see OVERVIEW.md)
	budgetMonthly := 12000.0
	if v := os.Getenv("MONTHLY_BUDGET_RUB"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			budgetMonthly = f
		}
	}

	// Build daily sum map
	txs := s.data.GetAllTransactions()
	daySum := map[string]float64{}
	const layout = "2006-01-02"
	minDate, maxDate := "", ""

	for _, tx := range txs {
		daySum[tx.Date] += tx.Amount
		if minDate == "" || tx.Date < minDate {
			minDate = tx.Date
		}
		if maxDate == "" || tx.Date > maxDate {
			maxDate = tx.Date
		}
	}

	// Default window: last 90 days (or full range if fewer)
	var from, to time.Time
	now := time.Now()
	defaultFrom := now.AddDate(0, 0, -90)

	if fromStr != "" {
		if t, err := time.Parse(layout, fromStr); err == nil {
			from = t
		}
	}
	if toStr != "" {
		if t, err := time.Parse(layout, toStr); err == nil {
			to = t
		}
	}

	// Fallbacks
	if from.IsZero() || to.IsZero() {
		if minDate != "" && maxDate != "" {
			mi, _ := time.Parse(layout, minDate)
			ma, _ := time.Parse(layout, maxDate)
			if from.IsZero() {
				if ma.Before(defaultFrom) {
					from = mi
				} else {
					from = defaultFrom
				}
			}
			if to.IsZero() {
				to = ma
			}
		} else {
			// No data: show last 30 days
			from = now.AddDate(0, 0, -30)
			to = now
		}
	}

	if from.After(to) {
		from, to = to, from
	}

	// Walk inclusive date range and compute series
	var res []point
	var cum float64
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format(layout)
		spend := daySum[key]

		// Per-month budget curve: monthly budget * (dayIndex / daysInMonth)
		firstOfMonth := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.UTC)
		daysInMonth := firstOfMonth.AddDate(0, 1, -1).Day()
		dayIndex := d.Day()
		budgetCum := budgetMonthly * float64(dayIndex) / float64(daysInMonth)

		// Reset cumulative at month start to reflect budget period
		if dayIndex == 1 {
			cum = 0
		}
		cum += spend

		res = append(res, point{
			Date:       key,
			Spend:      spend,
			Cumulative: cum,
			BudgetCum:  budgetCum,
			Saldo:      budgetCum - cum,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"from":          from.Format(layout),
		"to":            to.Format(layout),
		"monthlyBudget": budgetMonthly,
		"points":        res,
	})
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
		// Allow empty upload to reset data
		s.data.Clear()
		c.JSON(http.StatusOK, gin.H{"message": "Data reset (empty upload)"})
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
		// Empty file → clear data
		s.data.Clear()
		c.JSON(http.StatusOK, gin.H{"message": "Data reset (empty CSV)"})
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

	// Replace existing data with uploaded set atomically
	if err := s.data.ReplaceAll(transactions); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save transactions"})
		return
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
