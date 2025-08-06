package data

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNewAndLoad(t *testing.T) {
	// Create a temporary CSV file for testing
	tempDir, err := ioutil.TempDir("", "testdata")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	csvPath := filepath.Join(tempDir, "test.csv")

	// Test case 1: New file, should create an empty CSV with header
	d, err := New(csvPath)
	if err != nil {
		t.Fatalf("New() failed for new file: %v", err)
	}
	if len(d.Transactions) != 0 {
		t.Errorf("Expected 0 transactions for new file, got %d", len(d.Transactions))
	}

	// Test case 2: Load existing valid CSV
	validCSVContent := "Date,Category,Description,Amount\n2023-01-01,Food,Lunch,10.50\n2023-01-02,Transport,Bus,2.00\n"
	if err := ioutil.WriteFile(csvPath, []byte(validCSVContent), 0644); err != nil {
		t.Fatalf("Failed to write valid CSV: %v", err)
	}

	d, err = New(csvPath)
	if err != nil {
		t.Fatalf("New() failed for valid CSV: %v", err)
	}
	expectedTransactions := []Transaction{
		{Date: "2023-01-01", Category: "Food", Description: "Lunch", Amount: 10.50},
		{Date: "2023-01-02", Category: "Transport", Description: "Bus", Amount: 2.00},
	}
	if !reflect.DeepEqual(d.Transactions, expectedTransactions) {
		t.Errorf("Loaded transactions mismatch.\nExpected: %+v\nGot: %+v", expectedTransactions, d.Transactions)
	}

	// Test case 3: Load CSV with invalid header
	invalidHeaderCSVContent := "Date,Category,Description,Invalid\n2023-01-01,Food,Lunch,10.50\n"
	if err := ioutil.WriteFile(csvPath, []byte(invalidHeaderCSVContent), 0644); err != nil {
		t.Fatalf("Failed to write invalid header CSV: %v", err)
	}
	d, err = New(csvPath)
	if err == nil || err.Error() != "CSV header does not match expected format" {
		t.Errorf("Expected 'CSV header does not match expected format' error, got %v", err)
	}

	// Test case 4: Load CSV with invalid amount
	invalidAmountCSVContent := "Date,Category,Description,Amount\n2023-01-01,Food,Lunch,abc\n"
	if err := ioutil.WriteFile(csvPath, []byte(invalidAmountCSVContent), 0644); err != nil {
		t.Fatalf("Failed to write invalid amount CSV: %v", err)
	}
	d, err = New(csvPath)
	if err == nil || err.Error() != "invalid amount on line 2: strconv.ParseFloat: parsing \"abc\": invalid syntax" {
		t.Errorf("Expected 'invalid amount' error, got %v", err)
	}

	// Test case 5: Load CSV with invalid record length
	invalidLengthCSVContent := "Date,Category,Description,Amount\n2023-01-01,Food,Lunch\n"
	if err := ioutil.WriteFile(csvPath, []byte(invalidLengthCSVContent), 0644); err != nil {
		t.Fatalf("Failed to write invalid length CSV: %v", err)
	}
	d, err = New(csvPath)
	if err == nil || !strings.Contains(err.Error(), "record on line 2: wrong number of fields") {
		t.Errorf("Expected 'record on line 2: wrong number of fields' error, got %v", err)
	}
}

func TestAddTransactionAndSave(t *testing.T) {
	// Create a temporary CSV file for testing
	tempDir, err := ioutil.TempDir("", "testdata")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	csvPath := filepath.Join(tempDir, "test_add_save.csv")

	d, err := New(csvPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	tx1 := Transaction{Date: "2023-03-01", Category: "Shopping", Description: "Shirt", Amount: 25.99}
	if err := d.AddTransaction(tx1); err != nil {
		t.Fatalf("AddTransaction failed: %v", err)
	}

	tx2 := Transaction{Date: "2023-03-02", Category: "Utilities", Description: "Electricity", Amount: 50.00}
	if err := d.AddTransaction(tx2); err != nil {
		t.Fatalf("AddTransaction failed: %v", err)
	}

	expectedTransactions := []Transaction{tx1, tx2}
	if !reflect.DeepEqual(d.Transactions, expectedTransactions) {
		t.Errorf("Transactions after add mismatch.\nExpected: %+v\nGot: %+v", expectedTransactions, d.Transactions)
	}

	// Verify content of the saved CSV file
	savedContent, err := ioutil.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("Failed to read saved CSV: %v", err)
	}
	expectedSavedContent := "Date,Category,Description,Amount\n2023-03-01,Shopping,Shirt,25.99\n2023-03-02,Utilities,Electricity,50.00\n"
	if string(savedContent) != expectedSavedContent {
		t.Errorf("Saved CSV content mismatch.\nExpected:\n%s\nGot:\n%s", expectedSavedContent, string(savedContent))
	}
}

func TestCompareStringSlices(t *testing.T) {
	tests := []struct {
		name string
		a    []string
		b    []string
		want bool
	}{
		{"equal slices", []string{"a", "b", "c"}, []string{"a", "b", "c"}, true},
		{"different order", []string{"a", "b", "c"}, []string{"a", "c", "b"}, false},
		{"different length", []string{"a", "b"}, []string{"a", "b", "c"}, false},
		{"different values", []string{"a", "b", "c"}, []string{"a", "b", "d"}, false},
		{"empty slices", []string{}, []string{}, true},
		{"one empty, one not", []string{"a"}, []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareStringSlices(tt.a, tt.b); got != tt.want {
				t.Errorf("compareStringSlices(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
