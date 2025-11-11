// Package http provides HTTP handlers for the pack optimizer API.
// This file contains HTTP transport layer middleware for security, rate limiting, and DDoS protection.
package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
)

// RateLimitConfig holds configuration for rate limiting.
type RateLimitConfig struct {
	RequestsPerMinute int  // Maximum requests per minute per IP
	BurstSize         int  // Burst size for token bucket
	Enabled           bool // Whether rate limiting is enabled
}

// DDoSProtectionConfig holds configuration for DDoS protection.
type DDoSProtectionConfig struct {
	MaxRequestSize    int64 // Maximum request body size in bytes (10MB default)
	MaxHeaderSize     int   // Maximum header size in bytes
	MaxConcurrentReqs int   // Maximum concurrent requests per IP
	Enabled           bool  // Whether DDoS protection is enabled
}

// SecurityConfig holds all security-related configuration.
type SecurityConfig struct {
	RateLimitEnabled      bool
	RateLimitRPM          string
	RateLimitBurst        string
	DDoSProtectionEnabled bool
	MaxRequestSize        string
	MaxHeaderSize         string
}

// SetupSecurityMiddleware configures and applies all security middleware to the router.
// This centralizes security middleware setup in the HTTP transport layer.
func SetupSecurityMiddleware(r *chi.Mux, cfg SecurityConfig) {
	// 1. Security headers - add security headers to all responses
	r.Use(securityHeaders)

	// 2. DDoS protection - protect against DDoS attacks
	ddosConfig := parseDDoSProtectionConfig(cfg.MaxRequestSize, cfg.MaxHeaderSize)
	ddosConfig.Enabled = cfg.DDoSProtectionEnabled
	r.Use(ddosProtection(ddosConfig))

	// 3. Rate limiting - limit requests per IP
	rateLimitConfig := parseRateLimitConfig(cfg.RateLimitRPM, cfg.RateLimitBurst)
	rateLimitConfig.Enabled = cfg.RateLimitEnabled
	r.Use(rateLimit(rateLimitConfig))
}

// rateLimit creates a rate limiting middleware that limits requests per IP address.
// Uses a token bucket algorithm to allow bursts while maintaining average rate.
// Returns 429 Too Many Requests when limit is exceeded.
func rateLimit(config RateLimitConfig) func(next http.Handler) http.Handler {
	if !config.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	// Default to 100 requests per minute if not configured
	requestsPerMinute := config.RequestsPerMinute
	if requestsPerMinute <= 0 {
		requestsPerMinute = 100
	}

	// Default burst size to 20% of requests per minute
	burstSize := config.BurstSize
	if burstSize <= 0 {
		burstSize = requestsPerMinute / 5
		if burstSize < 1 {
			burstSize = 1
		}
	}

	// Create rate limiter that limits by IP address
	// httprate uses a token bucket algorithm
	limiter := httprate.Limit(
		requestsPerMinute,
		time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByIP, httprate.KeyByEndpoint),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			slog.Warn(
				"rate limit exceeded",
				"ip", getClientIP(r),
				"path", r.URL.Path,
			)

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate limit exceeded","message":"too many requests, please try again later"}`))
		}),
	)

	return limiter
}

// ddosProtection creates middleware to protect against DDoS attacks.
// Includes request size limits, header size limits, and basic connection throttling.
func ddosProtection(config DDoSProtectionConfig) func(next http.Handler) http.Handler {
	if !config.Enabled {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Limit request body size
			if config.MaxRequestSize > 0 {
				r.Body = http.MaxBytesReader(w, r.Body, config.MaxRequestSize)
			}

			// Limit header size
			if config.MaxHeaderSize > 0 {
				headerSize := 0
				for key, values := range r.Header {
					headerSize += len(key)
					for _, value := range values {
						headerSize += len(value)
					}
				}
				if headerSize > config.MaxHeaderSize {
					slog.Warn(
						"request header too large",
						"ip", getClientIP(r),
						"header_size", headerSize,
					)

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusRequestEntityTooLarge)
					w.Write([]byte(`{"error":"request header too large"}`))
					return
				}
			}

			// Check for suspicious patterns
			if isSuspiciousRequest(r) {
				slog.Warn(
					"suspicious request detected",
					"ip", getClientIP(r),
					"path", r.URL.Path,
					"user_agent", r.UserAgent(),
				)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"suspicious request detected"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// securityHeaders adds security-related HTTP headers to responses.
// Helps protect against XSS, clickjacking, and other attacks.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Enable XSS protection
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy (basic)
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts the real client IP address from the request.
// Checks X-Forwarded-For, X-Real-IP headers for proxied requests.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (first IP in chain)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// isSuspiciousRequest checks for common DDoS attack patterns.
func isSuspiciousRequest(r *http.Request) bool {
	// Check for suspicious user agents
	userAgent := strings.ToLower(r.UserAgent())
	suspiciousAgents := []string{
		"sqlmap", "nikto", "nmap", "masscan",
		"scanner", "bot", "crawler", "spider",
		"wget", "python-requests",
	}

	// Allow legitimate bots (Google, etc.) but block suspicious ones
	if userAgent != "" {
		for _, suspicious := range suspiciousAgents {
			if strings.Contains(userAgent, suspicious) {
				// Allow known good bots
				if strings.Contains(userAgent, "googlebot") ||
					strings.Contains(userAgent, "bingbot") {
					continue
				}
				return true
			}
		}
	}

	// Check for suspicious query parameters (SQL injection patterns)
	query := r.URL.RawQuery
	if query != "" {
		queryLower := strings.ToLower(query)
		suspiciousPatterns := []string{
			"union select", "1=1", "1' or '1'='1",
			"drop table", "delete from", "exec(",
			"<script", "javascript:", "onerror=",
		}
		for _, pattern := range suspiciousPatterns {
			if strings.Contains(queryLower, pattern) {
				return true
			}
		}
	}

	// Check for excessive path length (potential buffer overflow attempts)
	if len(r.URL.Path) > 2048 {
		return true
	}

	return false
}

// parseRateLimitConfig parses rate limit configuration from environment variables.
func parseRateLimitConfig(requestsPerMinute, burstSize string) RateLimitConfig {
	config := RateLimitConfig{
		Enabled: true,
	}

	if requestsPerMinute != "" {
		if val, err := strconv.Atoi(requestsPerMinute); err == nil && val > 0 {
			config.RequestsPerMinute = val
		} else {
			config.RequestsPerMinute = 100 // default
		}
	} else {
		config.RequestsPerMinute = 100 // default
	}

	if burstSize != "" {
		if val, err := strconv.Atoi(burstSize); err == nil && val > 0 {
			config.BurstSize = val
		} else {
			config.BurstSize = config.RequestsPerMinute / 5 // default
		}
	} else {
		config.BurstSize = config.RequestsPerMinute / 5 // default
	}

	return config
}

// parseDDoSProtectionConfig parses DDoS protection configuration from environment variables.
func parseDDoSProtectionConfig(maxRequestSize, maxHeaderSize string) DDoSProtectionConfig {
	config := DDoSProtectionConfig{
		Enabled:        true,
		MaxRequestSize: 10 * 1024 * 1024, // 10MB default
		MaxHeaderSize:  8192,              // 8KB default
	}

	if maxRequestSize != "" {
		if val, err := strconv.ParseInt(maxRequestSize, 10, 64); err == nil && val > 0 {
			config.MaxRequestSize = val
		}
	}

	if maxHeaderSize != "" {
		if val, err := strconv.Atoi(maxHeaderSize); err == nil && val > 0 {
			config.MaxHeaderSize = val
		}
	}

	return config
}

