package data

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
)

type Transaction struct {
	Date        string
	Category    string
	Description string
	Amount      float64
}

type Data struct {
	mu           sync.Mutex
	dataPath     string
	Transactions []Transaction
}

func New(dataPath string) (*Data, error) {
	d := &Data{
		dataPath: dataPath,
	}
	if err := d.load(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *Data) load() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	file, err := os.Open(d.dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create the file if it does not exist
			file, err = os.Create(d.dataPath)
			if err != nil {
				return err
			}
			file.Close()
			return nil
		}
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	if len(records) == 0 {
		return nil // Empty file, no transactions
	}

	// Validate header
	expectedHeader := []string{"Date", "Category", "Description", "Amount"}
	if !compareStringSlices(records[0], expectedHeader) {
		return errors.New("CSV header does not match expected format")
	}

	d.Transactions = make([]Transaction, 0, len(records)-1)
	for i, record := range records[1:] { // Skip header row
		if len(record) != 4 {
			return fmt.Errorf("invalid record length on line %d: expected 4 fields, got %d", i+2, len(record))
		}
		amount, err := strconv.ParseFloat(record[3], 64)
		if err != nil {
			return fmt.Errorf("invalid amount on line %d: %w", i+2, err)
		}
		d.Transactions = append(d.Transactions, Transaction{
			Date:        record[0],
			Category:    record[1],
			Description: record[2],
			Amount:      amount,
		})
	}

	return nil
}

func (d *Data) AddTransaction(tx Transaction) error {
	d.mu.Lock()
	d.Transactions = append(d.Transactions, tx)
	d.mu.Unlock()

	return d.save()
}

func (d *Data) save() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	file, err := os.Create(d.dataPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"Date", "Category", "Description", "Amount"})

	for _, tx := range d.Transactions {
		err := writer.Write([]string{
			tx.Date,
			tx.Category,
			tx.Description,
			strconv.FormatFloat(tx.Amount, 'f', 2, 64),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Data) GetTransactionsByDate(date string) []Transaction {
	d.mu.Lock()
	defer d.mu.Unlock()

	var result []Transaction
	for _, tx := range d.Transactions {
		if tx.Date == date {
			result = append(result, tx)
		}
	}
	return result
}

func (d *Data) GetAllTransactions() []Transaction {
	d.mu.Lock()
	defer d.mu.Unlock()

	result := make([]Transaction, len(d.Transactions))
	copy(result, d.Transactions)
	return result
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
