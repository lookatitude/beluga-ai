---
title: Customer Support API Gateway
description: "Build a production REST API gateway with authentication, rate limiting, and AI-specific controls. Protect against cost and data abuse."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "API gateway AI, support API, rate limiting, authentication middleware, REST gateway, Beluga AI, Go, enterprise API"
---

Exposing AI-powered customer support as an API creates a surface area that is both expensive and vulnerable. Each request triggers LLM inference costing $0.01-0.10, meaning an unauthenticated endpoint can generate thousands of dollars in charges from a single attacker running automated requests. Beyond cost, support APIs handle sensitive customer data (account details, conversation history, PII) that requires role-based access — an agent should see their assigned tickets but not all customer records, while an admin needs broader access. Without proper gateway controls, these APIs face credential stuffing, rate limit abuse, and unauthorized data access.

Traditional API gateway products (Kong, AWS API Gateway) provide generic HTTP controls but lack AI-specific awareness: they cannot enforce per-user token budgets, route requests based on LLM model requirements, or integrate safety guards on request payloads.

## Solution Architecture

Beluga AI's `server/` package provides REST API infrastructure with middleware composition following the standard `func(http.Handler) http.Handler` pattern. The middleware approach is composable — authentication, authorization, rate limiting, and observability are independent layers that stack in a defined order. This matters because each concern has different failure semantics: authentication failures return 401, authorization failures return 403, and rate limit failures return 429. Separating them ensures correct HTTP semantics and allows each layer to be tested, monitored, and configured independently.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│    Client    │───▶│     API      │───▶│  Authenticator│
│   Request    │    │   Gateway    │    │              │
└──────────────┘    └──────────────┘    └──────┬───────┘
                                               │
                                               ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Backend    │◀───│   Request    │◀───│     Rate     │
│   Response   │    │    Router    │    │   Limiter    │
└──────────────┘    └──────────────┘    └──────┬───────┘
                          ▲                     │
                          │                     ▼
                    ┌─────┴────────┐    ┌──────────────┐
                    │   Support    │    │  Authorizer  │
                    │   Backend    │    │              │
                    └──────────────┘    └──────────────┘
```

## Gateway Setup

Create a REST server with middleware for security and routing:

```go
package main

import (
    "context"
    "fmt"
    "net/http"

    "github.com/lookatitude/beluga-ai/server"
    "go.opentelemetry.io/otel/trace"
    "go.opentelemetry.io/otel/metric"
)

type CustomerSupportGateway struct {
    server        *server.RESTServer
    authenticator *Authenticator
    authorizer    *Authorizer
    rateLimiter   *RateLimiter
    router        *RequestRouter
    tracer        trace.Tracer
    meter         metric.Meter
}

func NewCustomerSupportGateway(ctx context.Context) (*CustomerSupportGateway, error) {
    restServer, err := server.NewRESTServer(
        server.WithHost("0.0.0.0"),
        server.WithPort(8080),
        server.WithBasePath("/api/v1"),
    )
    if err != nil {
        return nil, fmt.Errorf("create REST server: %w", err)
    }

    return &CustomerSupportGateway{
        server:        restServer,
        authenticator: NewAuthenticator(),
        authorizer:    NewAuthorizer(),
        rateLimiter:   NewRateLimiter(),
        router:        NewRequestRouter(),
    }, nil
}
```

## Authentication Middleware

Authentication runs first in the middleware chain and attaches user identity to the request context. Subsequent middleware (authorization, rate limiting) depends on this identity, so the ordering is critical — authorization without authentication would fail, and rate limiting without user identity could only limit by IP address:

```go
package main

import (
    "context"
    "net/http"
)

type User struct {
    ID       string
    Username string
    Roles    []string
}

type Authenticator struct {
    apiKeys map[string]*User
}

func NewAuthenticator() *Authenticator {
    return &Authenticator{
        apiKeys: make(map[string]*User),
    }
}

func (a *Authenticator) Authenticate(r *http.Request) (*User, error) {
    apiKey := r.Header.Get("X-API-Key")
    if apiKey == "" {
        return nil, fmt.Errorf("missing API key")
    }

    user, ok := a.apiKeys[apiKey]
    if !ok {
        return nil, fmt.Errorf("invalid API key")
    }

    return user, nil
}

func (g *CustomerSupportGateway) AuthenticationMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user, err := g.authenticator.Authenticate(r)
        if err != nil {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }

        // Attach user to context
        ctx := context.WithValue(r.Context(), "user", user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Authorization Middleware

Enforce role-based access control:

```go
package main

import (
    "net/http"
)

type Authorizer struct {
    permissions map[string]map[string][]string // path -> method -> allowed roles
}

func NewAuthorizer() *Authorizer {
    return &Authorizer{
        permissions: map[string]map[string][]string{
            "/api/v1/tickets": {
                "GET":  []string{"agent", "admin"},
                "POST": []string{"agent", "admin"},
            },
            "/api/v1/users": {
                "GET":  []string{"admin"},
                "POST": []string{"admin"},
            },
        },
    }
}

func (a *Authorizer) HasPermission(user *User, path, method string) bool {
    methods, ok := a.permissions[path]
    if !ok {
        return false
    }

    allowedRoles, ok := methods[method]
    if !ok {
        return false
    }

    for _, role := range user.Roles {
        for _, allowed := range allowedRoles {
            if role == allowed {
                return true
            }
        }
    }

    return false
}

func (g *CustomerSupportGateway) AuthorizationMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := r.Context().Value("user").(*User)

        if !g.authorizer.HasPermission(user, r.URL.Path, r.Method) {
            http.Error(w, "forbidden", http.StatusForbidden)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

## Rate Limiting

Prevent API abuse with per-user rate limits:

```go
package main

import (
    "context"
    "net/http"
    "sync"
    "time"
)

type RateLimiter struct {
    mu      sync.RWMutex
    limits  map[string]*RateLimit // userID -> rate limit state
    perUser int                   // requests per minute
}

type RateLimit struct {
    Count      int
    ResetTime  time.Time
}

func NewRateLimiter() *RateLimiter {
    return &RateLimiter{
        limits:  make(map[string]*RateLimit),
        perUser: 100, // 100 requests per minute
    }
}

func (r *RateLimiter) Allow(ctx context.Context, userID string) bool {
    r.mu.Lock()
    defer r.mu.Unlock()

    now := time.Now()
    limit, ok := r.limits[userID]

    if !ok || now.After(limit.ResetTime) {
        // New user or reset period expired
        r.limits[userID] = &RateLimit{
            Count:     1,
            ResetTime: now.Add(time.Minute),
        }
        return true
    }

    if limit.Count >= r.perUser {
        return false
    }

    limit.Count++
    return true
}

func (g *CustomerSupportGateway) RateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := r.Context().Value("user").(*User)

        if !g.rateLimiter.Allow(r.Context(), user.ID) {
            http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

## Request Routing

Route requests to backend services:

```go
package main

import (
    "context"
    "io"
    "net/http"
)

type RequestRouter struct {
    backends map[string]string // path pattern -> backend URL
    client   *http.Client
}

func NewRequestRouter() *RequestRouter {
    return &RequestRouter{
        backends: map[string]string{
            "/api/v1/tickets": "http://tickets-service:8080",
            "/api/v1/users":   "http://users-service:8080",
        },
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (r *RequestRouter) Route(ctx context.Context, req *http.Request) string {
    for pattern, backend := range r.backends {
        if strings.HasPrefix(req.URL.Path, pattern) {
            return backend
        }
    }
    return ""
}

func (g *CustomerSupportGateway) HandleRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    ctx, span := g.tracer.Start(ctx, "gateway.handle_request")
    defer span.End()

    backend := g.router.Route(ctx, r)
    if backend == "" {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }

    // Forward request to backend
    backendURL := backend + r.URL.Path
    backendReq, err := http.NewRequestWithContext(ctx, r.Method, backendURL, r.Body)
    if err != nil {
        http.Error(w, "internal server error", http.StatusInternalServerError)
        return
    }

    // Copy headers
    for key, values := range r.Header {
        for _, value := range values {
            backendReq.Header.Add(key, value)
        }
    }

    resp, err := g.router.client.Do(backendReq)
    if err != nil {
        http.Error(w, "backend error", http.StatusBadGateway)
        return
    }
    defer resp.Body.Close()

    // Copy response
    for key, values := range resp.Header {
        for _, value := range values {
            w.Header().Add(key, value)
        }
    }
    w.WriteHeader(resp.StatusCode)
    io.Copy(w, resp.Body)
}
```

## Production Considerations

### Observability

Track gateway metrics and request latency:

```go
import (
    "github.com/lookatitude/beluga-ai/o11y"
    "go.opentelemetry.io/otel/attribute"
)

func (g *CustomerSupportGateway) InstrumentedHandle(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    ctx, span := g.tracer.Start(ctx, "gateway.request")
    defer span.End()

    start := time.Now()

    span.SetAttributes(
        attribute.String("http.method", r.Method),
        attribute.String("http.path", r.URL.Path),
        attribute.String("user.id", r.Context().Value("user").(*User).ID),
    )

    g.HandleRequest(ctx, w, r)

    duration := time.Since(start)
    g.meter.RecordHistogram(ctx, "gateway.request.duration", duration.Milliseconds())
    g.meter.IncrementCounter(ctx, "gateway.requests.total")
}
```

### Resilience

Add circuit breaker for backend failures:

```go
import "github.com/lookatitude/beluga-ai/resilience"

breaker := resilience.NewCircuitBreaker(
    resilience.WithFailureThreshold(5),
    resilience.WithTimeout(30 * time.Second),
    resilience.WithResetTimeout(60 * time.Second),
)

resp, err := breaker.Execute(ctx, func(ctx context.Context) (*http.Response, error) {
    return g.router.client.Do(backendReq)
})
```

### Security

- Use TLS for all gateway connections with certificate pinning
- Implement token-based authentication (JWT) with short expiration times
- Log all authentication and authorization failures for security monitoring
- Use Beluga AI's `guard/` pipeline to validate request payloads
- Rotate API keys regularly and revoke compromised keys immediately

## Related Resources

- [Server Package Guide](/guides/server-patterns/) for REST API patterns
- [Middleware Guide](/guides/middleware/) for composable middleware
- [Observability Guide](/guides/observability/) for API monitoring
- [Search Everything Gateway](/use-cases/search-everything/) for API integration patterns
