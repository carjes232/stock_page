# syntax=docker/dockerfile:1

# Dev image to run Vite with HMR
FROM node:20-alpine

WORKDIR /app

# Install dependencies first (better layer caching)
COPY package.json package-lock.json* .npmrc* ./
# Enforce deterministic install via npm ci; fail if lockfile missing
RUN if [ -f package-lock.json ]; then npm ci; else echo "Error: package-lock.json is required. Generate it locally via 'npm install --package-lock-only'" && exit 1; fi

# Copy source
COPY . .

EXPOSE 5173

# Default to npm; parameterize port via env
ENV VITE_PORT=5173
CMD ["sh", "-lc", "npm run dev -- --host 0.0.0.0 --port ${VITE_PORT:-5173}"]
