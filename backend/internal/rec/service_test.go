package rec

import (
    "context"
    "math"
    "testing"
    "time"

    "github.com/pashagolub/pgxmock/v2"
    "github.com/stretchr/testify/assert"
)

func TestTransitionBonus(t *testing.T) {
    b, label := transitionBonus("Neutral", "Buy")
    if b != 1.0 || label != "upgrade" {
        t.Fatalf("expected upgrade 1.0, got %v %q", b, label)
    }
    b2, label2 := transitionBonus("Buy", "Underweight")
    if b2 >= 0 || label2 != "downgrade" {
        t.Fatalf("expected downgrade negative, got %v %q", b2, label2)
    }
}

func TestTransitionBonusEdgeCases(t *testing.T) {
    // Same rating -> reaffirm
    b, label := transitionBonus("Buy", "Buy")
    if b != 0.2 || label != "reaffirm" {
        t.Fatalf("expected reaffirm 0.2, got %v %q", b, label)
    }
    // Minor downgrade
    b, label = transitionBonus("Strong Buy", "Buy")
    if b != -1.0 || label != "downgrade" {
        t.Fatalf("expected downgrade -1.0, got %v %q", b, label)
    }
    // Unknown -> Unknown treated as neutral -> reaffirm
    b, label = transitionBonus("Unknown", "Super Buy")
    if b != 0.2 || label != "reaffirm" {
        t.Fatalf("expected reaffirm 0.2 for unknowns, got %v %q", b, label)
    }
}

func TestScoreOneWeightsAndDelta(t *testing.T) {
    weights := brokerageWeights()
    d := 10.0
    score, reasons := scoreOne("UBS Group", "Neutral", "Buy", nil, nil, &d, nil, weights)
    if len(reasons) == 0 {
        t.Fatalf("expected reasons")
    }
    // Expected base ~ tanh(10/50)*2 + 1.0 upgrade = ~0.394 + 1.0 = 1.394
    // Apply UBS weight 1.2 => ~1.6728
    want := 1.6728
    if math.Abs(score-want) > 0.2 {
        t.Fatalf("score outside tolerance: got %v want approx %v", score, want)
    }
}

func TestScoreOneRecency(t *testing.T) {
    weights := brokerageWeights()
    d := 10.0
    now := time.Now()
    score, _ := scoreOne("UBS Group", "Neutral", "Buy", nil, nil, &d, &now, weights)
    // Expected base ~ tanh(10/50)*2 + 1.0 upgrade = ~0.394 + 1.0 = 1.394
    // Recency bonus: 0.5, then weight 1.2 => ~2.2728
    want := 2.2728
    if math.Abs(score-want) > 0.2 {
        t.Fatalf("score outside tolerance: got %v want approx %v", score, want)
    }
}

func TestScoreOneEdgeCases(t *testing.T) {
    weights := brokerageWeights()
    // No delta/targets
    score, reasons := scoreOne("UBS Group", "Neutral", "Buy", nil, nil, nil, nil, weights)
    if len(reasons) == 0 {
        t.Fatalf("expected reasons")
    }
    want := 1.2 // 1.0 upgrade * 1.2 UBS
    if math.Abs(score-want) > 0.1 {
        t.Fatalf("score outside tolerance: got %v want approx %v", score, want)
    }
    // New price target only
    targetTo := 100.0
    score, reasons = scoreOne("UBS Group", "Neutral", "Buy", nil, &targetTo, nil, nil, weights)
    if len(reasons) == 0 {
        t.Fatalf("expected reasons")
    }
    want = 1.8 // (0.5 + 1.0) * 1.2
    if math.Abs(score-want) > 0.2 {
        t.Fatalf("score outside tolerance: got %v want approx %v", score, want)
    }
    // Large positive delta
    delta := 100.0
    score, _ = scoreOne("UBS Group", "Neutral", "Buy", nil, nil, &delta, nil, weights)
    want = 3.6 // approx (2 + 1) * 1.2
    if math.Abs(score-want) > 0.5 {
        t.Fatalf("score outside tolerance: got %v want approx %v", score, want)
    }
    // Large negative with downgrade
    delta = -100.0
    score, _ = scoreOne("UBS Group", "Buy", "Underweight", nil, nil, &delta, nil, weights)
    want = -3.6 // approx (-2 - 1) * 1.2
    if math.Abs(score-want) > 0.5 {
        t.Fatalf("score outside tolerance: got %v want approx %v", score, want)
    }
}

func TestNormalizeBrokerEdgeCases(t *testing.T) {
    if normalizeBroker("Goldman Sachs") != "goldman" {
        t.Fatalf("expected goldman")
    }
    if normalizeBroker("UBS Securities") != "ubs" {
        t.Fatalf("expected ubs")
    }
    if normalizeBroker("Unknown Broker") != "default" {
        t.Fatalf("expected default")
    }
}

func TestBrokerageWeights(t *testing.T) {
    w := brokerageWeights()
    if w["goldman"] != 1.3 {
        t.Fatalf("expected goldman 1.3")
    }
    if w["ubs"] != 1.2 {
        t.Fatalf("expected ubs 1.2")
    }
    if w["default"] != 1.0 {
        t.Fatalf("expected default 1.0")
    }
}

func TestTopN(t *testing.T) {
    mock, err := pgxmock.NewPool()
    if err != nil {
        t.Fatalf("failed to create mock pool: %v", err)
    }
    defer mock.Close()
    svc := NewService(mock)
    updatedAt := time.Now()
    delta1 := 10.0
    delta2 := 15.0
    rows := pgxmock.NewRows([]string{
        "ticker", "company", "brokerage", "rating_from", "rating_to",
        "target_from", "target_to", "price_target_delta", "last_rating_change_at", "updated_at",
    }).AddRow(
        "TEST", "Test Company", "UBS Group", "Neutral", "Buy",
        nil, nil, &delta1, nil, updatedAt,
    ).AddRow(
        "TEST2", "Test Company 2", "Goldman Sachs", "Buy", "Strong Buy",
        nil, nil, &delta2, nil, updatedAt.Add(-time.Hour),
    )
    mock.ExpectQuery("SELECT ticker, company, brokerage, rating_from, rating_to, target_from, target_to, price_target_delta, last_rating_change_at, updated_at FROM stocks").
        WillReturnRows(rows)
    recs, err := svc.TopN(context.Background(), 5)
    assert.NoError(t, err)
    assert.Len(t, recs, 2)
}

func TestTopNWithInvalidN(t *testing.T) {
    mock, _ := pgxmock.NewPool()
    defer mock.Close()
    svc := NewService(mock)
    updatedAt := time.Now()
    delta := 10.0
    rows := pgxmock.NewRows([]string{
        "ticker", "company", "brokerage", "rating_from", "rating_to",
        "target_from", "target_to", "price_target_delta", "last_rating_change_at", "updated_at",
    }).AddRow(
        "TEST", "Test Company", "UBS Group", "Neutral", "Buy",
        nil, nil, &delta, nil, updatedAt,
    )
    mock.ExpectQuery("SELECT ticker, company, brokerage, rating_from, rating_to, target_from, target_to, price_target_delta, last_rating_change_at, updated_at FROM stocks").
        WillReturnRows(rows)
    recs, err := svc.TopN(context.Background(), -1)
    assert.NoError(t, err)
    assert.Len(t, recs, 1)
}

func TestTopNWithNTooLarge(t *testing.T) {
    mock, _ := pgxmock.NewPool()
    defer mock.Close()
    svc := NewService(mock)
    updatedAt := time.Now()
    delta := 10.0
    rows := pgxmock.NewRows([]string{
        "ticker", "company", "brokerage", "rating_from", "rating_to",
        "target_from", "target_to", "price_target_delta", "last_rating_change_at", "updated_at",
    }).AddRow(
        "TEST", "Test Company", "UBS Group", "Neutral", "Buy",
        nil, nil, &delta, nil, updatedAt,
    )
    mock.ExpectQuery("SELECT ticker, company, brokerage, rating_from, rating_to, target_from, target_to, price_target_delta, last_rating_change_at, updated_at FROM stocks").
        WillReturnRows(rows)
    recs, err := svc.TopN(context.Background(), 100)
    assert.NoError(t, err)
    assert.Len(t, recs, 1)
}

func TestTopNWithDatabaseError(t *testing.T) {
    mock, _ := pgxmock.NewPool()
    defer mock.Close()
    svc := NewService(mock)
    mock.ExpectQuery("SELECT ticker, company, brokerage, rating_from, rating_to, target_from, target_to, price_target_delta, last_rating_change_at, updated_at FROM stocks").
        WillReturnError(assert.AnError)
    recs, err := svc.TopN(context.Background(), 5)
    assert.Error(t, err)
    assert.Nil(t, recs)
}

func TestTopNWithEmptyResults(t *testing.T) {
    mock, _ := pgxmock.NewPool()
    defer mock.Close()
    svc := NewService(mock)
    rows := pgxmock.NewRows([]string{
        "ticker", "company", "brokerage", "rating_from", "rating_to",
        "target_from", "target_to", "price_target_delta", "last_rating_change_at", "updated_at",
    })
    mock.ExpectQuery("SELECT ticker, company, brokerage, rating_from, rating_to, target_from, target_to, price_target_delta, last_rating_change_at, updated_at FROM stocks").
        WillReturnRows(rows)
    recs, err := svc.TopN(context.Background(), 5)
    assert.NoError(t, err)
    assert.Len(t, recs, 0)
}

func TestTopNWithScoringError(t *testing.T) {
    mock, _ := pgxmock.NewPool()
    defer mock.Close()
    svc := NewService(mock)
    updatedAt := time.Now()
    rows := pgxmock.NewRows([]string{
        "ticker", "company", "brokerage", "rating_from", "rating_to",
        "target_from", "target_to", "price_target_delta", "last_rating_change_at", "updated_at",
    }).AddRow(
        "TEST", "Test Company", "UBS Group", "Neutral", "Buy",
        "invalid", "invalid", "invalid", "invalid", updatedAt,
    )
    mock.ExpectQuery("SELECT ticker, company, brokerage, rating_from, rating_to, target_from, target_to, price_target_delta, last_rating_change_at, updated_at FROM stocks").
        WillReturnRows(rows)
    recs, err := svc.TopN(context.Background(), 5)
    assert.NoError(t, err)
    assert.Len(t, recs, 0)
}
