package api

import (
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
}

func NewRouter(db db.DBTX, ing *ingest.Service, recommender *rec.Service, log *zap.SugaredLogger) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	deps := &RouterDeps{
		DB:          db,
		Ingest:      ing,
		Recommender: recommender,
		Log:         log,
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
	}

	return r
}

func (h *RouterDeps) listStocks(c *gin.Context) {
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
		items = append(items, gin.H{
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
		})
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

func (h *RouterDeps) searchStocks(c *gin.Context) {
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
		items = append(items, gin.H{
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
		})
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
		items = append(items, gin.H{
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
		})
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
