package portfolio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"stockchallenge/backend/internal/db"

	"github.com/google/generative-ai-go/genai"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

// PositionsOut schema that matches the required JSON
type PositionsOut struct {
	Instruments []string  `json:"INSTRUMENTS"`
	Position    []float64 `json:"POSITION"`
	AvgPrice    []float64 `json:"AVG PRICE"`
}

const instruction = `
You are a precise data-extraction engine.

Goal: Return ONLY a JSON object with three arrays: "INSTRUMENTS", "POSITION", "AVG PRICE".
Use the table under the "Positions" header. Columns to read: INSTRUMENT (ticker only), POSITION, AVG PRICE.
Ignore all other columns.

Normalization:
- INSTRUMENTS: extract the ticker only; drop venue/exchange suffixes (e.g., "NASDAQ.NMS", "NYSE", "ARCA").
- POSITION and AVG PRICE: numbers only; drop currency letters/symbols; use "." as decimal; no thousands separators.
- If a row lacks POSITION or AVG PRICE, omit that row.
- Keep original visual order from top to bottom.

Return just the JSON.
`

type Service struct {
	DB    db.DBTX
	Log   *zap.SugaredLogger
	GenAI *genai.GenerativeModel
}

func NewService(db db.DBTX, log *zap.SugaredLogger, apiKey, modelID string) (*Service, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("genai.NewClient: %w", err)
	}

	model := client.GenerativeModel(modelID)
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(instruction)},
	}
	model.GenerationConfig = genai.GenerationConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"INSTRUMENTS": {
					Type:  genai.TypeArray,
					Items: &genai.Schema{Type: genai.TypeString},
				},
				"POSITION": {
					Type:  genai.TypeArray,
					Items: &genai.Schema{Type: genai.TypeNumber},
				},
				"AVG PRICE": {
					Type:  genai.TypeArray,
					Items: &genai.Schema{Type: genai.TypeNumber},
				},
			},
			Required: []string{"POSITION", "AVG PRICE"},
		},
	}

	return &Service{
		DB:    db,
		Log:   log,
		GenAI: model,
	}, nil
}

func (s *Service) ExtractAndSavePortfolio(ctx context.Context, userID string, imageData []byte) (*PositionsOut, error) {
	mimeType := http.DetectContentType(imageData)
	format := extractImageFormat(mimeType)
	resp, err := s.GenAI.GenerateContent(
		ctx,
		genai.ImageData(format, imageData),
		genai.Text("Extract the three arrays from this screenshot."),
	)
	if err != nil {
		return nil, fmt.Errorf("GenerateContent: %w", err)
	}

	jsonText := extractText(resp)
	if strings.TrimSpace(jsonText) == "" {
		return nil, fmt.Errorf("model returned empty text")
	}

	var out PositionsOut
	if err := json.Unmarshal([]byte(jsonText), &out); err != nil {
		return nil, fmt.Errorf("unmarshal model JSON: %w, raw=%s", err, jsonText)
	}

	if err := s.savePortfolio(ctx, userID, &out); err != nil {
		return nil, fmt.Errorf("failed to save portfolio: %w", err)
	}

	return &out, nil
}

func (s *Service) savePortfolio(ctx context.Context, userID string, data *PositionsOut) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := `
	INSERT INTO portfolio (user_id, ticker, position, average_price)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (user_id, ticker) DO UPDATE SET
		position = EXCLUDED.position,
		average_price = EXCLUDED.average_price,
		updated_at = now()
	`

	for i := range data.Instruments {
		if _, err := tx.Exec(ctx, q, userID, data.Instruments[i], data.Position[i], data.AvgPrice[i]); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// extractImageFormat converts MIME type to format string expected by Gemini AI
func extractImageFormat(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return "jpeg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	default:
		// Default to jpeg for unknown types
		return "jpeg"
	}
}

func extractText(resp *genai.GenerateContentResponse) string {
	var b strings.Builder
	for _, c := range resp.Candidates {
		if c.Content == nil {
			continue
		}
		for _, p := range c.Content.Parts {
			if t, ok := p.(genai.Text); ok {
				b.WriteString(string(t))
			}
		}
	}
	return b.String()
}
