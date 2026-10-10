package security

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type rateLimitEntry struct {
	started time.Time
	count   int
}

type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string]rateLimitEntry
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, entries: make(map[string]rateLimitEntry)}
}

func (r *RateLimiter) Allow(key string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.entries[key]
	if !ok || now.Sub(entry.started) >= r.window {
		r.entries[key] = rateLimitEntry{started: now, count: 1}
		return true
	}
	if entry.count >= r.limit {
		return false
	}
	entry.count++
	r.entries[key] = entry
	return true
}

func (r *RateLimiter) UnaryInterceptor(now func() time.Time) grpc.UnaryServerInterceptor {
	if now == nil {
		now = time.Now
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !isSensitiveMethod(info.FullMethod) {
			return handler(ctx, req)
		}
		key := remoteAddress(ctx) + ":" + info.FullMethod
		if !r.Allow(key, now()) {
			return nil, status.Error(codes.ResourceExhausted, "too many authentication attempts")
		}
		return handler(ctx, req)
	}
}

func isSensitiveMethod(method string) bool {
	return strings.HasSuffix(method, "/Login") || strings.HasSuffix(method, "/Register")
}

func remoteAddress(ctx context.Context) string {
	value, ok := peer.FromContext(ctx)
	if !ok || value.Addr == nil {
		return "unknown"
	}
	if host, _, err := net.SplitHostPort(value.Addr.String()); err == nil {
		return host
	}
	return value.Addr.String()
}
