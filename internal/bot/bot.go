package bot

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/NumeroQuadro/goofy-ahh-expenses-tracker/internal/data"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api  *tgbotapi.BotAPI
	data *data.Data
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
	return &Bot{
		api:  api,
		data: data,
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
		case "csv":
			b.handleCSVUpload(update.Message)
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
/start - Show this message
/report - Get today's spending report
/csv - Upload your CSV file
/help - Show help

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
	today := time.Now().Format("2006-01-02")
	transactions := b.data.GetTransactionsByDate(today)

	var total float64
	var report strings.Builder
	report.WriteString(fmt.Sprintf("📊 Daily Report for %s\n\n", today))

	if len(transactions) == 0 {
		report.WriteString("No expenses recorded today! 🎉")
	} else {
		report.WriteString("Today's expenses:\n")
		for _, tx := range transactions {
			report.WriteString(fmt.Sprintf("• %s: %.2f RUB", tx.Category, tx.Amount))
			if tx.Description != "" {
				report.WriteString(fmt.Sprintf(" (%s)", tx.Description))
			}
			report.WriteString("\n")
			total += tx.Amount
		}
		report.WriteString(fmt.Sprintf("\n💰 Total: %.2f RUB", total))

		// Calculate daily budget (example: 1000 RUB per day)
		dailyBudget := 1000.0
		remaining := dailyBudget - total
		report.WriteString(fmt.Sprintf("\n🎯 Daily Budget: %.2f RUB", dailyBudget))
		report.WriteString(fmt.Sprintf("\n💸 Remaining: %.2f RUB", remaining))

		if remaining < 0 {
			report.WriteString(" ⚠️ Over budget!")
		} else if remaining < 100 {
			report.WriteString(" ⚠️ Low budget!")
		} else {
			report.WriteString(" ✅ Good!")
		}
	}

	message := tgbotapi.NewMessage(msg.Chat.ID, report.String())
	b.api.Send(message)
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
