package marketdata

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "strings"
    "time"
)

// PriceProvider defines the minimal interface rec.Service needs for quotes.
type PriceProvider interface {
    Quote(ctx context.Context, symbol string) (float64, error)
}

type FMPClient struct {
    apiKey string
    base   string
    http   *http.Client
}

func NewFMPClient(apiKey string) *FMPClient {
    return &FMPClient{
        apiKey: strings.TrimSpace(apiKey),
        base:   "https://financialmodelingprep.com/api",
        http: &http.Client{
            Timeout: 8 * time.Second,
        },
    }
}

type fmpQuoteShort struct {
    Symbol string  `json:"symbol"`
    Price  float64 `json:"price"`
    Volume float64 `json:"volume"`
}

// Quote fetches a real-time snapshot price for the given symbol using
// FMP's quote-short endpoint.
func (c *FMPClient) Quote(ctx context.Context, symbol string) (float64, error) {
    if c == nil || c.apiKey == "" {
        return 0, fmt.Errorf("fmp: missing api key")
    }
    sym := strings.ToUpper(strings.TrimSpace(symbol))
    u := fmt.Sprintf("%s/v3/quote-short/%s?apikey=%s", c.base, url.PathEscape(sym), url.QueryEscape(c.apiKey))
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
    if err != nil {
        return 0, err
    }
    resp, err := c.http.Do(req)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return 0, fmt.Errorf("fmp: http %d", resp.StatusCode)
    }
    var out []fmpQuoteShort
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
        return 0, err
    }
    if len(out) == 0 {
        return 0, fmt.Errorf("fmp: empty quote")
    }
    return out[0].Price, nil
}

