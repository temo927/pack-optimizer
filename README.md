# Pack Optimizer

A full-stack application for calculating optimal pack distributions for customer orders. The system determines the minimum number of items and packs needed to fulfill an order while adhering to business rules.

## Architecture & Design Decisions

### Hexagonal Architecture (Ports & Adapters)

We've implemented a **hexagonal architecture** to ensure clean separation of concerns and maintainability:

- **Domain Layer** (`internal/domain/`): Contains core business logic interfaces (ports) and models. This layer is framework-agnostic and defines contracts that adapters must implement.
  - **Why**: Keeps business logic independent of infrastructure, making it testable and allowing easy swapping of implementations (e.g., switching from PostgreSQL to another database).

- **Application Layer** (`internal/app/`): Implements domain interfaces with business logic.
  - **Calculator Service**: Uses dynamic programming (DP) to solve the pack optimization problem efficiently.
  - **Packs Service**: Manages pack size operations with caching for performance.
  - **Why**: Separates business rules from infrastructure, enabling unit testing without external dependencies.

- **Adapters Layer** (`internal/adapters/`): Implements domain ports with concrete technologies.
  - **HTTP Adapter**: Handles REST API requests/responses using `chi` router.
  - **PostgreSQL Adapter**: Implements persistence with versioned, append-only storage.
  - **Redis Adapter**: Provides caching layer for performance optimization.
  - **Why**: Isolates external dependencies, making it easy to replace implementations or add new adapters (e.g., gRPC, MongoDB).

- **Platform Layer** (`internal/platform/`): Wires dependencies together (dependency injection).
  - **Why**: Centralizes configuration and initialization, following the Dependency Inversion Principle.

### Technology Choices

- **Go 1.23**: Strong typing, excellent concurrency, and fast compilation. Ideal for building robust APIs.
- **PostgreSQL**: Reliable relational database with array support for storing pack sizes efficiently.
- **Redis**: In-memory caching to reduce database load and improve response times.
- **React + Vite**: Modern frontend with fast development experience and optimized production builds.
- **Docker Compose**: Simplifies local development with containerized services.

### Data Persistence Strategy

We use a **versioned, append-only** approach for pack sizes:
- Each change creates a new version (new row) instead of updating existing data.
- **Why**: Provides audit trail, enables rollback capabilities, and simplifies concurrent access patterns.
- The `GetAllActive()` method always retrieves the latest version, ensuring consistency.

### Caching Strategy

- Pack sizes are cached with version-based keys to ensure cache invalidation on updates.
- **Why**: Reduces database load and improves API response times, especially for frequently accessed data.

### Security Features

The application includes production-ready security middleware:

- **Rate Limiting**: IP-based rate limiting using token bucket algorithm
  - Default: 100 requests per minute per IP
  - Configurable via `RATE_LIMIT_RPM` and `RATE_LIMIT_BURST` environment variables
  - Returns `429 Too Many Requests` when limit exceeded

- **DDoS Protection**: Multiple layers of protection against DDoS attacks
  - Request size limits (default: 10MB)
  - Header size limits (default: 8KB)
  - Suspicious request pattern detection
  - SQL injection and XSS pattern detection

- **Security Headers**: HTTP security headers on all responses
  - `X-Frame-Options: DENY` - Prevents clickjacking
  - `X-Content-Type-Options: nosniff` - Prevents MIME sniffing
  - `X-XSS-Protection: 1; mode=block` - XSS protection
  - `Content-Security-Policy` - Content security policy

- **Configurable**: All security features can be enabled/disabled via environment variables

## Quick Start

### Prerequisites

- Docker and Docker Compose

### Launch with Docker

```bash
# Clone the repository
git clone <repository-url>
cd pack-optimizer

# Start all services
docker compose up --build

# Or use Makefile
make up

Services will be available at:
- **Frontend**: `http://localhost:5173`
- **API**: `http://localhost:8080/api/v1`
- **PostgreSQL**: `localhost:5432`
- **Redis**: `localhost:6379`

The database will automatically:
- Create the schema on first startup
- Seed initial pack sizes: [250, 500, 1000, 2000, 5000]

### Running Tests

Run unit and integration tests:

```bash
# Run all unit tests
make test

# Run integration tests
make itest
```

### Troubleshooting

#### Make Command Issues (macOS/Linux)

If you get `getcwd: Operation not permitted` or `No rule to make target 'up'` error:

1. **Use direct docker compose command:**
   ```bash
   docker compose up --build
   ```

2. **Grant Full Disk Access to Terminal (macOS only):**
   - Go to System Settings → Privacy & Security → Full Disk Access
   - Add Terminal (or iTerm2 if using it)
   - Restart Terminal

3. **Check directory permissions:**
   ```bash
   ls -la
   ```
   Ensure you have read/write permissions in the directory.

4. **Move project if in restricted location:**
   If the project is in Downloads or Desktop, try moving it to `~/projects/` or `/usr/local/`

#### Container Issues

If the API endpoint is not accessible when running in containers:

1. **Check if containers are running:**
   ```bash
   docker compose ps
   ```
   All services (api, frontend, db, redis) should show "Up" status.

2. **Check API logs for errors:**
   ```bash
   docker compose logs api
   ```
   Look for "HTTP server starting" message with the port number.

3. **Test health endpoint:**
   ```bash
   curl http://localhost:8080/api/v1/healthz
   ```
   Should return `200 OK`.

4. **Rebuild containers if needed:**
   ```bash
   docker compose down
   docker compose build --no-cache
   docker compose up
   ```

## Features

### Core Functionality

1. **Pack Size Management**
   - View current pack sizes
   - Add new pack sizes (one at a time)
   - Remove pack sizes
   - Pack sizes must be between 1 and 10,000 items

2. **Pack Calculation**
   - Calculate optimal pack distribution for any order amount
   - Supports custom pack sizes or uses active pack sizes
   - Order amount must be between 1 and 1,000,000 items
   - Algorithm optimizes for:
     - **Rule 1**: Only whole packs (no partial packs)
     - **Rule 2**: Minimum total items (minimize overage)
     - **Rule 3**: Minimum number of packs (when items are equal)

3. **User Interface**
   - Modern, responsive design with Deep Navy/Indigo theme
   - Real-time validation and error messages
   - Interactive pack size management with delete buttons
   - Loading states and user feedback

## Project Structure

```
pack-optimizer/
├── backend/
│   ├── cmd/
│   │   └── api/
│   │       └── main.go              # Application entry point
│   ├── internal/
│   │   ├── adapters/                 # Infrastructure adapters
│   │   │   ├── http/                 # HTTP handlers and routing
│   │   │   ├── postgres/             # PostgreSQL repository
│   │   │   └── redis/                 # Redis cache adapter
│   │   ├── app/                      # Application services
│   │   │   ├── calculator/           # Pack calculation logic (DP algorithm)
│   │   │   └── packs/                # Pack size management service
│   │   ├── domain/                   # Domain models and interfaces (ports)
│   │   ├── integration/              # Integration tests
│   │   └── platform/                 # Dependency injection and configuration
│   ├── Dockerfile
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── main.tsx                  # React entry point
│   │   └── ui/
│   │       └── App.tsx               # Main UI component
│   ├── Dockerfile
│   └── package.json
├── migrations/
│   └── 0001_init.sql                 # Database schema and initial data
├── docker-compose.yml                # Local development setup
└── README.md
```
## API Documentation

### Base URL
`http://localhost:8080/api/v1` (when running locally)

### Endpoints

#### GET `/packs`
Get current active pack sizes.

**Endpoint:** `GET /api/v1/packs`

**Response:**
```json
{
  "sizes": [250, 500, 1000, 2000, 5000]
}
```

#### PUT `/packs`
Replace all pack sizes with a new set.

**Endpoint:** `PUT /api/v1/packs`

**Request:**
```json
{
  "sizes": [250, 500, 1000, 2000, 5000]
}
```

**Response:**
```json
{
  "sizes": [250, 500, 1000, 2000, 5000]
}
```

#### DELETE `/packs/{size}`
Remove a specific pack size.

**Endpoint:** `DELETE /api/v1/packs/{size}`

**Example:** `DELETE /api/v1/packs/250`

**Response:**
```json
{
  "sizes": [500, 1000, 2000, 5000]
}
```

#### POST `/calculate`
Calculate optimal pack distribution.

**Endpoint:** `POST /api/v1/calculate`

**Request:**
```json
{
  "amount": 500000
}
```

Or with custom pack sizes:
```json
{
  "amount": 500000,
  "sizes": [23, 31, 53]
}
```

**Response:**
```json
{
  "amount": 500000,
  "totalItems": 500000,
  "totalPacks": 9438,
  "overage": 0,
  "breakdown": {
    "53": 9429,
    "31": 7,
    "23": 2
  }
}
```

#### GET `/healthz`
Health check endpoint.

**Endpoint:** `GET /api/v1/healthz`

**Response:** `200 OK`

### Testing with curl

Here are curl commands to test all endpoints:

```bash
# 1. Health check
curl http://localhost:8080/api/v1/healthz

# 2. Get current pack sizes
curl http://localhost:8080/api/v1/packs

# 3. Replace all pack sizes
curl -X PUT http://localhost:8080/api/v1/packs \
  -H "Content-Type: application/json" \
  -d '{"sizes": [250, 500, 1000, 2000, 5000]}'

# 4. Delete a pack size
curl -X DELETE http://localhost:8080/api/v1/packs/250

# 5. Calculate with default pack sizes
curl -X POST http://localhost:8080/api/v1/calculate \
  -H "Content-Type: application/json" \
  -d '{"amount": 500000}'

# 6. Calculate with custom pack sizes
curl -X POST http://localhost:8080/api/v1/calculate \
  -H "Content-Type: application/json" \
  -d '{"amount": 500000, "sizes": [23, 31, 53]}'

# 7. Pretty print JSON responses (requires jq)
curl -s http://localhost:8080/api/v1/packs | jq
```

## Algorithm: Dynamic Programming

The pack calculation uses a **dynamic programming** approach:

1. **Problem**: Find the minimum number of items ≥ amount using only whole packs, then minimize the number of packs.

2. **Approach**:
   - Build a DP table where `dp[i]` = minimum packs needed for `i` items
   - For each target amount, try all pack sizes and choose the optimal combination
   - Reconstruct the solution by backtracking through choices

3. **Time Complexity**: O(amount × pack_sizes)
4. **Space Complexity**: O(amount)

5. **Why DP**: 
   - Guarantees optimal solution (not greedy)
   - Handles complex scenarios where greedy fails
   - Efficient for the problem constraints (amounts up to 1M)

## Edge Case Example

**Input:**
- Pack Sizes: [23, 31, 53]
- Amount: 500,000

**Expected Output:**
```json
{
  "breakdown": {
    "53": 9429,
    "31": 7,
    "23": 2
  },
  "totalItems": 500000,
  "totalPacks": 9438,
  "overage": 0
}
```

This demonstrates the algorithm correctly handles large amounts and non-standard pack sizes.

## Makefile Commands

```bash
make up            # Start all services with Docker Compose
make down          # Stop services and remove volumes
make test          # Run all unit tests
make itest         # Run integration tests
make api-compile   # Compile the Go API binary
```
