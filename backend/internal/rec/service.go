package rec

import (
	"context"
	"math"
	"strings"
	"time"

	"stockchallenge/backend/internal/db"
)

type Service struct {
	db                         db.DBTX
	prices                     PriceProvider
	useCache                   bool
	quotesTTL                  time.Duration
	topK                       int
	grahamValuationProvider    GrahamValuationProvider
	grahamValuationTTL         time.Duration
	corporateBondYieldProvider CorporateBondYieldProvider
}

func NewService(db db.DBTX) *Service {
	return &Service{db: db, topK: 20}
}

// PriceProvider is a narrow interface for fetching current prices.
// Implemented by internal/marketdata providers.
type PriceProvider interface {
	Quote(ctx context.Context, symbol string) (float64, error)
}

// CorporateBondYieldProvider defines the minimal interface for fetching the AAA corporate bond yield.
type CorporateBondYieldProvider interface {
	GetAAACorporateBondYield(ctx context.Context) (float64, error)
}

type GrahamValuationProvider interface {
	GetGrahamValuation(ctx context.Context, symbol string) (eps float64, growthRate float64, err error)
}

// SetPriceProvider wires an optional price provider.
func (s *Service) SetPriceProvider(p PriceProvider) { s.prices = p }

// EnableQuoteCache toggles DB-backed quote cache with TTL.
func (s *Service) EnableQuoteCache(ttl time.Duration) {
	if ttl <= 0 {
		s.useCache = false
		s.quotesTTL = 0
		return
	}
	s.useCache = true
	s.quotesTTL = ttl
}

// SetTopK configures how many top-ranked items to enrich with prices before final sort.
func (s *Service) SetTopK(k int) {
	if k < 1 {
		k = 20
	}
	s.topK = k
}

// SetGrahamValuationProvider wires an optional Graham valuation provider.
func (s *Service) SetGrahamValuationProvider(p GrahamValuationProvider) {
	s.grahamValuationProvider = p
}

// EnableGrahamValuation configures TTL for Graham valuation cache refreshes.
func (s *Service) EnableGrahamValuation(ttl time.Duration) {
	if ttl <= 0 {
		s.grahamValuationTTL = 0
		return
	}
	s.grahamValuationTTL = ttl
}

// SetCorporateBondYieldProvider wires an optional corporate bond yield provider.
func (s *Service) SetCorporateBondYieldProvider(p CorporateBondYieldProvider) {
	s.corporateBondYieldProvider = p
}

type Recommendation struct {
	Ticker          string     `json:"ticker"`
	Company         string     `json:"company"`
	Brokerage       string     `json:"brokerage"`
	RatingFrom      string     `json:"rating_from"`
	RatingTo        string     `json:"rating_to"`
	TargetFrom      *float64   `json:"target_from,omitempty"`
	TargetTo        *float64   `json:"target_to,omitempty"`
	PriceDelta      *float64   `json:"price_target_delta,omitempty"`
	CurrentPrice    *float64   `json:"current_price,omitempty"`
	PercentUpside   *float64   `json:"percent_upside,omitempty"`
	EPS             *float64   `json:"eps,omitempty"`
	Growth          *float64   `json:"growth,omitempty"`
	Intrinsic       *float64   `json:"intrinsic_value,omitempty"`
	IntrinsicValue2 *float64   `json:"intrinsic_value_2,omitempty"`
	Score           float64    `json:"score"`
	ScoreReasons    []string   `json:"reasons"`
	LastChange      *time.Time `json:"last_rating_change_at,omitempty"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (s *Service) TopN(ctx context.Context, n int) ([]Recommendation, error) {
	if n <= 0 || n > 50 {
		n = 5
	}
	rows, err := s.db.Query(ctx, `
SELECT ticker, company, brokerage, rating_from, rating_to, target_from, target_to, price_target_delta, last_rating_change_at, updated_at
FROM stocks
ORDER BY updated_at DESC
LIMIT 500
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	weights := brokerageWeights()

	recs := make([]Recommendation, 0, 50)
	tickers := make([]string, 0, 100)
	seen := make(map[string]struct{})
	for rows.Next() {
		var (
			ticker, company, brokerage, ratingFrom, ratingTo string
			targetFrom, targetTo, delta                      *float64
			lastChange                                       *time.Time
			updatedAt                                        time.Time
		)
		if err := rows.Scan(&ticker, &company, &brokerage, &ratingFrom, &ratingTo, &targetFrom, &targetTo, &delta, &lastChange, &updatedAt); err != nil {
			continue
		}
		score, reasons := scoreOne(brokerage, ratingFrom, ratingTo, targetFrom, targetTo, delta, lastChange, weights)
		recs = append(recs, Recommendation{
			Ticker:       ticker,
			Company:      company,
			Brokerage:    brokerage,
			RatingFrom:   ratingFrom,
			RatingTo:     ratingTo,
			TargetFrom:   targetFrom,
			TargetTo:     targetTo,
			PriceDelta:   delta,
			Score:        score,
			ScoreReasons: reasons,
			LastChange:   lastChange,
			UpdatedAt:    updatedAt,
		})
		if _, ok := seen[ticker]; !ok {
			seen[ticker] = struct{}{}
			tickers = append(tickers, ticker)
		}
	}

	// Phase 1: sort by base score, tie-break by updated_at desc
	sortSliceStable(recs, func(a, b Recommendation) bool {
		if a.Score == b.Score {
			return a.UpdatedAt.After(b.UpdatedAt)
		}
		return a.Score > b.Score
	})

	// Phase 2: price-aware enrichment for top-K only, then resort by new score.
	if s.prices != nil && len(recs) > 0 {
		k := s.topK
		if k > len(recs) {
			k = len(recs)
		}
		enrichSet := make(map[string]struct{}, k)
		for i := 0; i < k; i++ {
			enrichSet[recs[i].Ticker] = struct{}{}
		}
		priceByTicker := make(map[string]float64, k)
		for t := range enrichSet {
			if p, ok := s.getQuote(ctx, t); ok && p > 0 {
				priceByTicker[t] = p
			}
		}
		for i := 0; i < k; i++ {
			if p, ok := priceByTicker[recs[i].Ticker]; ok {
				cp := p
				recs[i].CurrentPrice = &cp
				if recs[i].TargetTo != nil && *recs[i].TargetTo > 0 {
					up := (*recs[i].TargetTo / cp) - 1.0
					recs[i].PercentUpside = &up
					bonus := math.Tanh(up*2.0) * 2.0
					recs[i].Score += bonus
					recs[i].ScoreReasons = append(recs[i].ScoreReasons, "relative upside vs price")
				}
			}
		}
		// Resort by score after enrichment
		sortSliceStable(recs, func(a, b Recommendation) bool {
			if a.Score == b.Score {
				return a.UpdatedAt.After(b.UpdatedAt)
			}
			return a.Score > b.Score
		})
	}

	// Phase 3: fundamentals enrichment (EPS and basic intrinsic value) for top-K
	if len(recs) > 0 {
		k := s.topK
		if k > len(recs) {
			k = len(recs)
		}
		for i := 0; i < k; i++ {
			if eps, growth, ok := s.getGrahamValuation(ctx, recs[i].Ticker); ok {
				e := eps
				g := growth
				recs[i].EPS = &e
				recs[i].Growth = &g
				// Graham formula: V = EPS * (8.5 + 2g)
				iv := e * (8.5 + 2*g*100)
				recs[i].Intrinsic = &iv

				if s.corporateBondYieldProvider != nil {
					if bondYield, err := s.corporateBondYieldProvider.GetAAACorporateBondYield(ctx); err == nil && bondYield > 0 {
						iv2 := iv * 4.4 / bondYield
						recs[i].IntrinsicValue2 = &iv2
					}
				}
			}
		}
	}

	if len(recs) > n {
		recs = recs[:n]
	}
	return recs, nil
}

// getQuote returns a price from cache if fresh, otherwise calls provider and upserts cache when enabled.
func (s *Service) getQuote(ctx context.Context, symbol string) (float64, bool) {
	if s.prices == nil {
		return 0, false
	}
	if s.useCache && s.quotesTTL > 0 {
		var price float64
		var asof time.Time
		err := s.db.QueryRow(ctx, `SELECT price, as_of FROM quotes_cache WHERE symbol = $1`, symbol).Scan(&price, &asof)
		if err == nil {
			if time.Since(asof) <= s.quotesTTL {
				return price, true
			}
		}
		// stale or missing: fetch
		p, err2 := s.prices.Quote(ctx, symbol)
		if err2 != nil || p <= 0 {
			if err == nil {
				// return stale if we had one
				return price, true
			}
			return 0, false
		}
		_, _ = s.db.Exec(ctx, `
INSERT INTO quotes_cache(symbol, price, as_of, updated_at)
VALUES ($1,$2, now(), now())
ON CONFLICT (symbol) DO UPDATE SET price=EXCLUDED.price, as_of=EXCLUDED.as_of, updated_at=EXCLUDED.updated_at
`, symbol, p)
		return p, true
	}
	// No cache enabled: call provider directly
	p, err := s.prices.Quote(ctx, symbol)
	if err != nil || p <= 0 {
		return 0, false
	}
	return p, true
}

// EnrichTicker returns optional enrichment for a single ticker.
// It uses the configured price provider/cache and fundamentals table when available.
func (s *Service) EnrichTicker(ctx context.Context, ticker string, targetTo *float64) (price *float64, percentUpside *float64, eps *float64, growth *float64, intrinsic *float64, intrinsic2 *float64) {
	// Price and upside
	if s.prices != nil {
		if p, ok := s.getQuote(ctx, ticker); ok && p > 0 {
			cp := p
			price = &cp
			if targetTo != nil && *targetTo > 0 {
				up := (*targetTo / cp) - 1.0
				percentUpside = &up
			}
		}
	}
	// Fundamentals
	if e, g, ok := s.getGrahamValuation(ctx, ticker); ok {
		eps = &e
		growth = &g
		iv := e * (8.5 + 2*g)
		intrinsic = &iv
		if s.corporateBondYieldProvider != nil {
			if bondYield, err := s.corporateBondYieldProvider.GetAAACorporateBondYield(ctx); err == nil && bondYield > 0 {
				iv2 := iv * 4.4 / bondYield
				intrinsic2 = &iv2
			}
		}
	}
	return
}

// getGrahamValuation returns eps_avg and growth_estimate from cache if fresh, otherwise calls provider
// and upserts cache when enabled. Returns false if not available.
func (s *Service) getGrahamValuation(ctx context.Context, ticker string) (float64, float64, bool) {
	// Try cache first
	var eps, growth float64
	var updatedAt time.Time
	err := s.db.QueryRow(ctx, `SELECT eps_avg, growth_estimate, updated_at FROM fundamentals WHERE ticker = $1`, ticker).Scan(&eps, &growth, &updatedAt)
	if err == nil {
		if s.grahamValuationTTL <= 0 || time.Since(updatedAt) <= s.grahamValuationTTL {
			return eps, growth, true
		}
		// stale, consider refresh if provider exists
		if s.grahamValuationProvider != nil {
			if v, g, err2 := s.grahamValuationProvider.GetGrahamValuation(ctx, ticker); err2 == nil && v != 0 {
				_, _ = s.db.Exec(ctx, `
UPSERT INTO fundamentals (ticker, eps_avg, growth_estimate, updated_at)
VALUES ($1, $2, $3, now())
`, ticker, v, g)
				return v, g, true
			}
			// return stale if provider failed
			return eps, growth, true
		}
		return eps, growth, true
	}
	// Not found, fetch if possible
	if s.grahamValuationProvider != nil {
		if v, g, err := s.grahamValuationProvider.GetGrahamValuation(ctx, ticker); err == nil && v != 0 {
			_, _ = s.db.Exec(ctx, `
UPSERT INTO fundamentals (ticker, eps_avg, growth_estimate, updated_at)
VALUES ($1, $2, $3, now())
`, ticker, v, g)
			return v, g, true
		}
	}
	return 0, 0, false
}

// WarmCachesForTopK prefetches quotes and fundamentals for the most recently updated tickers (up to topK).
// This helps stay within free API limits by doing a single daily refresh instead of many on-demand calls.
func (s *Service) WarmCachesForTopK(ctx context.Context) error {
	// Gather top-K distinct tickers by recency
	rows, err := s.db.Query(ctx, `
SELECT DISTINCT ticker FROM stocks
ORDER BY updated_at DESC
LIMIT $1
`, s.topK)
	if err != nil {
		return err
	}
	defer rows.Close()

	tickers := make([]string, 0, s.topK)
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err == nil && t != "" {
			tickers = append(tickers, t)
		}
	}

	// Prefetch quotes and fundamentals with existing TTL behavior
	for _, t := range tickers {
		// Quote: ignore result; side effect updates cache
		if s.prices != nil {
			_, _ = s.getQuote(ctx, t)
		}
		// Fundamentals: triggers refresh only if TTL expired or not present
		_, _, _ = s.getGrahamValuation(ctx, t)
	}
	return nil
}

func brokerageWeights() map[string]float64 {
	return map[string]float64{
		"goldman":       1.3,
		"ubs":           1.2,
		"morgan":        1.15,
		"keycorp":       1.05,
		"piper":         1.05,
		"royal bank":    1.1,
		"evercore":      1.1,
		"hc wainwright": 1.0,
		"default":       1.0,
	}
}

func normalizeBroker(b string) string {
	b = strings.ToLower(b)
	switch {
	case strings.Contains(b, "goldman"):
		return "goldman"
	case strings.Contains(b, "ubs"):
		return "ubs"
	case strings.Contains(b, "morgan"):
		return "morgan"
	case strings.Contains(b, "keycorp"):
		return "keycorp"
	case strings.Contains(b, "piper"):
		return "piper"
	case strings.Contains(b, "royal bank"):
		return "royal bank"
	case strings.Contains(b, "evercore"):
		return "evercore"
	case strings.Contains(b, "wainwright"):
		return "hc wainwright"
	default:
		return "default"
	}
}

func transitionBonus(from, to string) (float64, string) {
	f := strings.ToLower(strings.TrimSpace(from))
	t := strings.ToLower(strings.TrimSpace(to))

	// Map ratings to score bands
	rank := func(s string) int {
		switch s {
		case "strong buy":
			return 3
		case "buy", "outperform", "overweight":
			return 2
		case "equal weight", "neutral", "market perform":
			return 1
		case "underweight", "sell", "underperform":
			return 0
		default:
			return 1
		}
	}
	df := rank(t) - rank(f)
	switch {
	case df >= 2:
		return 2.0, "major upgrade"
	case df == 1:
		return 1.0, "upgrade"
	case df == 0:
		return 0.2, "reaffirm"
	default:
		// downgrade; penalize
		return -1.0, "downgrade"
	}
}

func scoreOne(brokerage, ratingFrom, ratingTo string, targetFrom, targetTo, delta *float64, lastChange *time.Time, weights map[string]float64) (float64, []string) {
	reasons := []string{}
	score := 0.0

	// Base from price target change
	if delta != nil {
		// dampen outliers
		df := math.Tanh(*delta/50.0) * 2.0
		score += df
		reasons = append(reasons, "price target change contribution")
	} else if targetTo != nil && targetFrom == nil {
		score += 0.5
		reasons = append(reasons, "new price target")
	}

	// Rating transition
	tb, label := transitionBonus(ratingFrom, ratingTo)
	score += tb
	reasons = append(reasons, label)

	// Recency bonus
	if lastChange != nil {
		daysAgo := time.Since(*lastChange).Hours() / 24
		if daysAgo < 2 {
			score += 0.5
			reasons = append(reasons, "very recent change")
		} else if daysAgo < 7 {
			score += 0.2
			reasons = append(reasons, "recent change")
		}
	}

	// Brokerage trust
	w := weights[normalizeBroker(brokerage)]
	if w == 0 {
		w = weights["default"]
	}
	score *= w
	reasons = append(reasons, "brokerage weight applied")

	// Floor and cap
	if score > 10 {
		score = 10
	}
	if score < -5 {
		score = -5
	}
	return score, reasons
}

// small generic helpers (no generics to keep go1.22 minor tidy)

func sortSliceStable(a []Recommendation, less func(a, b Recommendation) bool) {
	for i := 0; i < len(a); i++ {
		for j := i + 1; j < len(a); j++ {
			if less(a[j], a[i]) {
				a[i], a[j] = a[j], a[i]
			}
		}
	}
}
