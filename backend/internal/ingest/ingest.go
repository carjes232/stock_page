package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"stockchallenge/backend/internal/db"

	"go.uber.org/zap"
)

type Service struct {
	apiBase string
	token   string
	db      db.DBTX
	log     *zap.SugaredLogger
	client  *http.Client
}

func NewService(apiBase, token string, db db.DBTX, log *zap.SugaredLogger) *Service {
	return &Service{
		apiBase: strings.TrimSpace(apiBase),
		token:   strings.TrimSpace(token),
		db:      db,
		log:     log,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

type apiResponse struct {
	Items    []apiItem `json:"items"`
	NextPage string    `json:"next_page"`
}

type apiItem struct {
	Ticker     string  `json:"ticker"`
	Company    string  `json:"company"`
	Brokerage  string  `json:"brokerage"`
	Action     string  `json:"action"`
	RatingFrom string  `json:"rating_from"`
	RatingTo   string  `json:"rating_to"`
	TargetFrom *string `json:"target_from"`
	TargetTo   *string `json:"target_to"`
	Time       *string `json:"time"` // ignore if present to avoid DisallowUnknownFields error
	// Some fields might not exist; we focus on the above from sample.
}

func (s *Service) RunOnce(ctx context.Context) error {
	if s.apiBase == "" || s.token == "" {
		return fmt.Errorf("missing API base or token")
	}

	next := ""
	total := 0
	for {
		items, np, err := s.fetchPage(ctx, next)
		if err != nil {
			return err
		}
		if len(items) == 0 && np == "" {
			break
		}
		if err := s.upsertItems(ctx, items); err != nil {
			return err
		}
		total += len(items)
		if np == "" {
			break
		}
		next = np
	}
	s.log.Infof("ingest completed: %d items", total)
	return nil
}

func StartCron(svc *Service, every time.Duration, log *zap.SugaredLogger, stop <-chan struct{}) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			ctx, cancel := context.WithTimeout(context.Background(), 2*every)
			_ = svc.RunOnce(ctx) // errors already logged by RunOnce
			cancel()
		case <-stop:
			log.Infof("ingest cron stopped")
			return
		}
	}
}

func (s *Service) fetchPage(ctx context.Context, next string) ([]apiItem, string, error) {
	u, err := url.Parse(s.apiBase)
	if err != nil {
		return nil, "", fmt.Errorf("invalid api base: %w", err)
	}
	q := u.Query()
	if next != "" {
		q.Set("next_page", next)
	}
	u.RawQuery = q.Encode()

	// Debug logging: print full URL (do not log tokens)
	if s.log != nil {
		s.log.Infof("ingest fetching URL: %s", u.String())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}
	// Ensure Authorization header has Bearer prefix even if env provided raw token
	auth := s.token
	low := strings.ToLower(auth)
	if !strings.HasPrefix(low, "bearer ") && auth != "" {
		auth = "Bearer " + auth
	}
	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, "", fmt.Errorf("api error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var out apiResponse
	dec := json.NewDecoder(resp.Body)
	// Be tolerant to unknown fields to avoid breaking if API changes
	if err := dec.Decode(&out); err != nil {
		if s.log != nil {
			s.log.Warnf("decode error: %v", err)
		}
		return nil, "", nil
	}
	return out.Items, out.NextPage, nil
}

func parseDollars(s *string) *float64 {
    if s == nil {
        return nil
    }
    v := strings.TrimSpace(*s)
    // Be tolerant to formats like "$ 50.25" or spaces within
    v = strings.ReplaceAll(v, "$", "")
    v = strings.ReplaceAll(v, ",", "")
    v = strings.ReplaceAll(v, " ", "")
    if v == "" {
        return nil
    }
    f, err := parseFloat(v)
    if err != nil {
        return nil
    }
    return &f
}

func parseFloat(v string) (float64, error) {
	return strconvParse(v)
}

func parseTime(t *string) *time.Time {
    if t == nil {
        return nil
    }
    v := strings.TrimSpace(*t)
    if v == "" {
        return nil
    }
    // Be strict: accept RFC3339, RFC3339Nano, and RFC3339 without seconds (with zone)
    if parsed, err := time.Parse(time.RFC3339, v); err == nil {
        return &parsed
    }
    if parsed, err := time.Parse(time.RFC3339Nano, v); err == nil {
        return &parsed
    }
    if parsed, err := time.Parse("2006-01-02T15:04Z07:00", v); err == nil {
        return &parsed
    }
    return nil
}

// indirection for testability
var strconvParse = func(v string) (float64, error) {
	return strconv.ParseFloat(v, 64)
}

func (s *Service) upsertItems(ctx context.Context, items []apiItem) error {
	batch := &pgxBatch{}
	now := time.Now().UTC()

	for _, it := range items {
		targetFrom := parseDollars(it.TargetFrom)
		targetTo := parseDollars(it.TargetTo)
		lastRatingChange := parseTime(it.Time)

		sql := `
INSERT INTO stocks (ticker, company, brokerage, action, rating_from, rating_to, target_from, target_to, last_rating_change_at, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (ticker) DO UPDATE SET
 company = EXCLUDED.company,
 brokerage = EXCLUDED.brokerage,
 action = EXCLUDED.action,
 rating_from = EXCLUDED.rating_from,
 rating_to = EXCLUDED.rating_to,
 target_from = EXCLUDED.target_from,
 target_to = EXCLUDED.target_to,
 last_rating_change_at = EXCLUDED.last_rating_change_at,
 updated_at = EXCLUDED.updated_at
`
		batch.Queue(sql, it.Ticker, it.Company, it.Brokerage, it.Action, it.RatingFrom, it.RatingTo, targetFrom, targetTo, lastRatingChange, now, now)
	}
	return batch.Send(ctx, s.db)
}

// Minimal batch wrapper to avoid importing pgx Batch everywhere
type pgxBatch struct {
	stmts []stmt
}
type stmt struct {
	sql  string
	args []any
}

func (b *pgxBatch) Queue(sql string, args ...any) {
	b.stmts = append(b.stmts, stmt{sql: sql, args: args})
}

func (b *pgxBatch) Send(ctx context.Context, pool db.DBTX) error {
	if len(b.stmts) == 0 {
		return nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	for _, s := range b.stmts {
		if _, err := tx.Exec(ctx, s.sql, s.args...); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}
