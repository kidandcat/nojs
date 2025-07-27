package nojs

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Logger middleware logs HTTP requests
func Logger() Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			start := time.Now()
			
			// Call the next handler
			err := next(ctx)
			
			// Log the request
			duration := time.Since(start)
			status := http.StatusOK
			if err != nil {
				if httpErr, ok := err.(*HTTPError); ok {
					status = httpErr.Code
				} else {
					status = http.StatusInternalServerError
				}
			}
			
			log.Printf("%s %s %d %v", ctx.Request.Method, ctx.Request.URL.Path, status, duration)
			
			return err
		}
	}
}

// Recovery middleware recovers from panics
func Recovery() Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic recovered: %v", r)
					err = NewHTTPError(http.StatusInternalServerError, "Internal Server Error")
				}
			}()
			
			return next(ctx)
		}
	}
}

// CORS middleware adds CORS headers
func CORS(allowedOrigins []string) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			origin := ctx.Request.Header.Get("Origin")
			
			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}
			
			if allowed {
				ctx.ResponseWriter.Header().Set("Access-Control-Allow-Origin", origin)
				ctx.ResponseWriter.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				ctx.ResponseWriter.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			}
			
			// Handle preflight requests
			if ctx.Request.Method == "OPTIONS" {
				ctx.ResponseWriter.WriteHeader(http.StatusNoContent)
				return nil
			}
			
			return next(ctx)
		}
	}
}

// NoCache middleware prevents caching
func NoCache() Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			ctx.ResponseWriter.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			ctx.ResponseWriter.Header().Set("Pragma", "no-cache")
			ctx.ResponseWriter.Header().Set("Expires", "0")
			
			return next(ctx)
		}
	}
}

// RequireMethod ensures the request uses specific HTTP methods
func RequireMethod(methods ...string) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			for _, method := range methods {
				if ctx.Method() == method {
					return next(ctx)
				}
			}
			
			return NewHTTPError(http.StatusMethodNotAllowed, "Method Not Allowed")
		}
	}
}

// RateLimit implements basic rate limiting
func RateLimit(requests int, duration time.Duration) Middleware {
	type visitor struct {
		count    int
		lastSeen time.Time
	}
	
	visitors := make(map[string]*visitor)
	
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			ip := ctx.Request.RemoteAddr
			now := time.Now()
			
			v, exists := visitors[ip]
			if !exists {
				visitors[ip] = &visitor{count: 1, lastSeen: now}
			} else {
				if now.Sub(v.lastSeen) > duration {
					v.count = 1
					v.lastSeen = now
				} else {
					v.count++
					if v.count > requests {
						return NewHTTPError(http.StatusTooManyRequests, "Rate limit exceeded")
					}
				}
			}
			
			// Clean up old entries periodically
			if len(visitors) > 1000 {
				for k, v := range visitors {
					if now.Sub(v.lastSeen) > duration {
						delete(visitors, k)
					}
				}
			}
			
			return next(ctx)
		}
	}
}

// BasicAuth implements HTTP Basic Authentication
func BasicAuth(realm string, users map[string]string) Middleware {
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			username, password, ok := ctx.Request.BasicAuth()
			if !ok {
				ctx.ResponseWriter.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
				return NewHTTPError(http.StatusUnauthorized, "Unauthorized")
			}
			
			expectedPassword, userExists := users[username]
			if !userExists || password != expectedPassword {
				ctx.ResponseWriter.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
				return NewHTTPError(http.StatusUnauthorized, "Unauthorized")
			}
			
			return next(ctx)
		}
	}
}

// Session represents a simple session store
type Session struct {
	store map[string]interface{}
}

// Get retrieves a value from the session
func (s *Session) Get(key string) interface{} {
	return s.store[key]
}

// Set stores a value in the session
func (s *Session) Set(key string, value interface{}) {
	s.store[key] = value
}

// SessionManager provides simple cookie-based sessions
func SessionManager(secret string) Middleware {
	// This is a simplified implementation
	// In production, use proper session management with encryption
	sessions := make(map[string]*Session)
	
	return func(next Handler) Handler {
		return func(ctx *Context) error {
			// Get or create session ID
			cookie, err := ctx.Request.Cookie("session")
			sessionID := ""
			
			if err == nil {
				sessionID = cookie.Value
			} else {
				sessionID = fmt.Sprintf("%d", time.Now().UnixNano())
				http.SetCookie(ctx.ResponseWriter, &http.Cookie{
					Name:     "session",
					Value:    sessionID,
					Path:     "/",
					HttpOnly: true,
					SameSite: http.SameSiteStrictMode,
				})
			}
			
			// Get or create session
			session, exists := sessions[sessionID]
			if !exists {
				session = &Session{store: make(map[string]interface{})}
				sessions[sessionID] = session
			}
			
			// Add session to request context
			ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), "session", session))
			
			return next(ctx)
		}
	}
}

// GetSession retrieves the session from context
func GetSession(ctx *Context) *Session {
	if session, ok := ctx.Request.Context().Value("session").(*Session); ok {
		return session
	}
	return nil
}