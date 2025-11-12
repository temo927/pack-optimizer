FROM golang:1.23-alpine AS build
WORKDIR /app
# Pre-fetch modules for better layer caching
COPY go.mod .
RUN apk add --no-cache git && go mod download
COPY . .
RUN go build -o /bin/api ./cmd/api

# Final runtime stage - includes Go for testing
FROM golang:1.23-alpine
ENV HTTP_PORT=8080
EXPOSE 8080
WORKDIR /app
# Copy the built binary
COPY --from=build /bin/api /bin/api
# Copy source code and go.mod/go.sum for tests
COPY --from=build /app /app
ENTRYPOINT ["/bin/api"]


