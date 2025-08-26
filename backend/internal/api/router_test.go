package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"stockchallenge/backend/internal/ingest"
	"stockchallenge/backend/internal/portfolio"
	"stockchallenge/backend/internal/rec"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pashagolub/pgxmock/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockClock struct {
	now time.Time
}

func (m *mockClock) Now() time.Time {
	return m.now
}

// Mock portfolio service for testing
type mockPortfolioService struct{}

func (m *mockPortfolioService) ExtractAndSavePortfolio(ctx context.Context, userID string, imageData []byte) (*portfolio.PositionsOut, error) {
	return &portfolio.PositionsOut{}, nil
}

func setupMockRouter(t *testing.T) (*gin.Engine, pgxmock.PgxPoolIface) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}

	logger, _ := zap.NewDevelopment()
	log := logger.Sugar()

	ingestSvc := ingest.NewService("", "", mock, log)
	recSvc := rec.NewService(mock)

	// Create a mock portfolio service
	portSvc := &mockPortfolioService{}

	router := NewRouter(mock, ingestSvc, recSvc, portSvc, log, "")

	return router.(*gin.Engine), mock
}

func TestHealthz(t *testing.T) {
	router, _ := setupMockRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp["ok"].(bool))
}

func TestListStocks(t *testing.T) {
	router, mock := setupMockRouter(t)
	defer mock.Close()

	p100 := 100.0
	p120 := 120.0
	pd := 0.2
	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "ticker", "company", "brokerage", "action", "rating_from", "rating_to", "target_from", "target_to", "last_rating_change_at", "price_target_delta", "created_at", "updated_at"}).
		AddRow("1", "TEST", "Test Company", "Test Brokerage", "Buy", "Neutral", "Buy", &p100, &p120, &now, &pd, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT id, ticker, company, brokerage, action, rating_from, rating_to, target_from, target_to, last_rating_change_at, price_target_delta, created_at, updated_at FROM stocks ORDER BY updated_at DESC LIMIT`).
		WithArgs(20, 0).
		WillReturnRows(rows)
	mock.ExpectQuery(`SELECT count`).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/stocks", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetStock(t *testing.T) {
	router, mock := setupMockRouter(t)
	defer mock.Close()

	p100 := 100.0
	p120 := 120.0
	pd := 0.2
	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "company", "brokerage", "action", "rating_from", "rating_to", "target_from", "target_to", "last_rating_change_at", "price_target_delta", "created_at", "updated_at"}).
		AddRow("1", "Test Company", "Test Brokerage", "Buy", "Neutral", "Buy", &p100, &p120, &now, &pd, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT id, company, brokerage, action, rating_from, rating_to, target_from, target_to, last_rating_change_at, price_target_delta, created_at, updated_at FROM stocks WHERE ticker`).
		WithArgs("TEST").
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/stocks/TEST", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSearchStocks(t *testing.T) {
	router, mock := setupMockRouter(t)
	defer mock.Close()

	p100 := 100.0
	p120 := 120.0
	pd := 0.2
	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "ticker", "company", "brokerage", "action", "rating_from", "rating_to", "target_from", "target_to", "last_rating_change_at", "price_target_delta", "created_at", "updated_at"}).
		AddRow("1", "TEST", "Test Company", "Test Brokerage", "Buy", "Neutral", "Buy", &p100, &p120, &now, &pd, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT id, ticker, company, brokerage, action, rating_from, rating_to, target_from, target_to, last_rating_change_at, price_target_delta, created_at, updated_at FROM stocks WHERE ticker ILIKE`).
		WithArgs("TEST", 20, 0).
		WillReturnRows(rows)
	mock.ExpectQuery(`SELECT count`).WithArgs("TEST").WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/stocks/search?q=TEST", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSortStocks(t *testing.T) {
	router, mock := setupMockRouter(t)
	defer mock.Close()

	p100 := 100.0
	p120 := 120.0
	pd := 0.2
	now := time.Now()
	rows := pgxmock.NewRows([]string{"id", "ticker", "company", "brokerage", "action", "rating_from", "rating_to", "target_from", "target_to", "last_rating_change_at", "price_target_delta", "created_at", "updated_at"}).
		AddRow("1", "TEST", "Test Company", "Test Brokerage", "Buy", "Neutral", "Buy", &p100, &p120, &now, &pd, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT id, ticker, company, brokerage, action, rating_from, rating_to, target_from, target_to, last_rating_change_at, price_target_delta, created_at, updated_at FROM stocks ORDER BY ticker ASC LIMIT`).
		WithArgs(20, 0).
		WillReturnRows(rows)
	mock.ExpectQuery(`SELECT count`).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/stocks/sort?field=ticker&order=ASC", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetRecommendations(t *testing.T) {
	router, mock := setupMockRouter(t)
	defer mock.Close()

	p100 := 100.0
	p120 := 120.0
	pd := 0.2
	now := time.Now()
	rows := pgxmock.NewRows([]string{"ticker", "company", "brokerage", "rating_from", "rating_to", "target_from", "target_to", "price_target_delta", "last_rating_change_at", "updated_at"}).
		AddRow("TEST", "Test Company", "Test Brokerage", "Neutral", "Buy", &p100, &p120, &pd, &now, time.Now())

	mock.ExpectQuery(`SELECT ticker, company, brokerage, rating_from, rating_to, target_from, target_to, price_target_delta, last_rating_change_at, updated_at FROM stocks ORDER BY updated_at DESC LIMIT`).
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/recommendations", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string][]rec.Recommendation
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp["items"], 1)
	assert.Equal(t, "TEST", resp["items"][0].Ticker)
}

func TestRunIngest(t *testing.T) {
	router, mock := setupMockRouter(t)
	defer mock.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/admin/ingest", bytes.NewBuffer([]byte{}))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
}
