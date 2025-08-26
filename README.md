# Stock Info Challenge

> A fully containerized stock analysis and portfolio management platform with AI-powered features

## 📋 Table of Contents

- [Features](#-features)
- [Tech Stack](#️-tech-stack)
- [Quick Start](#-quick-start)
- [Local Dev (No Docker)](#-local-dev-no-docker)
- [API Endpoints](#-api-endpoints)
- [Configuration](#️-configuration)
- [Development](#-development)
- [Project Structure](#-project-structure)
- [Testing](#-testing)
- [Advanced Features](#-advanced-features)
 - [CI/CD](#-cicd-pipeline)
 - [Contributing](#-contributing)

## ✨ Features

- **Portfolio Management**: Upload brokerage screenshots and extract positions using Gemini AI
- **Real-time Quotes**: Fetch current prices for any ticker
- **Smart Recommendations**: Advanced valuation with intrinsic value calculations
- **Fundamentals Analysis**: Automated EPS and growth metrics collection
- **Watchlist Management**: Track your favorite stocks with guaranteed coverage
- **Bond Yield Integration**: Enhanced valuations adjusted by AAA corporate bond yields

## 🛠️ Tech Stack

| Component | Technology |
|-----------|------------|
| **Backend** | Go (Gin, pgx) with REST API |
| **Frontend** | Vue 3, Vite, TypeScript, Pinia, Tailwind |
| **Database** | CockroachDB (single-node for local development) |
| **Orchestration** | Docker Compose |
| **AI Integration** | Google Gemini for image processing |
| **External APIs** | FMP (Financial Modeling Prep), Alpha Vantage, FRED |

## 🚀 Quick Start

### Prerequisites
- Docker and Docker Compose
- Git

### Setup

1. **Clone and setup environment**
   ```bash
   git clone <repository-url>
   cd stock_page
   cp .env.example .env
   ```
   
2. **Configure API keys** (edit `.env`)
   ```bash
   # External ratings ingest
   API_BASE=https://example.com/api/data # Source for ratings/targets (optional)
   API_TOKEN=your_api_token_here         # Raw token only; "Bearer " is added automatically

   # Market/fundamentals data
   FMP_API_KEY=your_fmp_key_here         # For quotes and fundamentals provider
   ALPHAVANTAGE_KEY=your_key_here        # For Python fundamentals tools (EPS/growth)

   # AI portfolio extraction
   GEMINI_API_KEY=your_gemini_key        # For portfolio image processing
   ```

3. **Build and run**
   ```bash
   docker compose up --build
   ```

4. **Access the applications**
   - 🖥️ **Frontend**: http://localhost:5173
   - ⚙️ **Backend API**: http://localhost:8080
   - 🗄️ **Database UI**: http://localhost:8082

### Port Configuration

| Service | Host Port | Container Port | Purpose |
|---------|-----------|----------------|---------|
| Backend | 8080 | 8080 | REST API |
| Frontend | 5173 | 5173 | Vue.js dev server |
| Database (SQL) | 26258 | 26259 | CockroachDB SQL |
| Database (Admin) | 8082 | 8081 | CockroachDB Admin UI |

## 💻 Local Dev (No Docker)

Run the database via Docker, then run backend/frontend on your host for fast iteration:

```bash
# 1) Start only the database
docker compose up -d db

# 2) Copy env and point DB_URL at localhost
cp .env.example .env
export DB_URL=postgresql://root@localhost:26258/stocks?sslmode=disable

# 3) Backend
cd backend && go mod download && go run ./cmd/api

# 4) Frontend (in a new shell)
cd frontend && npm ci && VITE_API_BASE=http://localhost:8080 npm run dev
```

Notes:
- `VITE_API_BASE` tells the frontend where the API lives in dev.
- The backend will create the `stocks` database and run migrations automatically.

## ⚙️ Configuration

### Environment Variables (`.env`)

#### Core Settings
| Variable | Default | Description |
|----------|---------|-------------|
| `API_BASE` | - | External ratings source base URL (optional) |
| `API_TOKEN` | - | Raw API token (Bearer prefix added automatically) |
| `GEMINI_API_KEY` | - | Google Gemini API key for image processing |
| `DB_URL` | `postgresql://root@db:26259/stocks?sslmode=disable` | Database connection string |

#### External APIs
| Variable | Default | Description |
|----------|---------|-------------|
| `FMP_API_KEY` | - | Financial Modeling Prep API key (quotes/fundamentals) |
| `ALPHAVANTAGE_KEY` | - | Alpha Vantage API key for Python fundamentals tools |

#### Application Ports
| Variable | Default | Description |
|----------|---------|-------------|
| `BACKEND_PORT` | `8080` | Backend server port |
| `FRONTEND_PORT` | `5173` | Frontend development server port |
| `FRONTEND_PUBLIC_BASE` | `/` | Public base path for frontend build |
| `VITE_API_BASE` | - | Frontend API base (build arg/env for dev) |

#### Data & Caching
| Variable | Default | Description |
|----------|---------|-------------|
| `QUOTES_TTL` | `24h` | Quote cache time-to-live |
| `QUOTES_MIN_REFRESH_AGE` | `6h` | Skip refreshing quotes newer than this |
| `PRICE_TOPK` | `20` | Top-K enrichment in backend scoring |
| `TOP_RECENT_COUNT` | `50` | Number of recent symbols to track |
| `FUNDAMENTALS_TTL` | `720h` | Cache lifetime for fundamentals in backend |
| `PRICE_WARM_INTERVAL` | `24h` | How often to prefetch top-K quotes/fundamentals |

#### Scheduling & Updates
| Variable | Default | Description |
|----------|---------|-------------|
| `INGEST_INTERVAL` | `15m` | General ingestion interval |
| `INGEST_ON_START` | `true` | Run one ingestion on service start |
| `PRICE_UPDATE_INTERVAL` | `24h` | Price update frequency |
| `FUNDAMENTALS_UPDATE_INTERVAL` | `720h` | Fundamentals update frequency (30 days) |

#### Fundamentals Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `DISABLE_GRAHAM_PROVIDER` | `true` | Use Python fundamentals service |
| `FUNDAMENTALS_API_BASE` | `http://fundamentals-api:9000` | Fundamentals API endpoint |
| `FUNDAMENTALS_SYMBOLS` | `NVDA,AAPL,MSFT` | Comma-separated symbols (or empty for watchlist+recent) |
| `FUNDAMENTALS_USE_FINAL_METRIC` | `false` | Upsert the blended final metric as growth |

## 🔌 API Endpoints

### Health & Status
- `GET /healthz` - Health check endpoint

### Stock Data
- `GET /api/stocks` - List all stocks
- `GET /api/stocks/:ticker` - Get specific stock details  
- `GET /api/quotes/:ticker` - Get current price for any ticker
- `GET /api/stocks/search?q=<query>&page=<n>&limit=<n>` - Search stocks
- `GET /api/stocks/sort?field=<field>&order=ASC|DESC&page=<n>&limit=<n>` - Sort stocks

### Recommendations
- `GET /api/recommendations` - Get investment recommendations
  - Includes `current_price` and `percent_upside` when quotes are cached
  - Includes `eps` and `intrinsic_value` when fundamentals are available  
  - Includes `intrinsic_value_2` (Graham value scaled by AAA corporate bond yield via FRED)

### Portfolio Management (AI-Powered)
> Requires `GEMINI_API_KEY` in environment

- `POST /api/portfolio/upload` - Upload brokerage screenshot
  ```bash
  curl -F image=@/path/to/positions.png http://localhost:8080/api/portfolio/upload
  ```
- `GET /api/portfolio` - Get saved portfolio positions

### Watchlist
- `GET /api/watchlist` - Get watchlist
- `POST /api/watchlist` - Add to watchlist
  ```json
  { "ticker": "NVDA", "notes": "high conviction" }
  ```
- `DELETE /api/watchlist/:ticker` - Remove from watchlist

### Admin Operations
- `POST /api/admin/ingest` - Manual data ingestion
- `POST /api/admin/fundamentals/refresh` - Refresh fundamentals data
  ```json
  { "symbols": ["NVDA","AAPL"], "use_final_metric": false }
  ```

## 🏗️ Docker Services

| Service | Description |
|---------|-------------|
| **db** | CockroachDB single-node with persistent volume |
| **backend** | Go application with migrations, REST API, and ingestion |
| **frontend** | Vite development server with backend proxy |
| **fundamentals-api** | FastAPI service for fundamentals data (EPS/growth + quotes cache) |
| **fundamentals-scheduler** | Automated data refresh service |

## 📁 Project Structure

```
stock_page/
├── backend/                    # Go backend application
│   ├── cmd/api/main.go        # Application entry point
│   ├── internal/              # Internal packages
│   │   ├── api/               # HTTP handlers and router
│   │   ├── db/                # Database pool and migrations
│   │   ├── ingest/            # External API client and ingestion
│   │   ├── models/            # Domain structs and types
│   │   ├── rec/               # Recommendation scoring engine
│   │   ├── portfolio/         # AI-powered portfolio OCR
│   │   └── config/            # Environment configuration
│   └── Dockerfile             # Backend container definition
├── frontend/                  # Vue.js frontend application  
│   ├── src/
│   │   ├── main.ts           # Application entry point
│   │   ├── App.vue           # Root component
│   │   ├── stores/           # Pinia state management
│   │   ├── components/       # Reusable Vue components
│   │   └── pages/            # Page-level components
│   ├── package.json          # Node.js dependencies
│   ├── vite.config.ts        # Vite configuration
│   ├── tailwind.config.js    # Tailwind CSS config
│   └── Dockerfile            # Frontend container (dev mode)
├── docker-compose.yml         # Service orchestration
├── .env.example              # Environment template
└── README.md                 # This file
```

## 🛠️ Development

### Build Commands
```bash
# Build all services
docker compose build

# Start all services
docker compose up

# Build and start (rebuild if needed)
docker compose up --build

# Start in background
docker compose up -d
```

### Local Development
```bash
# Backend development
cd backend && go mod download && go test ./...

# Frontend development
cd frontend && npm ci && npm run dev

# View logs
docker compose logs -f [service-name]
```

## 🧪 Testing

| Component | Command | Status |
|-----------|---------|--------|
| **Backend** | `cd backend && go test ./... && go vet ./...` | ✅ Configured |
| **Frontend** | `cd frontend && npm test` | ⏳ Not configured |

## 🚀 Advanced Features

### Fundamentals Data Integration

#### Automated Setup (Recommended)
Configure in `.env` and run with Docker Compose:
```bash
DISABLE_GRAHAM_PROVIDER=true
ALPHAVANTAGE_KEY=your_key_here
FUNDAMENTALS_SYMBOLS=NVDA,AAPL,MSFT  # Your stock universe
# Optional: prefer blended growth metric when upserting via tools
FUNDAMENTALS_USE_FINAL_METRIC=false
```

**Included Services:**
- `fundamentals-api`: FastAPI service for on-demand fundamentals
- `fundamentals-scheduler`: Automated data refresh
  - Prices: Daily updates
  - Fundamentals: Monthly updates  
  - Scope: Watchlist ∪ Portfolio (or `FUNDAMENTALS_SYMBOLS`)
  - Smart caching: Skips recent quotes (< `QUOTES_MIN_REFRESH_AGE`)

#### Manual Setup
```bash
# Install dependencies
pip install -r backend/tools/requirements.txt

# Set environment
export DB_URL=postgresql://root@localhost:26258/stocks?sslmode=disable
export ALPHAVANTAGE_KEY=your_key_here

# Run fundamentals update
python3 backend/tools/upsert_fundamentals.py --symbols NVDA,AAPL,MSFT

# Inspect single symbol metrics
python3 backend/tools/eps_metric_free.py --symbol NVDA --json
```

### AI-Powered Portfolio Management

Upload brokerage screenshots to automatically extract positions:

```bash
# Set Gemini API key
GEMINI_API_KEY=your_gemini_key
GEMINI_MODEL_ID=gemini-2.5-flash-lite  # Optional, this is the default

# Upload portfolio screenshot
curl -F image=@/path/to/positions.png http://localhost:8080/api/portfolio/upload

# Retrieve extracted positions
curl http://localhost:8080/api/portfolio
```

The AI extracts aligned arrays:
- **Instruments**: Stock tickers
- **Positions**: Share quantities  
- **Average Price**: Cost basis per share

### Watchlist Management

Manage your tracked stocks with guaranteed data coverage:

```bash
# Get current watchlist
curl http://localhost:8080/api/watchlist

# Add stock to watchlist
curl -X POST http://localhost:8080/api/watchlist \
  -H "Content-Type: application/json" \
  -d '{"ticker": "NVDA", "notes": "high conviction"}'

# Remove from watchlist
curl -X DELETE http://localhost:8080/api/watchlist/NVDA
```

## 🔄 CI/CD Pipeline

### GitHub Actions
Runs on pushes and PRs to `main`:

| Component | Runtime | Commands |
|-----------|---------|----------|
| **Backend** | Go 1.23.x | `go mod download`, `go test ./...`, `go vet ./...` |
| **Frontend** | Node 20.x | `npm ci`, `npm run build` |

### Production Considerations
- ✅ Deterministic installs via `npm ci`
- ✅ Frontend Dockerfile enforces `npm ci`
- 🔒 Never commit real API keys
- 📦 Future: Add nginx stage for production frontend serving

## 📝 Important Notes

- **Security**: Keep `.env.example` with placeholders only
- **Privacy**: Avoid company names or confidential info in public repos  
- **Dependencies**: Commit `package-lock.json`, avoid `npm install` in CI
- **Scheduling**: Automatic data refresh respects rate limits and caching
