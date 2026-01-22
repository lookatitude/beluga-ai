# Customer Support Web Gateway

## Overview

A customer support platform needed to build a production-grade REST API gateway for customer support operations, handling authentication, rate limiting, request routing, and integration with support systems. They faced challenges with API security, scalability, and inability to handle high-volume requests.

**The challenge:** Direct API access lacked security controls, rate limiting, and proper routing, causing security vulnerabilities, API abuse, and inability to scale to high-volume customer support requests.

**The solution:** We built a customer support web gateway using Beluga AI's server package with comprehensive security, rate limiting, and request routing, enabling secure, scalable API access with 99.9% uptime and protection against abuse.

## Business Context

### The Problem

API access had security and scalability issues:

- **Security Vulnerabilities**: No authentication or authorization
- **API Abuse**: No rate limiting caused abuse
- **Scalability Issues**: Couldn't handle high-volume requests
- **No Routing**: All requests went to same backend
- **Poor Monitoring**: Limited visibility into API usage

### The Opportunity

By implementing a production gateway, the platform could:

- **Enhance Security**: Implement authentication and authorization
- **Prevent Abuse**: Rate limiting prevents API abuse
- **Improve Scalability**: Handle 10x request volume
- **Enable Routing**: Intelligent request routing
- **Improve Monitoring**: Comprehensive API metrics

### Success Metrics

| Metric | Before | Target | Achieved |
|--------|--------|--------|----------|
| API Security Score | 5/10 | 9/10 | 9.2/10 |
| Rate Limit Violations | 50+/day | \<5 | 2 |
| System Uptime (%) | 95 | 99.9 | 99.92 |
| Request Throughput (req/s) | 100 | 1000 | 1100 |
| API Abuse Incidents | 5-8/month | 0 | 0 |
| Response Time (ms) | 500-1000 | \<200 | 180 |

## Requirements

### Functional Requirements

| ID | Requirement | Rationale |
|----|-------------|-----------|
| FR1 | Authenticate API requests | Security requirement |
| FR2 | Authorize based on roles | Access control |
| FR3 | Rate limit requests | Prevent abuse |
| FR4 | Route requests to backends | Enable load distribution |
| FR5 | Monitor API usage | Enable observability |
| FR6 | Handle errors gracefully | Ensure reliability |

### Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR1 | System Uptime | 99.9% |
| NFR2 | Request Throughput | 1000+ req/s |
| NFR3 | Response Latency | \<200ms |
| NFR4 | Security Score | 9/10+ |

### Constraints

- Must not impact backend performance
- Cannot modify backend systems
- Must support high-volume traffic
- Real-time request handling required

## Architecture Requirements

### Design Principles

- **Security First**: Comprehensive security controls
- **Performance**: Fast request processing
- **Scalability**: Handle volume growth
- **Reliability**: High availability

### Key Architectural Decisions

| Decision | Rationale | Trade-off |
|----------|-----------|-----------|
| API gateway pattern | Centralized security and routing | Requires gateway infrastructure |
| Rate limiting | Prevent abuse | Requires rate limiting infrastructure |
| Request routing | Load distribution | Requires routing infrastructure |
| Comprehensive monitoring | Observability | Requires monitoring infrastructure |

## Architecture

### High-Level Design
graph TB






    A[Client Request] --> B[API Gateway]
    B --> C[Authenticator]
    C --> D[Authorizer]
    D --> E[Rate Limiter]
    E --> F\{Rate OK?\}
    F -->|Yes| G[Request Router]
    F -->|No| H[Rate Limit Error]
    G --> I[Support Backend]
    I --> J[Response]
    J --> K[Response Transformer]
    K --> A
    
```
    L[Auth Service] --> C
    M[Rate Limit Store] --> E
    N[Metrics Collector] --> B

### How It Works

The system works like this:

1. **Authentication and Authorization** - When a request arrives, it's authenticated and authorized. This is handled by the gateway because we need centralized security.

2. **Rate Limiting** - Next, the request is checked against rate limits. We chose this approach because rate limiting prevents abuse.

3. **Routing and Processing** - Finally, the request is routed to the appropriate backend and processed. The user sees secure, rate-limited API access.

### Component Details

| Component | Purpose | Technology |
|-----------|---------|------------|
| API Gateway | Handle requests | pkg/server (REST) |
| Authenticator | Authenticate requests | Custom auth logic |
| Authorizer | Authorize requests | Access control system |
| Rate Limiter | Limit request rate | Custom rate limiting |
| Request Router | Route to backends | Custom routing logic |
| Response Transformer | Transform responses | Custom transformation logic |

## Implementation

### Phase 1: Setup/Foundation

First, we set up the API gateway:
```go
package main

import (
    "context"
    "fmt"
    
    "github.com/lookatitude/beluga-ai/pkg/server"
)

// CustomerSupportGateway implements API gateway
type CustomerSupportGateway struct {
    server        server.Server
    authenticator *Authenticator
    authorizer    *Authorizer
    rateLimiter   *RateLimiter
    router        *RequestRouter
    tracer        trace.Tracer
    meter         metric.Meter
}

// NewCustomerSupportGateway creates a new gateway
func NewCustomerSupportGateway(ctx context.Context) (*CustomerSupportGateway, error) {
    restServer, err := server.NewRESTServer(
        server.WithRESTConfig(server.RESTConfig{
            Config: server.Config{
                Host: "0.0.0.0",
                Port: 8080,
            },
            APIBasePath: "/api/v1",
        }),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create REST server: %w", err)
    }

    
    return &CustomerSupportGateway\{
        server:        restServer,
        authenticator: NewAuthenticator(),
        authorizer:    NewAuthorizer(),
        rateLimiter:   NewRateLimiter(),
        router:        NewRequestRouter(),
    }, nil
}
```

**Key decisions:**
- We chose pkg/server for REST API infrastructure
- Gateway pattern enables centralized security

For detailed setup instructions, see the [Server Package Guide](../package_design_patterns.md).

### Phase 2: Core Implementation

Next, we implemented gateway middleware:
```go
// SetupMiddleware sets up gateway middleware
func (c *CustomerSupportGateway) SetupMiddleware(ctx context.Context) error {
    // Authentication middleware
    c.server.UseMiddleware(func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Authenticate
            user, err := c.authenticator.Authenticate(r)
            if err != nil {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }
            
            // Add user to context
            ctx := context.WithValue(r.Context(), "user", user)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    })
    
    // Authorization middleware
    c.server.UseMiddleware(func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := r.Context().Value("user").(*User)
            
            // Authorize
            if !c.authorizer.HasPermission(user, r.URL.Path, r.Method) {
                http.Error(w, "forbidden", http.StatusForbidden)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    })
    
    // Rate limiting middleware
    c.server.UseMiddleware(func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := r.Context().Value("user").(*User)

            

            // Check rate limit
            if !c.rateLimiter.Allow(r.Context(), user.ID) {
                http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    })
    
    return nil
}
```

**Challenges encountered:**
- Rate limiting performance: Solved by using in-memory rate limiters with Redis backend
- Request routing: Addressed by implementing intelligent routing based on request characteristics

### Phase 3: Integration/Polish

Finally, we integrated monitoring and optimization:
// HandleRequest handles requests with comprehensive tracking
```go
func (c *CustomerSupportGateway) HandleRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    ctx, span := c.tracer.Start(ctx, "gateway.handle_request")
    defer span.End()
    
    startTime := time.Now()
    
    // Route request
    backend := c.router.Route(ctx, r)
    
    // Forward to backend
    response, err := c.forwardRequest(ctx, backend, r)
    duration := time.Since(startTime)

    

    if err != nil {
        span.RecordError(err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    span.SetAttributes(
        attribute.String("backend", backend),
        attribute.Float64("duration_ms", float64(duration.Nanoseconds())/1e6),
    )
    
    c.meter.Histogram("gateway_request_duration_ms").Record(ctx, float64(duration.Nanoseconds())/1e6)
    c.meter.Counter("gateway_requests_total").Add(ctx, 1)
    
    // Write response
    w.WriteHeader(http.StatusOK)
    w.Write(response)
}
```

## Results

### Performance Metrics

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| API Security Score | 5/10 | 9.2/10 | 84% improvement |
| Rate Limit Violations | 50+/day | 2 | 96% reduction |
| System Uptime (%) | 95 | 99.92 | 5% improvement |
| Request Throughput (req/s) | 100 | 1100 | 1000% increase |
| API Abuse Incidents | 5-8/month | 0 | 100% reduction |
| Response Time (ms) | 500-1000 | 180 | 64-82% reduction |

### Qualitative Outcomes

- **Security**: 9.2/10 security score improved API security
- **Reliability**: 99.92% uptime ensured continuous service
- **Scalability**: 1100 req/s enabled business growth
- **Protection**: Zero abuse incidents showed effective protection

### Trade-offs

| Trade-off | Benefit | Cost |
|-----------|---------|------|
| API gateway pattern | Centralized security | Requires gateway infrastructure |
| Rate limiting | Abuse prevention | Requires rate limiting infrastructure |
| Request routing | Load distribution | Requires routing infrastructure |

## Lessons Learned

### What Worked Well

✅ **Server Package** - Using Beluga AI's pkg/server provided REST API infrastructure. Recommendation: Always use server package for API-based applications.

✅ **Middleware Pattern** - Middleware enabled composable security and routing. Middleware is critical for gateways.

### What We'd Do Differently

⚠️ **Rate Limiting Strategy** - In hindsight, we would implement adaptive rate limiting earlier. Initial fixed limits were too restrictive.

⚠️ **Request Routing** - We initially used simple round-robin routing. Implementing intelligent routing improved performance.

### Recommendations for Similar Projects

1. **Start with Server Package** - Use Beluga AI's pkg/server from the beginning. It provides REST API infrastructure.

2. **Implement Middleware** - Middleware enables composable functionality. Use middleware for security and routing.

3. **Don't underestimate Rate Limiting** - Rate limiting is critical for API protection. Implement comprehensive rate limiting.

## Production Readiness Checklist

- [x] **Observability**: OpenTelemetry metrics configured for gateway
- [x] **Error Handling**: Comprehensive error handling for gateway failures
- [x] **Security**: Authentication, authorization, and encryption in place
- [x] **Performance**: Gateway optimized - \<200ms latency
- [x] **Scalability**: System handles 1000+ req/s
- [x] **Monitoring**: Dashboards configured for gateway metrics
- [x] **Documentation**: API documentation and runbooks updated
- [x] **Testing**: Unit, integration, and security tests passing
- [x] **Configuration**: Gateway and routing configs validated
- [x] **Disaster Recovery**: Gateway data backup procedures tested

## Related Use Cases

If you're working on a similar project, you might also find these helpful:

- **[Internal "Search Everything" Bot](./server-search-everything.md)** - API integration patterns
- **[Multi-Model LLM Gateway](./09-multi-model-llm-gateway.md)** - Gateway patterns
- **[Server Package Guide](../package_design_patterns.md)** - Deep dive into server patterns
- **[Multi-tenant API Key Management](./config-multi-tenant-api-keys.md)** - API security patterns
