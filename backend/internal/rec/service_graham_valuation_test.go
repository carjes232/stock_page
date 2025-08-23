package rec

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGrahamValuationProvider is a mock type for the GrahamValuationProvider interface
type MockGrahamValuationProvider struct {
	mock.Mock
}

// GetGrahamValuation is a mock method for GetGrahamValuation
func (m *MockGrahamValuationProvider) GetGrahamValuation(ctx context.Context, symbol string) (float64, float64, error) {
	args := m.Called(ctx, symbol)
	return args.Get(0).(float64), args.Get(1).(float64), args.Error(2)
}

// MockCorporateBondYieldProvider is a mock type for the CorporateBondYieldProvider interface
type MockCorporateBondYieldProvider struct {
	mock.Mock
}

// GetAAACorporateBondYield is a mock method for GetAAACorporateBondYield
func (m *MockCorporateBondYieldProvider) GetAAACorporateBondYield(ctx context.Context) (float64, error) {
	args := m.Called(ctx)
	return args.Get(0).(float64), args.Error(1)
}

func TestGetGrahamValuation_Cache(t *testing.T) {
	mockPool, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mockPool.Close()

	svc := NewService(mockPool)
	mockGrahamProvider := new(MockGrahamValuationProvider)
	mockBondProvider := new(MockCorporateBondYieldProvider)
	svc.SetGrahamValuationProvider(mockGrahamProvider)
	svc.SetCorporateBondYieldProvider(mockBondProvider)
	svc.EnableGrahamValuation(1 * time.Second)

	ticker := "AAPL"
	expectedEPS := 6.45
	expectedGrowth := 0.08
	expectedBondYield := 4.5

	// 1. First call, cache miss, provider is called
	mockPool.ExpectQuery(`SELECT eps_avg, growth_estimate, updated_at FROM fundamentals WHERE ticker = \$1`).
		WithArgs(ticker).
		WillReturnError(assert.AnError) // Simulate cache miss
	mockGrahamProvider.On("GetGrahamValuation", mock.Anything, ticker).Return(expectedEPS, expectedGrowth, nil).Once()
	mockBondProvider.On("GetAAACorporateBondYield", mock.Anything).Return(expectedBondYield, nil).Once()
	mockPool.ExpectExec(`UPSERT INTO fundamentals`).
		WithArgs(ticker, expectedEPS, expectedGrowth).
		WillReturnResult(pgxmock.NewResult("UPSERT", 1))

	eps, growth, ok := svc.getGrahamValuation(context.Background(), ticker)
	assert.True(t, ok)
	assert.Equal(t, expectedEPS, eps)
	assert.Equal(t, expectedGrowth, growth)
	mockGrahamProvider.AssertExpectations(t)

	// 2. Second call, cache hit, provider is not called
	mockPool.ExpectQuery(`SELECT eps_avg, growth_estimate, updated_at FROM fundamentals WHERE ticker = \$1`).
		WithArgs(ticker).
		WillReturnRows(pgxmock.NewRows([]string{"eps_avg", "growth_estimate", "updated_at"}).AddRow(expectedEPS, expectedGrowth, time.Now()))

	eps, growth, ok = svc.getGrahamValuation(context.Background(), ticker)
	assert.True(t, ok)
	assert.Equal(t, expectedEPS, eps)
	assert.Equal(t, expectedGrowth, growth)
	mockGrahamProvider.AssertExpectations(t) // No new calls to provider

	// 3. Wait for TTL to expire
	time.Sleep(1 * time.Second)

	// 4. Third call, cache stale, provider is called again
	mockPool.ExpectQuery(`SELECT eps_avg, growth_estimate, updated_at FROM fundamentals WHERE ticker = \$1`).
		WithArgs(ticker).
		WillReturnRows(pgxmock.NewRows([]string{"eps_avg", "growth_estimate", "updated_at"}).AddRow(expectedEPS, expectedGrowth, time.Now().Add(-2*time.Second)))
	mockGrahamProvider.On("GetGrahamValuation", mock.Anything, ticker).Return(expectedEPS, expectedGrowth, nil).Once()
	mockPool.ExpectExec(`UPSERT INTO fundamentals`).
		WithArgs(ticker, expectedEPS, expectedGrowth).
		WillReturnResult(pgxmock.NewResult("UPSERT", 1))

	eps, growth, ok = svc.getGrahamValuation(context.Background(), ticker)
	assert.True(t, ok)
	assert.Equal(t, expectedEPS, eps)
	assert.Equal(t, expectedGrowth, growth)
	mockGrahamProvider.AssertExpectations(t)
}