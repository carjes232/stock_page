package rec

import (
    "context"
    "testing"
    "time"

    "github.com/pashagolub/pgxmock/v2"
    "github.com/stretchr/testify/assert"
)

type fakePriceProvider struct{}

func (fakePriceProvider) Quote(ctx context.Context, symbol string) (float64, error) {
    return 100.0, nil
}

func TestTopNWithPriceProviderAddsPercentUpside(t *testing.T) {
    mock, err := pgxmock.NewPool()
    if err != nil {
        t.Fatalf("failed mock: %v", err)
    }
    defer mock.Close()

    svc := NewService(mock)
    svc.SetPriceProvider(fakePriceProvider{})

    updatedAt := time.Now()
    targetTo := 120.0 // 20% upside at price 100
    rows := pgxmock.NewRows([]string{
        "ticker", "company", "brokerage", "rating_from", "rating_to",
        "target_from", "target_to", "price_target_delta", "last_rating_change_at", "updated_at",
    }).AddRow(
        "TEST", "Test Company", "UBS Group", "Neutral", "Buy",
        nil, &targetTo, nil, func() *time.Time { t := updatedAt.Add(-24 * time.Hour); return &t }(), updatedAt,
    )
    mock.ExpectQuery("SELECT ticker, company, brokerage, rating_from, rating_to, target_from, target_to, price_target_delta, last_rating_change_at, updated_at FROM stocks").
        WillReturnRows(rows)

    recs, err := svc.TopN(context.Background(), 5)
    assert.NoError(t, err)
    assert.Len(t, recs, 1)
    // Verify current price and percent upside are populated
    assert.NotNil(t, recs[0].CurrentPrice)
    assert.NotNil(t, recs[0].PercentUpside)
    // Percent should be ~0.20
    assert.InDelta(t, 0.20, *recs[0].PercentUpside, 0.05)
    // Score should be higher than base (upgrade only)
    base, _ := scoreOne("UBS Group", "Neutral", "Buy", nil, &targetTo, nil, nil, brokerageWeights())
    if recs[0].Score <= base {
        t.Fatalf("expected price-based bonus to increase score: got %v base %v", recs[0].Score, base)
    }
}
