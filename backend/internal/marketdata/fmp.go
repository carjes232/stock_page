package marketdata

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
    "time"
)

// PriceProvider defines the minimal interface rec.Service needs for quotes.
type PriceProvider interface {
	Quote(ctx context.Context, symbol string) (float64, error)
}

// GrahamValuationProvider defines the minimal interface for fetching data for a Graham valuation.
type GrahamValuationProvider interface {
	GetGrahamValuation(ctx context.Context, symbol string) (eps float64, growthRate float64, err error)
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

type fmpKeyMetricsTTM struct {
	NetIncomePerShareTTM float64 `json:"netIncomePerShareTTM"`
}

type fmpFinancialGrowth struct {
    // FMP uses camelCase "epsGrowth" in financial-growth endpoint
    EPSGrowth float64 `json:"epsGrowth"`
}

// GetGrahamValuation fetches the TTM EPS and the most recent annual EPS growth rate.
func (c *FMPClient) GetGrahamValuation(ctx context.Context, symbol string) (float64, float64, error) {
	if c == nil || c.apiKey == "" {
		return 0, 0, fmt.Errorf("fmp: missing api key")
	}
	sym := strings.ToUpper(strings.TrimSpace(symbol))

	// Get TTM EPS
	epsURL := fmt.Sprintf("%s/v3/key-metrics-ttm/%s?apikey=%s", c.base, url.PathEscape(sym), url.QueryEscape(c.apiKey))
	eps, err := c.fetchGrahamValuationData(ctx, epsURL, func(body []byte) (float64, error) {
		var metrics []fmpKeyMetricsTTM
		if err := json.Unmarshal(body, &metrics); err != nil {
			return 0, err
		}
		if len(metrics) == 0 {
			return 0, fmt.Errorf("fmp: empty key metrics")
		}
		return metrics[0].NetIncomePerShareTTM, nil
	})
	if err != nil {
		return 0, 0, err
	}

	// Get growth rate
	growthURL := fmt.Sprintf("%s/v3/financial-growth/%s?period=annual&limit=1&apikey=%s", c.base, url.PathEscape(sym), url.QueryEscape(c.apiKey))
	growth, err := c.fetchGrahamValuationData(ctx, growthURL, func(body []byte) (float64, error) {
		var growth []fmpFinancialGrowth
		if err := json.Unmarshal(body, &growth); err != nil {
			return 0, err
		}
		if len(growth) == 0 {
			return 0, fmt.Errorf("fmp: empty financial growth")
		}
		return growth[0].EPSGrowth, nil
	})
	if err != nil {
		return 0, 0, err
	}

	return eps, growth, nil
}

func (c *FMPClient) fetchGrahamValuationData(ctx context.Context, url string, parser func([]byte) (float64, error)) (float64, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return 0, err
    }
    return parser(body)
}
