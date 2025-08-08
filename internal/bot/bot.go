package bot

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/NumeroQuadro/goofy-ahh-expenses-tracker/internal/data"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api      *tgbotapi.BotAPI
	data     *data.Data
	location *time.Location
}

type TransactionData struct {
	Date        string  `json:"date"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

type Transaction struct {
	Date        string
	Category    string
	Description string
	Amount      float64
}

func New(api *tgbotapi.BotAPI, data *data.Data) *Bot {
	tz := os.Getenv("DAILY_REPORT_TIMEZONE")
	if tz == "" {
		tz = "UTC"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		log.Printf("Invalid DAILY_REPORT_TIMEZONE '%s', falling back to UTC: %v", tz, err)
		loc = time.UTC
	}
	return &Bot{
		api:      api,
		data:     data,
		location: loc,
	}
}

func (b *Bot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		switch update.Message.Command() {
		case "start":
			b.handleStart(update.Message)
		case "report":
			b.handleDailyReport(update.Message)
		case "saldo":
			b.handleSaldo(update.Message)
		case "csv":
			b.handleCSVUpload(update.Message)
		case "export":
			b.handleExport(update.Message)
		case "help":
			b.handleHelp(update.Message)
		default:
			b.handleUnknownCommand(update.Message)
		}

		// Handle file uploads
		if update.Message.Document != nil {
			b.handleFileUpload(update.Message)
		}
	}
}

func (b *Bot) handleStart(msg *tgbotapi.Message) {
    text := `Welcome to the Goofy Ahh Expenses Tracker! 🎉

Available commands:
/start  — Show this message
/report — Daily spending summary (use /report YYYY-MM-DD for a specific day)
/saldo  — Today's saldo/allowance (also /saldo YYYY-MM-DD)
/csv    — Upload your CSV file
/export — Download full CSV
/help   — Help

To add expenses, use the mini app by clicking the button below.`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("📱 Open Mini App", "https://tralalero-tralala.ru/expenses/"),
		),
	)

	message := tgbotapi.NewMessage(msg.Chat.ID, text)
	message.ReplyMarkup = keyboard
	b.api.Send(message)
}

func (b *Bot) handleDailyReport(msg *tgbotapi.Message) {
	// allow optional date: /report YYYY-MM-DD
	parts := strings.Fields(msg.Text)
	var dateStr string
	if len(parts) > 1 {
		if _, err := time.Parse("2006-01-02", parts[1]); err == nil {
			dateStr = parts[1]
		}
	}
	if dateStr == "" {
		dateStr = time.Now().In(b.location).Format("2006-01-02")
	}

	// Parse selected date
	selectedDate, err := time.ParseInLocation("2006-01-02", dateStr, b.location)
	if err != nil {
		selectedDate = time.Now().In(b.location)
		dateStr = selectedDate.Format("2006-01-02")
	}

	// Today's transactions and total
	transactions := b.data.GetTransactionsByDate(dateStr)
	var todayTotal float64
	for _, tx := range transactions {
		todayTotal += tx.Amount
	}

	// Monthly budget from env (default 12000 RUB)
	monthlyBudget := 12000.0
	if mbStr := os.Getenv("MONTHLY_BUDGET_RUB"); mbStr != "" {
		if v, err := strconv.ParseFloat(mbStr, 64); err == nil && v > 0 {
			monthlyBudget = v
		}
	}

	// Compute month boundaries and stats
	lastOfMonth := time.Date(selectedDate.Year(), selectedDate.Month()+1, 0, 0, 0, 0, 0, b.location)
	daysInMonth := lastOfMonth.Day()
	dayOfMonth := selectedDate.Day()

	// Sum spent in month up to and including selected date
	var monthSpentThroughToday float64
	for _, tx := range b.data.GetAllTransactions() {
		// parse tx date
		d, err := time.ParseInLocation("2006-01-02", tx.Date, b.location)
		if err != nil {
			continue
		}
		if d.Year() == selectedDate.Year() && d.Month() == selectedDate.Month() && !d.After(selectedDate) {
			monthSpentThroughToday += tx.Amount
		}
	}

	// Even monthly distribution: allowed cumulative spend through today
	allowedCumulative := monthlyBudget * (float64(dayOfMonth) / float64(daysInMonth))
	saldoToday := allowedCumulative - monthSpentThroughToday

	// Tomorrow's allowance (dynamic), if there are days left in the month
	remainingDaysAfterToday := daysInMonth - dayOfMonth
	var tomorrowAllowance float64
	if remainingDaysAfterToday > 0 {
		remainingBudgetAfterToday := monthlyBudget - monthSpentThroughToday
		if remainingBudgetAfterToday < 0 {
			remainingBudgetAfterToday = 0
		}
		tomorrowAllowance = remainingBudgetAfterToday / float64(remainingDaysAfterToday)
	}

	var report strings.Builder
	report.WriteString(fmt.Sprintf("📊 %s\n", dateStr))
	report.WriteString(fmt.Sprintf("💰 Today: %.2f RUB\n", todayTotal))
	report.WriteString(fmt.Sprintf("🎯 Saldo today: %.2f RUB\n", saldoToday))
	if remainingDaysAfterToday > 0 {
		report.WriteString(fmt.Sprintf("➡️ Tomorrow: %.2f RUB\n", tomorrowAllowance))
	}
	if saldoToday < 0 {
		report.WriteString("⚠️ Over track for the month.")
	} else {
		report.WriteString("✅ On track.")
	}

	// Send text report
	message := tgbotapi.NewMessage(msg.Chat.ID, report.String())
	b.api.Send(message)

	// Also send full CSV export with all expenses across all months
	all := b.data.GetAllTransactions()
	var sb strings.Builder
	sb.WriteString("Date,Category,Description,Amount\n")
	for _, tx := range all {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%.2f\n", tx.Date, tx.Category, strings.ReplaceAll(tx.Description, ",", " "), tx.Amount))
	}
	doc := tgbotapi.FileBytes{Name: "expenses.csv", Bytes: []byte(sb.String())}
	msgDoc := tgbotapi.NewDocument(msg.Chat.ID, doc)
	b.api.Send(msgDoc)

}

func (b *Bot) handleCSVUpload(msg *tgbotapi.Message) {
	text := `📁 CSV Upload Instructions:

1. Your CSV file must have this exact header:
   Date,Category,Description,Amount

2. Date format: YYYY-MM-DD
3. Amount should be a number (e.g., 100.50)
4. Description is optional

Example:
Date,Category,Description,Amount
2024-01-15,Food,Lunch,500.00
2024-01-15,Transport,Bus,50.00

Send your CSV file and I'll validate and import it!`

	message := tgbotapi.NewMessage(msg.Chat.ID, text)
	b.api.Send(message)
}

func (b *Bot) handleHelp(msg *tgbotapi.Message) {
	text := `🤖 Goofy Ahh Expenses Tracker Help

Commands:
• /start - Welcome message and mini app
• /report - Get today's spending summary
• /report YYYY-MM-DD - Get spending summary for a specific date
• /saldo - Show today's saldo/allowance
• /saldo YYYY-MM-DD - Saldo for a specific date
• /csv - Upload your expense data
• /help - This help message

Features:
• Track daily expenses
• Calculate daily budget
• Upload CSV files
• Daily spending reports
• Telegram Mini App interface

For support, contact the bot administrator.`

	message := tgbotapi.NewMessage(msg.Chat.ID, text)
	b.api.Send(message)
}

func (b *Bot) handleSaldo(msg *tgbotapi.Message) {
	// Optional date arg
	parts := strings.Fields(msg.Text)
	var dateStr string
	if len(parts) > 1 {
		if _, err := time.Parse("2006-01-02", parts[1]); err == nil {
			dateStr = parts[1]
		}
	}
	if dateStr == "" {
		dateStr = time.Now().In(b.location).Format("2006-01-02")
	}

	selectedDate, err := time.ParseInLocation("2006-01-02", dateStr, b.location)
	if err != nil {
		selectedDate = time.Now().In(b.location)
		dateStr = selectedDate.Format("2006-01-02")
	}

	// Today's total
	todayTx := b.data.GetTransactionsByDate(dateStr)
	var todayTotal float64
	for _, tx := range todayTx {
		todayTotal += tx.Amount
	}

	// Monthly budget
	monthlyBudget := 12000.0
	if mbStr := os.Getenv("MONTHLY_BUDGET_RUB"); mbStr != "" {
		if v, err := strconv.ParseFloat(mbStr, 64); err == nil && v > 0 {
			monthlyBudget = v
		}
	}

	// Month stats
	lastOfMonth := time.Date(selectedDate.Year(), selectedDate.Month()+1, 0, 0, 0, 0, 0, b.location)
	daysInMonth := lastOfMonth.Day()
	dayOfMonth := selectedDate.Day()

	var monthSpentThroughToday float64
	for _, tx := range b.data.GetAllTransactions() {
		d, err := time.ParseInLocation("2006-01-02", tx.Date, b.location)
		if err != nil {
			continue
		}
		if d.Year() == selectedDate.Year() && d.Month() == selectedDate.Month() && !d.After(selectedDate) {
			monthSpentThroughToday += tx.Amount
		}
	}

	allowedCumulative := monthlyBudget * (float64(dayOfMonth) / float64(daysInMonth))
	saldoToday := allowedCumulative - monthSpentThroughToday

	remainingDaysAfterToday := daysInMonth - dayOfMonth
	var tomorrowAllowance float64
	if remainingDaysAfterToday > 0 {
		remainingBudgetAfterToday := monthlyBudget - monthSpentThroughToday
		if remainingBudgetAfterToday < 0 {
			remainingBudgetAfterToday = 0
		}
		tomorrowAllowance = remainingBudgetAfterToday / float64(remainingDaysAfterToday)
	}

	// Compose concise response
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📅 %s\n", dateStr))
	sb.WriteString(fmt.Sprintf("💳 Spent today: %.2f RUB\n", todayTotal))
	sb.WriteString(fmt.Sprintf("🎯 Allowed so far: %.2f RUB\n", allowedCumulative))
	sb.WriteString(fmt.Sprintf("💸 Saldo today: %.2f RUB\n", saldoToday))
	if remainingDaysAfterToday > 0 {
		sb.WriteString(fmt.Sprintf("➡️ Tomorrow allowance: %.2f RUB", tomorrowAllowance))
	}

	b.api.Send(tgbotapi.NewMessage(msg.Chat.ID, sb.String()))
}
func (b *Bot) handleExport(msg *tgbotapi.Message) {
	// stream current CSV data back to the user
	all := b.data.GetAllTransactions()
	var sb strings.Builder
	sb.WriteString("Date,Category,Description,Amount\n")
	for _, tx := range all {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%.2f\n", tx.Date, tx.Category, strings.ReplaceAll(tx.Description, ",", " "), tx.Amount))
	}
	doc := tgbotapi.FileBytes{Name: "expenses.csv", Bytes: []byte(sb.String())}
	msgDoc := tgbotapi.NewDocument(msg.Chat.ID, doc)
	b.api.Send(msgDoc)
}

func (b *Bot) handleUnknownCommand(msg *tgbotapi.Message) {
	text := `❓ Unknown command. Type /help for available commands.`
	message := tgbotapi.NewMessage(msg.Chat.ID, text)
	b.api.Send(message)
}

// HandleWebAppData processes data from the Telegram Mini App
func (b *Bot) HandleWebAppData(chatID int64, data string) error {
	var txData TransactionData
	if err := json.Unmarshal([]byte(data), &txData); err != nil {
		return fmt.Errorf("failed to parse web app data: %w", err)
	}

	// Validate the transaction data
	if txData.Date == "" {
		return fmt.Errorf("date is required")
	}
	if txData.Category == "" {
		return fmt.Errorf("category is required")
	}
	if txData.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	// Create transaction
	tx := Transaction{
		Date:        txData.Date,
		Category:    txData.Category,
		Description: txData.Description,
		Amount:      txData.Amount,
	}

	// Add to database using the data package's AddTransaction method
	// We'll pass the fields directly to avoid type conversion issues
	if err := b.data.AddTransaction(struct {
		Date        string
		Category    string
		Description string
		Amount      float64
	}{
		Date:        tx.Date,
		Category:    tx.Category,
		Description: tx.Description,
		Amount:      tx.Amount,
	}); err != nil {
		return fmt.Errorf("failed to save transaction: %w", err)
	}

	// Send confirmation message
	text := fmt.Sprintf("✅ Expense added!\n\n📅 Date: %s\n🏷️ Category: %s", tx.Date, tx.Category)
	if tx.Description != "" {
		text += fmt.Sprintf("\n📝 Description: %s", tx.Description)
	}
	text += fmt.Sprintf("\n💰 Amount: %.2f RUB", tx.Amount)

	message := tgbotapi.NewMessage(chatID, text)
	b.api.Send(message)

	return nil
}

func (b *Bot) handleFileUpload(msg *tgbotapi.Message) {
	// Check if it's a CSV file
	if !strings.HasSuffix(strings.ToLower(msg.Document.FileName), ".csv") {
		response := tgbotapi.NewMessage(msg.Chat.ID, "❌ Please upload a CSV file (.csv extension)")
		b.api.Send(response)
		return
	}

	// Get file from Telegram
	file, err := b.api.GetFile(tgbotapi.FileConfig{FileID: msg.Document.FileID})
	if err != nil {
		log.Printf("Failed to get file: %v", err)
		response := tgbotapi.NewMessage(msg.Chat.ID, "❌ Failed to download file")
		b.api.Send(response)
		return
	}

	// Download file content
	resp, err := http.Get(file.Link(b.api.Token))
	if err != nil {
		log.Printf("Failed to download file content: %v", err)
		response := tgbotapi.NewMessage(msg.Chat.ID, "❌ Failed to download file content")
		b.api.Send(response)
		return
	}
	defer resp.Body.Close()

	// Parse CSV content
	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil {
		response := tgbotapi.NewMessage(msg.Chat.ID, "❌ Invalid CSV format")
		b.api.Send(response)
		return
	}

	if len(records) == 0 {
		response := tgbotapi.NewMessage(msg.Chat.ID, "❌ CSV file is empty")
		b.api.Send(response)
		return
	}

	// Validate header
	expectedHeader := []string{"Date", "Category", "Description", "Amount"}
	if len(records[0]) != 4 || !compareStringSlices(records[0], expectedHeader) {
		response := tgbotapi.NewMessage(msg.Chat.ID, "❌ CSV header must be: Date,Category,Description,Amount")
		b.api.Send(response)
		return
	}

	// Process transactions
	var transactions []Transaction
	var errors []string
	var totalAmount float64

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

		tx := Transaction{
			Date:        record[0],
			Category:    record[1],
			Description: record[2],
			Amount:      amount,
		}

		transactions = append(transactions, tx)
		totalAmount += amount
	}

	// If there are validation errors, send them
	if len(errors) > 0 {
		errorMsg := "❌ CSV validation failed:\n\n"
		for _, err := range errors[:10] { // Limit to first 10 errors
			errorMsg += "• " + err + "\n"
		}
		if len(errors) > 10 {
			errorMsg += fmt.Sprintf("\n... and %d more errors", len(errors)-10)
		}
		response := tgbotapi.NewMessage(msg.Chat.ID, errorMsg)
		b.api.Send(response)
		return
	}

	// Add all valid transactions
	for _, tx := range transactions {
		if err := b.data.AddTransaction(struct {
			Date        string
			Category    string
			Description string
			Amount      float64
		}{
			Date:        tx.Date,
			Category:    tx.Category,
			Description: tx.Description,
			Amount:      tx.Amount,
		}); err != nil {
			log.Printf("Failed to save transaction: %v", err)
			response := tgbotapi.NewMessage(msg.Chat.ID, "❌ Failed to save transactions")
			b.api.Send(response)
			return
		}
	}

	// Send success message
	successMsg := fmt.Sprintf("✅ Successfully imported %d transactions!\n\n", len(transactions))
	successMsg += fmt.Sprintf("💰 Total amount: %.2f RUB\n", totalAmount)
	successMsg += fmt.Sprintf("📅 Date range: %s to %s", transactions[0].Date, transactions[len(transactions)-1].Date)

	response := tgbotapi.NewMessage(msg.Chat.ID, successMsg)
	b.api.Send(response)
}

// SendDailyReport sends the daily expense report to all users
func (b *Bot) SendDailyReport() error {
	// This would typically iterate through all registered users
	// For now, we'll just log that the report would be sent
	log.Println("Daily report would be sent at this time")
	return nil
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
