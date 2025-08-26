package rec

import (
    "context"
    "regexp"
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

func TestTopNUsesCacheWithoutProvider(t *testing.T) {
    mock, err := pgxmock.NewPool()
    if err != nil {
        t.Fatalf("failed mock: %v", err)
    }
    defer mock.Close()

    svc := NewService(mock)
    // Enable cache with TTL so service will consult quotes_cache even without provider
    svc.EnableQuoteCache(24 * time.Hour)

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

    // Expect a cache lookup for the price
    price := 100.0
    asOf := time.Now()
    mock.ExpectQuery(regexp.QuoteMeta("SELECT price, as_of FROM quotes_cache WHERE symbol = $1")).
        WithArgs("TEST").
        WillReturnRows(pgxmock.NewRows([]string{"price", "as_of"}).AddRow(price, asOf))

    recs, err := svc.TopN(context.Background(), 5)
    assert.NoError(t, err)
    assert.Len(t, recs, 1)
    assert.NotNil(t, recs[0].CurrentPrice)
    assert.NotNil(t, recs[0].PercentUpside)
    assert.InDelta(t, 0.20, *recs[0].PercentUpside, 0.05)
}
