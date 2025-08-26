package marketdata

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

// CorporateBondYieldProvider defines the minimal interface for fetching the AAA corporate bond yield.
type CorporateBondYieldProvider interface {
	GetAAACorporateBondYield(ctx context.Context) (float64, error)
}

type FredClient struct {
	http *http.Client
}

func NewFredClient() *FredClient {
	return &FredClient{
		http: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

// GetAAACorporateBondYield fetches the Moody's Seasoned Aaa Corporate Bond Yield from the FRED website.
func (c *FredClient) GetAAACorporateBondYield(ctx context.Context) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://fred.stlouisfed.org/series/AAA", nil)
	if err != nil {
		return 0, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("fred: http %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	re := regexp.MustCompile(`<span class="series-meta-observation-value">([\d.]+)</span>`)
	matches := re.FindStringSubmatch(string(body))

	if len(matches) < 2 {
		return 0, fmt.Errorf("fred: could not find yield value")
	}

	yield, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, err
	}

	return yield, nil
}