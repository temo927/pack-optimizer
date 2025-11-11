FROM golang:1.23-alpine AS build
WORKDIR /app
# Pre-fetch modules for better layer caching
COPY go.mod .
RUN apk add --no-cache git && go mod download
COPY . .
RUN go build -o /bin/api ./cmd/api

FROM alpine:3.20
ENV HTTP_PORT=8080
EXPOSE 8080
COPY --from=build /bin/api /bin/api
ENTRYPOINT ["/bin/api"]


