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
- Calculation results can be cached (infrastructure ready, not currently implemented).
- **Why**: Reduces database load and improves API response times, especially for frequently accessed data.

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

# Services will be available at:
# - Frontend: http://localhost:5173
# - API: http://localhost:8080/api/v1
# - PostgreSQL: localhost:5432
# - Redis: localhost:6379
```

The database will automatically:
- Create the schema on first startup
- Seed initial pack sizes: [250, 500, 1000, 2000, 5000]

### Running Tests in Docker

Run unit tests in the Docker environment:

```bash
# Run all unit tests
docker compose exec api go test ./...

# Run integration tests
docker compose exec api go test -tags=integration ./internal/integration/...

# Or use Makefile commands
make test-docker    # Run unit tests in Docker
make itest-docker   # Run integration tests in Docker
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
```
http://localhost:8080/api/v1
```

### Endpoints

#### GET `/packs`
Get current active pack sizes.

**Response:**
```json
{
  "sizes": [250, 500, 1000, 2000, 5000]
}
```

#### PUT `/packs`
Replace all pack sizes with a new set.

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

**Example:** `DELETE /packs/250`

**Response:**
```json
{
  "sizes": [500, 1000, 2000, 5000]
}
```

#### POST `/calculate`
Calculate optimal pack distribution.

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

**Response:** `200 OK`

## Testing

### Unit Tests

Run all unit tests:
```bash
cd backend
go test ./...
```

Or in Docker:
```bash
docker compose exec api go test ./...
# Or use: make test-docker
```

**Test Coverage:**
- **HTTP Handlers** (`internal/adapters/http/handlers_test.go`):
  - Pack size CRUD operations
  - Calculation endpoint
  - Input validation
  - Error handling
  - Edge cases (deleting last pack, invalid inputs)

- **Calculator Service** (`internal/app/calculator/service_test.go`):
  - Standard pack size scenarios
  - Edge cases (500,000 items with [23, 31, 53])
  - Boundary conditions
  - Optimization rules verification
  - Invalid input handling

### Integration Tests

Run integration tests (requires Docker):
```bash
cd backend
go test -tags=integration ./internal/integration/...
```

Or in Docker:
```bash
docker compose exec api go test -tags=integration ./internal/integration/...
# Or use: make itest-docker
```

**Test Coverage:**
- PostgreSQL repository operations
- Database connectivity
- Data persistence

### Test Scenarios

See `TEST_SCENARIOS.md` for comprehensive test cases covering:
- Standard scenarios
- Edge cases
- Boundary conditions
- Optimization rules
- Invalid inputs
- Performance considerations

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
make test          # Run all unit tests (requires Go installed locally)
make itest         # Run integration tests (requires Go installed locally)
make test-docker   # Run unit tests in Docker container
make itest-docker  # Run integration tests in Docker container
make api-compile   # Compile the Go API binary
```
