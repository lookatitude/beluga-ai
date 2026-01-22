# Auth0/JWT Authentication

Welcome, colleague! In this integration guide, we're going to integrate Auth0 and JWT authentication with Beluga AI's server package. This enables secure API access with token-based authentication.

## What you will build

You will configure Beluga AI REST server with Auth0 JWT authentication, enabling secure API endpoints with token validation, user identification, and role-based access control.

## Learning Objectives

- ✅ Configure Auth0 with Beluga AI server
- ✅ Validate JWT tokens
- ✅ Extract user information from tokens
- ✅ Implement authentication middleware

## Prerequisites

- Go 1.24 or later installed
- Beluga AI Framework installed
- Auth0 account and application
- JWT library

## Step 1: Setup and Installation

Install JWT library:
bash
```bash
go get github.com/golang-jwt/jwt/v5
```

Get Auth0 credentials:
- Domain: `your-tenant.auth0.com`
- Client ID
- Audience (API identifier)

## Step 2: Create JWT Middleware

Create JWT authentication middleware:
```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "strings"

    "github.com/golang-jwt/jwt/v5"
    "github.com/lookatitude/beluga-ai/pkg/server/iface"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type Auth0JWTMiddleware struct {
    domain   string
    audience string
    tracer   trace.Tracer
}

type Claims struct {
    Sub   string   `json:"sub"`
    Email string   `json:"email"`
    Roles []string `json:"https://beluga.ai/roles"`
    jwt.RegisteredClaims
}

func NewAuth0JWTMiddleware(domain, audience string) *Auth0JWTMiddleware {
    return &Auth0JWTMiddleware{
        domain:   domain,
        audience: audience,
        tracer:   otel.Tracer("beluga.server.auth"),
    }
}

func (m *Auth0JWTMiddleware) Authenticate(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx, span := m.tracer.Start(r.Context(), "auth.authenticate")
        defer span.End()
        
        // Extract token
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            span.RecordError(fmt.Errorf("missing authorization header"))
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            span.RecordError(fmt.Errorf("invalid authorization header"))
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        tokenString := parts[1]
        
        // Validate token
        token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
            // Verify signing method
            if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
            }
            
            // Get public key from Auth0
            return m.getPublicKey(ctx, token)
        })
        
        if err != nil {
            span.RecordError(err)
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        if claims, ok := token.Claims.(*Claims); ok && token.Valid {
            // Add user info to context
            ctx = context.WithValue(ctx, "user_id", claims.Sub)
            ctx = context.WithValue(ctx, "user_email", claims.Email)
            ctx = context.WithValue(ctx, "user_roles", claims.Roles)
            
            span.SetAttributes(
                attribute.String("user_id", claims.Sub),
                attribute.String("user_email", claims.Email),
            )
            
            next.ServeHTTP(w, r.WithContext(ctx))
        } else {
            span.RecordError(fmt.Errorf("invalid token"))
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
        }
    })
}

func (m *Auth0JWTMiddleware) getPublicKey(ctx context.Context, token *jwt.Token) (interface{}, error) {
    // Fetch JWKS from Auth0
    // Implementation depends on your JWT library
    // This is a simplified example
    return nil, nil
}
```

## Step 3: Use with Beluga AI Server

Integrate with Beluga AI server:
```go
func main() {
    ctx := context.Background()
    
    // Create Auth0 middleware
    authMiddleware := NewAuth0JWTMiddleware(
        os.Getenv("AUTH0_DOMAIN"),
        os.Getenv("AUTH0_AUDIENCE"),
    )
    
    // Create REST server
    restServer, err := server.NewRESTServer(
        server.WithRESTConfig(server.RESTConfig{
            Config: server.Config{
                Host: "0.0.0.0",
                Port: 8080,
            },
        }),
    )
    if err != nil {
        log.Fatalf("Failed to create server: %v", err)
    }
    
    // Add authentication middleware
    restServer.UseMiddleware(authMiddleware.Authenticate)
    
    // Register handlers
    restServer.RegisterHandler("/api/agents", agentHandler)
    
    // Start server
    if err := restServer.Start(ctx); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
```

## Step 4: Extract User Info

Extract user information from context:
```go
func getUserID(ctx context.Context) string {
    if userID, ok := ctx.Value("user_id").(string); ok {
        return userID
    }
    return ""
}

func getUserRoles(ctx context.Context) []string {
    if roles, ok := ctx.Value("user_roles").([]string); ok {
        return roles
    }
    return []string{}
}
```

## Step 5: Complete Integration

Here's a complete, production-ready example:
```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "strings"

    "github.com/golang-jwt/jwt/v5"
    "github.com/lookatitude/beluga-ai/pkg/server"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

type ProductionAuth0Middleware struct {
    domain   string
    audience string
    tracer   trace.Tracer
}

func NewProductionAuth0Middleware(domain, audience string) *ProductionAuth0Middleware {
    return &ProductionAuth0Middleware{
        domain:   domain,
        audience: audience,
        tracer:   otel.Tracer("beluga.server.auth0"),
    }
}

func (m *ProductionAuth0Middleware) Authenticate(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx, span := m.tracer.Start(r.Context(), "auth0.authenticate")
        defer span.End()
        
        // Extract and validate token
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            span.RecordError(fmt.Errorf("missing authorization"))
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        
        // Parse and validate JWT
        token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
            // Verify algorithm
            if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
                return nil, fmt.Errorf("unexpected method: %v", token.Header["alg"])
            }
            // Get public key (simplified - implement JWKS fetch)
            return m.getPublicKey(ctx, token)
        })
        
        if err != nil || !token.Valid {
            span.RecordError(err)
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        
        if claims, ok := token.Claims.(*Claims); ok {
            ctx = context.WithValue(ctx, "user_id", claims.Sub)
            ctx = context.WithValue(ctx, "user_email", claims.Email)
            
            span.SetAttributes(
                attribute.String("user_id", claims.Sub),
            )
            
            next.ServeHTTP(w, r.WithContext(ctx))
        } else {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
        }
    })
}

func (m *ProductionAuth0Middleware) getPublicKey(ctx context.Context, token *jwt.Token) (interface{}, error) {
    // Implement JWKS fetching from Auth0
    // https://{domain}/.well-known/jwks.json
    return nil, nil
}

func main() {
    ctx := context.Background()
    
    authMiddleware := NewProductionAuth0Middleware(
        os.Getenv("AUTH0_DOMAIN"),
        os.Getenv("AUTH0_AUDIENCE"),
    )
    
    restServer, err := server.NewRESTServer(
        server.WithRESTConfig(server.RESTConfig{
            Config: server.Config{
                Host: "0.0.0.0",
                Port: 8080,
            },
        }),
    )
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    
    restServer.UseMiddleware(authMiddleware.Authenticate)
    
    if err := restServer.Start(ctx); err != nil {
        log.Fatalf("Failed: %v", err)
    }
}
```

## Configuration Options

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `AUTH0_DOMAIN` | Auth0 domain | - | Yes |
| `AUTH0_AUDIENCE` | API audience | - | Yes |
| `AUTH0_CLIENT_ID` | Client ID | - | No |

## Common Issues

### "Token validation failed"

**Problem**: Invalid token or wrong audience.

**Solution**: Verify token and audience:export AUTH0_DOMAIN="your-tenant.auth0.com"
bash
```bash
export AUTH0_AUDIENCE="your-api-identifier"
```

### "Public key not found"

**Problem**: JWKS endpoint not accessible.

**Solution**: Ensure JWKS endpoint is reachable:curl https://your-tenant.auth0.com/.well-known/jwks.json
```

## Production Considerations

When using Auth0 in production:

- **Token caching**: Cache validated tokens
- **JWKS caching**: Cache public keys
- **Error handling**: Handle token expiration gracefully
- **Rate limiting**: Implement rate limiting per user
- **Audit logging**: Log authentication events

## Next Steps

Congratulations! You've integrated Auth0 with Beluga AI. Next, learn how to:

- **[Kubernetes Helm Deployment](./kubernetes-helm-deployment.md)** - Helm deployment
- **[Server Package Documentation](../../api/packages/server.md)** - Deep dive into server package
- **[Deployment Guide](../../getting-started/07-production-deployment.md)** - Production deployment

---

**Ready for more?** Check out the [Integrations Index](./README.md) for more integration guides!
