package api

import (
    "context"
    "encoding/json"
    "net/http"
    "strconv"
    "strings"
    "time"

	"stockchallenge/backend/internal/db"
	"stockchallenge/backend/internal/ingest"
	"stockchallenge/backend/internal/rec"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type RouterDeps struct {
    DB          db.DBTX
    Ingest      *ingest.Service
    Recommender *rec.Service
    Log         *zap.SugaredLogger
    FundamentalsAPI string
}

func NewRouter(db db.DBTX, ing *ingest.Service, recommender *rec.Service, log *zap.SugaredLogger, fundamentalsAPI string) http.Handler {
    gin.SetMode(gin.ReleaseMode)
    r := gin.New()
    r.Use(gin.Recovery())
    r.Use(corsMiddleware())

	deps := &RouterDeps{
        DB:              db,
        Ingest:          ing,
        Recommender:     recommender,
        Log:             log,
        FundamentalsAPI: fundamentalsAPI,
    }

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "time": time.Now().UTC()})
	})

	api := r.Group("/api")
	{
		api.GET("/stocks", deps.listStocks)
		api.GET("/stocks/search", deps.searchStocks)
		api.GET("/stocks/sort", deps.sortStocks)
		api.GET("/stocks/:ticker", deps.getStock)
        api.GET("/recommendations", deps.getRecommendations)
        api.POST("/admin/ingest", deps.runIngest)
        api.POST("/admin/fundamentals/refresh", deps.refreshFundamentals)
        api.GET("/watchlist", deps.getWatchlist)
        api.POST("/watchlist", deps.addToWatchlist)
        api.DELETE("/watchlist/:ticker", deps.removeFromWatchlist)
    }

	return r
}

func (h *RouterDeps) listStocks(c *gin.Context) {
    enrich := strings.ToLower(strings.TrimSpace(c.DefaultQuery("enrich", "false"))) == "true"
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	q := `
SELECT id, ticker, company, brokerage, action, rating_from, rating_to, target_from, target_to, last_rating_change_at, price_target_delta, created_at, updated_at
FROM stocks
ORDER BY updated_at DESC
LIMIT $1 OFFSET $2
`
	rows, err := h.DB.Query(c, q, pageSize, offset)
	if err != nil {
		h.Log.Warnf("list query error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	items := []map[string]any{}
    for rows.Next() {
        var (
            id, ticker, company, brokerage, action, ratingFrom, ratingTo string
            targetFrom, targetTo, priceDelta                             *float64
            lastChange                                                   *time.Time
            createdAt, updatedAt                                         time.Time
        )
        if err := rows.Scan(
            &id, &ticker, &company, &brokerage, &action, &ratingFrom, &ratingTo,
            &targetFrom, &targetTo, &lastChange, &priceDelta, &createdAt, &updatedAt,
        ); err != nil {
            h.Log.Warnf("scan error: %v", err)
            continue
        }
        m := gin.H{
            "id":                    id,
            "ticker":                ticker,
            "company":               company,
            "brokerage":             brokerage,
            "action":                action,
            "rating_from":           ratingFrom,
            "rating_to":             ratingTo,
            "target_from":           targetFrom,
            "target_to":             targetTo,
            "last_rating_change_at": lastChange,
            "price_target_delta":    priceDelta,
            "created_at":            createdAt,
            "updated_at":            updatedAt,
        }
        if enrich && h.Recommender != nil {
            cp, up, eps, growth, iv, iv2 := h.Recommender.EnrichTicker(c, ticker, targetTo)
            if cp != nil { m["current_price"] = *cp }
            if up != nil { m["percent_upside"] = *up }
            if eps != nil { m["eps"] = *eps }
            if growth != nil { m["growth"] = *growth }
            if iv != nil { m["intrinsic_value"] = *iv }
            if iv2 != nil { m["intrinsic_value_2"] = *iv2 }
        }
        items = append(items, m)
    }

	// Count
	var total int64
	if err := h.DB.QueryRow(c, `
SELECT count(*) FROM stocks
`).Scan(&total); err != nil {
		total = int64(len(items))
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"page":  page,
		"limit": pageSize,
		"total": total,
	})
}

func (h *RouterDeps) getStock(c *gin.Context) {
    ticker := c.Param("ticker")
	var (
		id, company, brokerage, action, ratingFrom, ratingTo string
		targetFrom, targetTo, priceDelta                     *float64
		lastChange                                           *time.Time
		createdAt, updatedAt                                 time.Time
	)
	err := h.DB.QueryRow(c, `
SELECT id, company, brokerage, action, rating_from, rating_to, target_from, target_to, last_rating_change_at, price_target_delta, created_at, updated_at
FROM stocks WHERE ticker = $1
`, ticker).Scan(&id, &company, &brokerage, &action, &ratingFrom, &ratingTo, &targetFrom, &targetTo, &lastChange, &priceDelta, &createdAt, &updatedAt)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
        return
    }
    // Fire-and-forget refresh for this ticker (quotes always; fundamentals if stale)
    if strings.TrimSpace(h.FundamentalsAPI) != "" {
        h.refreshTickerAsync(ticker)
    }
	// Optional enrichment (current price, percent upside, eps, intrinsic)
	var currPrice, pctUpside, eps, growth, intrinsic, intrinsic2 *float64
	if h.Recommender != nil {
		currPrice, pctUpside, eps, growth, intrinsic, intrinsic2 = h.Recommender.EnrichTicker(c, ticker, targetTo)
	}
	c.JSON(http.StatusOK, gin.H{
		"id":                    id,
		"ticker":                ticker,
		"company":               company,
		"brokerage":             brokerage,
		"action":                action,
		"rating_from":           ratingFrom,
		"rating_to":             ratingTo,
		"target_from":           targetFrom,
		"target_to":             targetTo,
		"last_rating_change_at": lastChange,
		"price_target_delta":    priceDelta,
		"current_price":         currPrice,
		"percent_upside":        pctUpside,
		"eps":                   eps,
		"growth":                growth,
		"intrinsic_value":       intrinsic,
		"intrinsic_value_2":     intrinsic2,
		"created_at":            createdAt,
		"updated_at":            updatedAt,
	})
}

// refreshTickerAsync triggers on-demand updates for the requested ticker using the Fundamentals API.
// Quotes are refreshed unconditionally. Fundamentals refresh is only attempted if stale (> ~30 days) or missing.
func (h *RouterDeps) refreshTickerAsync(ticker string) {
    go func(t string) {
        // Quotes update (best effort, short timeout via HTTP client default in http.Post)
        bodyQ, _ := json.Marshal(gin.H{"symbols": []string{t}})
        _, _ = http.Post(strings.TrimRight(h.FundamentalsAPI, "/")+"/api/update/quotes", "application/json", strings.NewReader(string(bodyQ)))

        // Fundamentals: check staleness (~30 days) then refresh
        const maxAge = 30 * 24 * time.Hour
        ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
        defer cancel()
        var updatedAt time.Time
        err := h.DB.QueryRow(ctx, `SELECT updated_at FROM fundamentals WHERE ticker = $1`, t).Scan(&updatedAt)
        need := false
        if err != nil {
            need = true
        } else if time.Since(updatedAt) > maxAge {
            need = true
        }
        if need {
            bodyF, _ := json.Marshal(gin.H{"symbols": []string{t}, "use_final_metric": false})
            _, _ = http.Post(strings.TrimRight(h.FundamentalsAPI, "/")+"/api/update/fundamentals", "application/json", strings.NewReader(string(bodyF)))
        }
    }(strings.ToUpper(strings.TrimSpace(ticker)))
}

func (h *RouterDeps) searchStocks(c *gin.Context) {
    enrich := strings.ToLower(strings.TrimSpace(c.DefaultQuery("enrich", "false"))) == "true"
    query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query 'q' is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	q := `
SELECT id, ticker, company, brokerage, action, rating_from, rating_to, target_from, target_to, last_rating_change_at, price_target_delta, created_at, updated_at
FROM stocks
WHERE ticker ILIKE '%' || $1 || '%' OR company ILIKE '%' || $1 || '%'
ORDER BY updated_at DESC
LIMIT $2 OFFSET $3
`
	rows, err := h.DB.Query(c, q, query, pageSize, offset)
	if err != nil {
		h.Log.Warnf("search query error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	items := []map[string]any{}
    for rows.Next() {
        var (
            id, ticker, company, brokerage, action, ratingFrom, ratingTo string
            targetFrom, targetTo, priceDelta                             *float64
            lastChange                                                   *time.Time
            createdAt, updatedAt                                         time.Time
        )
        if err := rows.Scan(
            &id, &ticker, &company, &brokerage, &action, &ratingFrom, &ratingTo,
            &targetFrom, &targetTo, &lastChange, &priceDelta, &createdAt, &updatedAt,
        ); err != nil {
            h.Log.Warnf("scan error: %v", err)
            continue
        }
        m := gin.H{
            "id":                    id,
            "ticker":                ticker,
            "company":               company,
            "brokerage":             brokerage,
            "action":                action,
            "rating_from":           ratingFrom,
            "rating_to":             ratingTo,
            "target_from":           targetFrom,
            "target_to":             targetTo,
            "last_rating_change_at": lastChange,
            "price_target_delta":    priceDelta,
            "created_at":            createdAt,
            "updated_at":            updatedAt,
        }
        if enrich && h.Recommender != nil {
            cp, up, eps, growth, iv, iv2 := h.Recommender.EnrichTicker(c, ticker, targetTo)
            if cp != nil { m["current_price"] = *cp }
            if up != nil { m["percent_upside"] = *up }
            if eps != nil { m["eps"] = *eps }
            if growth != nil { m["growth"] = *growth }
            if iv != nil { m["intrinsic_value"] = *iv }
            if iv2 != nil { m["intrinsic_value_2"] = *iv2 }
        }
        items = append(items, m)
    }

	// Count
	var total int64
	if err := h.DB.QueryRow(c, `
SELECT count(*) FROM stocks
WHERE ticker ILIKE '%' || $1 || '%' OR company ILIKE '%' || $1 || '%'
`, query).Scan(&total); err != nil {
		total = int64(len(items))
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"page":  page,
		"limit": pageSize,
		"total": total,
	})
}

func (h *RouterDeps) sortStocks(c *gin.Context) {
    enrich := strings.ToLower(strings.TrimSpace(c.DefaultQuery("enrich", "false"))) == "true"
    sortField := c.DefaultQuery("field", "ticker")
    order := strings.ToUpper(c.DefaultQuery("order", "ASC"))
    if order != "ASC" && order != "DESC" {
        order = "ASC"
    }

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

    // Whitelist sort fields (include commonly used computed/temporal fields)
    switch sortField {
    case "ticker", "company", "brokerage", "action", "rating_from", "rating_to", "target_from", "target_to", "updated_at", "price_target_delta":
    default:
        sortField = "ticker"
    }

	q := `
SELECT id, ticker, company, brokerage, action, rating_from, rating_to, target_from, target_to, last_rating_change_at, price_target_delta, created_at, updated_at
FROM stocks
ORDER BY ` + sortField + ` ` + order + `
LIMIT $1 OFFSET $2
`
	rows, err := h.DB.Query(c, q, pageSize, offset)
	if err != nil {
		h.Log.Warnf("sort query error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()

	items := []map[string]any{}
    for rows.Next() {
        var (
            id, ticker, company, brokerage, action, ratingFrom, ratingTo string
            targetFrom, targetTo, priceDelta                             *float64
            lastChange                                                   *time.Time
            createdAt, updatedAt                                         time.Time
        )
        if err := rows.Scan(
            &id, &ticker, &company, &brokerage, &action, &ratingFrom, &ratingTo,
            &targetFrom, &targetTo, &lastChange, &priceDelta, &createdAt, &updatedAt,
        ); err != nil {
            h.Log.Warnf("scan error: %v", err)
            continue
        }
        m := gin.H{
            "id":                    id,
            "ticker":                ticker,
            "company":               company,
            "brokerage":             brokerage,
            "action":                action,
            "rating_from":           ratingFrom,
            "rating_to":             ratingTo,
            "target_from":           targetFrom,
            "target_to":             targetTo,
            "last_rating_change_at": lastChange,
            "price_target_delta":    priceDelta,
            "created_at":            createdAt,
            "updated_at":            updatedAt,
        }
        if enrich && h.Recommender != nil {
            cp, up, eps, growth, iv, iv2 := h.Recommender.EnrichTicker(c, ticker, targetTo)
            if cp != nil { m["current_price"] = *cp }
            if up != nil { m["percent_upside"] = *up }
            if eps != nil { m["eps"] = *eps }
            if growth != nil { m["growth"] = *growth }
            if iv != nil { m["intrinsic_value"] = *iv }
            if iv2 != nil { m["intrinsic_value_2"] = *iv2 }
        }
        items = append(items, m)
    }

	// Count
	var total int64
	if err := h.DB.QueryRow(c, `
SELECT count(*) FROM stocks
`).Scan(&total); err != nil {
		total = int64(len(items))
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"page":  page,
		"limit": pageSize,
		"total": total,
		"sort":  sortField,
		"order": order,
	})
}

func (h *RouterDeps) getRecommendations(c *gin.Context) {
	top, err := h.Recommender.TopN(c, 5)
	if err != nil {
		h.Log.Warnf("recommendation error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": top})
}

func (h *RouterDeps) runIngest(c *gin.Context) {
	go func() {
		if err := h.Ingest.RunOnce(c); err != nil {
			h.Log.Warnf("manual ingest error: %v", err)
		}
	}()
	c.JSON(http.StatusAccepted, gin.H{"status": "ingest started"})
}

// refreshFundamentals proxies a refresh request to the external Fundamentals API service
// configured via FUNDAMENTALS_API_BASE. Expects JSON body: {"symbols": ["NVDA","AAPL"], "use_final_metric": false}
func (h *RouterDeps) refreshFundamentals(c *gin.Context) {
    if strings.TrimSpace(h.FundamentalsAPI) == "" {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "fundamentals API not configured"})
        return
    }
    // Accept both JSON and query param formats
    var body struct {
        Symbols       []string `json:"symbols"`
        UseFinalMetric bool    `json:"use_final_metric"`
    }
    if err := c.BindJSON(&body); err != nil {
        // fall back to query param
        syms := c.Query("symbols")
        if syms == "" {
            c.JSON(http.StatusBadRequest, gin.H{"error": "symbols required"})
            return
        }
        parts := strings.Split(syms, ",")
        for i := range parts {
            parts[i] = strings.TrimSpace(parts[i])
        }
        body.Symbols = parts
        body.UseFinalMetric = c.DefaultQuery("use_final_metric", "false") == "true"
    }
    // Build request to Fundamentals API
    reqBody, _ := json.Marshal(gin.H{"symbols": body.Symbols, "use_final_metric": body.UseFinalMetric})
    resp, err := http.Post(strings.TrimRight(h.FundamentalsAPI, "/")+"/api/update/fundamentals", "application/json", strings.NewReader(string(reqBody)))
    if err != nil {
        h.Log.Warnf("fundamentals api error: %v", err)
        c.JSON(http.StatusBadGateway, gin.H{"error": "upstream error"})
        return
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        c.JSON(http.StatusBadGateway, gin.H{"error": "upstream status", "status": resp.StatusCode})
        return
    }
    c.JSON(http.StatusAccepted, gin.H{"status": "refresh requested", "symbols": body.Symbols})
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// getWatchlist returns all tickers from the watchlist table.
func (h *RouterDeps) getWatchlist(c *gin.Context) {
    rows, err := h.DB.Query(c, `SELECT ticker, notes, added_at FROM watchlist ORDER BY added_at DESC`)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
        return
    }
    defer rows.Close()
    items := make([]gin.H, 0, 64)
    for rows.Next() {
        var t, notes *string
        var added time.Time
        if err := rows.Scan(&t, &notes, &added); err == nil {
            items = append(items, gin.H{"ticker": t, "notes": notes, "added_at": added})
        }
    }
    c.JSON(http.StatusOK, gin.H{"items": items})
}

// addToWatchlist upserts a single ticker.
func (h *RouterDeps) addToWatchlist(c *gin.Context) {
    var body struct {
        Ticker string  `json:"ticker"`
        Notes  *string `json:"notes"`
    }
    if err := c.BindJSON(&body); err != nil || strings.TrimSpace(body.Ticker) == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ticker required"})
        return
    }
    t := strings.ToUpper(strings.TrimSpace(body.Ticker))
    if _, err := h.DB.Exec(c, `UPSERT INTO watchlist (ticker, notes, added_at) VALUES ($1, $2, now())`, t, body.Notes); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "upsert failed"})
        return
    }
    c.JSON(http.StatusAccepted, gin.H{"ticker": t, "status": "ok"})
}

// removeFromWatchlist deletes a ticker.
func (h *RouterDeps) removeFromWatchlist(c *gin.Context) {
    t := strings.ToUpper(strings.TrimSpace(c.Param("ticker")))
    if t == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "ticker required"})
        return
    }
    if _, err := h.DB.Exec(c, `DELETE FROM watchlist WHERE ticker = $1`, t); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"ticker": t, "status": "deleted"})
}
