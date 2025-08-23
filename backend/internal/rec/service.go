package rec

import (
	"context"
	"math"
	"strings"
	"time"

	"stockchallenge/backend/internal/db"
)

type Service struct {
	db db.DBTX
}

func NewService(db db.DBTX) *Service {
	return &Service{db: db}
}

type Recommendation struct {
	Ticker       string     `json:"ticker"`
	Company      string     `json:"company"`
	Brokerage    string     `json:"brokerage"`
	RatingFrom   string     `json:"rating_from"`
	RatingTo     string     `json:"rating_to"`
	TargetFrom   *float64   `json:"target_from,omitempty"`
	TargetTo     *float64   `json:"target_to,omitempty"`
	PriceDelta   *float64   `json:"price_target_delta,omitempty"`
	Score        float64    `json:"score"`
	ScoreReasons []string   `json:"reasons"`
	LastChange   *time.Time `json:"last_rating_change_at,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
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
	}

	// sort by score desc, tie-break by updated_at desc
	sortSliceStable(recs, func(a, b Recommendation) bool {
		if a.Score == b.Score {
			return a.UpdatedAt.After(b.UpdatedAt)
		}
		return a.Score > b.Score
	})

	if len(recs) > n {
		recs = recs[:n]
	}
	return recs, nil
}

func brokerageWeights() map[string]float64 {
	// Simple priors; in real life you'd calibrate from backtests
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
		df := math.Tanh(*delta/50.0) * 2.0 // scale -2..2 roughly
		score += df
		reasons = append(reasons, "price target change contribution")
	} else if targetTo != nil && targetFrom == nil {
		score += 0.5 // new price targets are a good signal
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
