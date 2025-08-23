package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestParseDollars(t *testing.T) {
	cases := []struct {
		in   *string
		want *float64
	}{
		{nil, nil},
		{ptr(""), nil},
		{ptr("$8.00"), ptrf(8)},
		{ptr("$33,000.50"), ptrf(33000.5)},
		{ptr("3.5"), ptrf(3.5)},
		{ptr("abc"), nil},
	}
	for i, c := range cases {
		got := parseDollars(c.in)
		if (got == nil) != (c.want == nil) {
			t.Fatalf("case %d: got nil=%v want nil=%v", i, got == nil, c.want == nil)
		}
		if got != nil && c.want != nil {
			if *got != *c.want {
				t.Fatalf("case %d: got %v want %v", i, *got, *c.want)
			}
		}
	}
}

func TestParseTime(t *testing.T) {
	// Valid RFC3339 time string
	validTimeStr := "2023-01-01T12:00:00Z"
	validTime, _ := time.Parse(time.RFC3339, validTimeStr)

	cases := []struct {
		in   *string
		want *time.Time
	}{
		{nil, nil},
		{ptr(""), nil},
		{ptr(validTimeStr), &validTime},
		// Add more test cases as needed
	}
	for i, c := range cases {
		got := parseTime(c.in)
		if (got == nil) != (c.want == nil) {
			t.Fatalf("case %d: got nil=%v want nil=%v", i, got == nil, c.want == nil)
		}
		if got != nil && c.want != nil {
			if *got != *c.want {
				t.Fatalf("case %d: got %v want %v", i, *got, *c.want)
			}
		}
	}
}

func TestRunOnceWithMissingConfig(t *testing.T) {
	// Create a mock database connection
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	// Create a logger
	logger, _ := zap.NewDevelopment()
	log := logger.Sugar()

	// Create the service with empty API base and token
	service := NewService("", "", mock, log)

	// Call RunOnce - should return an error
	err = service.RunOnce(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing API base or token")
}

func TestFetchPage(t *testing.T) {
	// Create a test server that returns mock data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		assert.Equal(t, "/api/data", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Return mock data
		response := apiResponse{
			Items: []apiItem{
				{
					Ticker:     "TEST",
					Company:    "Test Company",
					Brokerage:  "Test Brokerage",
					Action:     "Buy",
					RatingFrom: "Neutral",
					RatingTo:   "Buy",
					TargetFrom: ptr("$100.00"),
					TargetTo:   ptr("$120.00"),
					Time:       ptr("2023-01-01T12:00:00Z"),
				},
			},
			NextPage: "",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create a mock database connection
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	// Create a logger
	logger, _ := zap.NewDevelopment()
	log := logger.Sugar()

	// Create the service with the test server URL
	service := NewService(server.URL+"/api/data", "test-token", mock, log)

	// Call fetchPage
	items, nextPage, err := service.fetchPage(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, "", nextPage)
	assert.Len(t, items, 1)
	assert.Equal(t, "TEST", items[0].Ticker)
	assert.Equal(t, "Test Company", items[0].Company)
}

func TestFetchPageWithNextPage(t *testing.T) {
	// Create a test server that returns mock data with a next page
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		assert.Equal(t, "/api/data", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Check if next_page parameter is present
		nextPage := r.URL.Query().Get("next_page")
		assert.Equal(t, "page2", nextPage)

		// Return mock data
		response := apiResponse{
			Items: []apiItem{
				{
					Ticker:     "TEST2",
					Company:    "Test Company 2",
					Brokerage:  "Test Brokerage 2",
					Action:     "Sell",
					RatingFrom: "Buy",
					RatingTo:   "Underweight",
					TargetFrom: ptr("$120.00"),
					TargetTo:   ptr("$100.00"),
					Time:       ptr("2023-01-02T12:00:00Z"),
				},
			},
			NextPage: "",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create a mock database connection
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	// Create a logger
	logger, _ := zap.NewDevelopment()
	log := logger.Sugar()

	// Create the service with the test server URL
	service := NewService(server.URL+"/api/data", "test-token", mock, log)

	// Call fetchPage with a next page parameter
	items, nextPage, err := service.fetchPage(context.Background(), "page2")
	assert.NoError(t, err)
	assert.Equal(t, "", nextPage)
	assert.Len(t, items, 1)
	assert.Equal(t, "TEST2", items[0].Ticker)
	assert.Equal(t, "Test Company 2", items[0].Company)
}

func TestFetchPageWithHTTPError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create a mock database connection
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	// Create a logger
	logger, _ := zap.NewDevelopment()
	log := logger.Sugar()

	// Create the service with the test server URL
	service := NewService(server.URL+"/api/data", "test-token", mock, log)

	// Call fetchPage - should return an error
	items, nextPage, err := service.fetchPage(context.Background(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "api error: status=500")
	assert.Equal(t, "", nextPage)
	assert.Len(t, items, 0)
}

func TestUpsertItems(t *testing.T) {
	// Create a mock database connection
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	// Create a logger
	logger, _ := zap.NewDevelopment()
	log := logger.Sugar()

	// Create the service with the mock database
	service := NewService("http://example.com", "test-token", mock, log)

	// Set up the mock to expect the transaction and queries
	mock.ExpectBegin()

	// Parse the time string from the test data
	lastRatingChangeStr := "2023-01-01T12:00:00Z"
	lastRatingChange, _ := time.Parse(time.RFC3339, lastRatingChangeStr)

	// Create pointers to the float values
	targetFrom := 100.0
	targetTo := 120.0

	mock.ExpectExec(`INSERT INTO stocks`).WithArgs(
		"TEST", "Test Company", "Test Brokerage", "Buy", "Neutral", "Buy",
		&targetFrom, &targetTo, // parsed target values (pointers)
		&lastRatingChange, // parsed last_rating_change_at
		pgxmock.AnyArg(),  // created_at (time.Now().UTC())
		pgxmock.AnyArg(),  // updated_at (time.Now().UTC())
	).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	// Create test items
	items := []apiItem{
		{
			Ticker:     "TEST",
			Company:    "Test Company",
			Brokerage:  "Test Brokerage",
			Action:     "Buy",
			RatingFrom: "Neutral",
			RatingTo:   "Buy",
			TargetFrom: ptr("$100.00"),
			TargetTo:   ptr("$120.00"),
			Time:       ptr("2023-01-01T12:00:00Z"),
		},
	}

	// Call upsertItems
	err = service.upsertItems(context.Background(), items)
	assert.NoError(t, err)

	// Verify that the mock expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunOnce(t *testing.T) {
	// Create a test server that returns mock data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return mock data
		response := apiResponse{
			Items: []apiItem{
				{
					Ticker:     "TEST",
					Company:    "Test Company",
					Brokerage:  "Test Brokerage",
					Action:     "Buy",
					RatingFrom: "Neutral",
					RatingTo:   "Buy",
					TargetFrom: ptr("$100.00"),
					TargetTo:   ptr("$120.00"),
					Time:       ptr("2023-01-01T12:00:00Z"),
				},
			},
			NextPage: "",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create a mock database connection
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	// Create a logger
	logger, _ := zap.NewDevelopment()
	log := logger.Sugar()

	// Create the service with the test server URL
	service := NewService(server.URL, "test-token", mock, log)

	// Set up the mock to expect the transaction and queries
	mock.ExpectBegin()

	// Parse the time string from the test data
	lastRatingChangeStr := "2023-01-01T12:00:00Z"
	lastRatingChange, _ := time.Parse(time.RFC3339, lastRatingChangeStr)

	// Create pointers to the float values
	targetFrom := 100.0
	targetTo := 120.0

	mock.ExpectExec(`INSERT INTO stocks`).WithArgs(
		"TEST", "Test Company", "Test Brokerage", "Buy", "Neutral", "Buy",
		&targetFrom, &targetTo, // parsed target values (pointers)
		&lastRatingChange, // parsed last_rating_change_at
		pgxmock.AnyArg(),  // created_at (time.Now().UTC())
		pgxmock.AnyArg(),  // updated_at (time.Now().UTC())
	).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	// Call RunOnce
	err = service.RunOnce(context.Background())
	assert.NoError(t, err)

	// Verify that the mock expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func ptr(s string) *string { return &s }
func TestParseDollarsEdgeCases(t *testing.T) {
	cases := []struct {
		in   *string
		want *float64
	}{
		{ptr("$"), nil},
		{ptr("$,"), nil},
		{ptr("$-100.50"), ptrf(-100.5)},
		{ptr(" $ 50.25 "), ptrf(50.25)},
		{ptr("$1,234.56"), ptrf(1234.56)},
	}
	for i, c := range cases {
		got := parseDollars(c.in)
		if (got == nil) != (c.want == nil) {
			t.Fatalf("case %d: got nil=%v want nil=%v", i, got == nil, c.want == nil)
		}
		if got != nil && c.want != nil {
			if *got != *c.want {
				t.Fatalf("case %d: got %v want %v", i, *got, *c.want)
			}
		}
	}
}

func TestParseTimeEdgeCases(t *testing.T) {
	// Valid RFC3339 time string
	validTimeStr := "2023-01-01T12:00:00Z"
	validTime, _ := time.Parse(time.RFC3339, validTimeStr)

	cases := []struct {
		in   *string
		want *time.Time
	}{
		{nil, nil},
		{ptr(""), nil},
		{ptr(validTimeStr), &validTime},
		{ptr("invalid-time"), nil},
		{ptr("2023-01-01T12:00:00"), nil},  // Missing Z suffix
		{ptr("2023-01-01 12:00:00Z"), nil}, // Wrong format
	}
	for i, c := range cases {
		got := parseTime(c.in)
		if (got == nil) != (c.want == nil) {
			t.Fatalf("case %d: got nil=%v want nil=%v", i, got == nil, c.want == nil)
		}
		if got != nil && c.want != nil {
			if *got != *c.want {
				t.Fatalf("case %d: got %v want %v", i, *got, *c.want)
			}
		}
	}
}

func TestFetchPageWithInvalidURL(t *testing.T) {
	// Create a mock database connection
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	// Create a logger
	logger, _ := zap.NewDevelopment()
	log := logger.Sugar()

	// Create the service with an invalid URL
	service := NewService("not-a-valid-url:", "test-token", mock, log)

	// Call fetchPage - should return an error
	items, nextPage, err := service.fetchPage(context.Background(), "")
	assert.Error(t, err)
	assert.Equal(t, "", nextPage)
	assert.Len(t, items, 0)
}

func TestFetchPageWithDecodeError(t *testing.T) {
	// Create a test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return invalid JSON
		w.Write([]byte("{ invalid json }"))
	}))
	defer server.Close()

	// Create a mock database connection
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	// Create a logger
	logger, _ := zap.NewDevelopment()
	log := logger.Sugar()

	// Create the service with the test server URL
	service := NewService(server.URL+"/api/data", "test-token", mock, log)

	// Call fetchPage - should return no error but empty results
	items, nextPage, err := service.fetchPage(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, "", nextPage)
	assert.Len(t, items, 0)
}

func TestUpsertItemsWithDatabaseError(t *testing.T) {
	// Create a mock database connection
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	// Create a logger
	logger, _ := zap.NewDevelopment()
	log := logger.Sugar()

	// Create the service with the mock database
	service := NewService("http://example.com", "test-token", mock, log)

	// Set up the mock to expect the transaction and return an error
	mock.ExpectBegin().WillReturnError(fmt.Errorf("database error"))

	// Create test items
	items := []apiItem{
		{
			Ticker:     "TEST",
			Company:    "Test Company",
			Brokerage:  "Test Brokerage",
			Action:     "Buy",
			RatingFrom: "Neutral",
			RatingTo:   "Buy",
			TargetFrom: ptr("$100.0"),
			TargetTo:   ptr("$120.0"),
			Time:       ptr("2023-01-01T12:00:00Z"),
		},
	}

	// Call upsertItems - should return an error
	err = service.upsertItems(context.Background(), items)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")

	// Verify that the mock expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunOnceWithMultiplePages(t *testing.T) {
	// Create a test server that returns mock data with multiple pages
	var callCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		var response apiResponse
		if callCount == 1 {
			// First page
			response = apiResponse{
				Items: []apiItem{
					{
						Ticker:     "TEST1",
						Company:    "Test Company 1",
						Brokerage:  "Test Brokerage 1",
						Action:     "Buy",
						RatingFrom: "Neutral",
						RatingTo:   "Buy",
						TargetFrom: ptr("$100.00"),
						TargetTo:   ptr("$120.0"),
						Time:       ptr("2023-01-01T12:00:00Z"),
					},
				},
				NextPage: "page2",
			}
		} else {
			// Second page
			response = apiResponse{
				Items: []apiItem{
					{
						Ticker:     "TEST2",
						Company:    "Test Company 2",
						Brokerage:  "Test Brokerage 2",
						Action:     "Sell",
						RatingFrom: "Buy",
						RatingTo:   "Underweight",
						TargetFrom: ptr("$120.00"),
						TargetTo:   ptr("$100.00"),
						Time:       ptr("2023-01-02T12:00Z"),
					},
				},
				NextPage: "",
			}
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create a mock database connection
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	// Create a logger
	logger, _ := zap.NewDevelopment()
	log := logger.Sugar()

	// Create the service with the test server URL
	service := NewService(server.URL, "test-token", mock, log)

	// Set up the mock to expect the transactions and queries
	mock.ExpectBegin()

	// Parse the time strings from the test data
	lastRatingChangeStr1 := "2023-01-01T12:00:00Z"
	lastRatingChange1, _ := time.Parse(time.RFC3339, lastRatingChangeStr1)
	targetFrom1 := 100.0
	targetTo1 := 120.0

	mock.ExpectExec(`INSERT INTO stocks`).WithArgs(
		"TEST1", "Test Company 1", "Test Brokerage 1", "Buy", "Neutral", "Buy",
		&targetFrom1, &targetTo1,
		&lastRatingChange1,
		pgxmock.AnyArg(),
		pgxmock.AnyArg(),
	).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	mock.ExpectBegin()

	lastRatingChangeStr2 := "2023-01-02T12:00:00Z"
	lastRatingChange2, _ := time.Parse(time.RFC3339, lastRatingChangeStr2)
	targetFrom2 := 120.0
	targetTo2 := 100.0

	mock.ExpectExec(`INSERT INTO stocks`).WithArgs(
		"TEST2", "Test Company 2", "Test Brokerage 2", "Sell", "Buy", "Underweight",
		&targetFrom2, &targetTo2,
		&lastRatingChange2,
		pgxmock.AnyArg(),
		pgxmock.AnyArg(),
	).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()

	// Call RunOnce
	err = service.RunOnce(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 2, callCount)

	// Verify that the mock expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunOnceWithDatabaseError(t *testing.T) {
	// Create a test server that returns mock data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return mock data
		response := apiResponse{
			Items: []apiItem{
				{
					Ticker:     "TEST",
					Company:    "Test Company",
					Brokerage:  "Test Brokerage",
					Action:     "Buy",
					RatingFrom: "Neutral",
					RatingTo:   "Buy",
					TargetFrom: ptr("$100.00"),
					TargetTo:   ptr("$120.00"),
					Time:       ptr("2023-01-01T12:00:00Z"),
				},
			},
			NextPage: "",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create a mock database connection
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	// Create a logger
	logger, _ := zap.NewDevelopment()
	log := logger.Sugar()

	// Create the service with the test server URL
	service := NewService(server.URL, "test-token", mock, log)

	// Set up the mock to expect the transaction and return an error
	mock.ExpectBegin().WillReturnError(fmt.Errorf("database error"))

	// Call RunOnce - should return an error
	err = service.RunOnce(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")

	// Verify that the mock expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}
func ptrf(f float64) *float64 { return &f }
