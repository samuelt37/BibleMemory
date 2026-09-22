package router

import (
	"log"
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/samuelt37/BibleMemory/internal/auth"
	"github.com/samuelt37/BibleMemory/internal/service"
)

func RequireAuth(userService *service.UserService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionToken := extractBearerToken(r)
			if sessionToken == "" {
				http.Error(w, "missing authorization token", http.StatusUnauthorized)
				return
			}

			claims, err := jwt.Verify(r.Context(), &jwt.VerifyParams{
				Token: sessionToken,
			})
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			internalUserID, err := userService.EnsureUser(claims.Subject)
			if err != nil {
				http.Error(w, "failed to resolve user", http.StatusInternalServerError)
				return
			}

			ctx := auth.WithUserID(r.Context(), internalUserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func OptionalAuth(userService *service.UserService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				log.Println("OptionalAuth: no Bearer token found in request headers")
				next.ServeHTTP(w, r) // no token, proceed as anonymous
				return
			}

			claims, err := jwt.Verify(r.Context(), &jwt.VerifyParams{Token: token})
			if err != nil {
				log.Printf("OptionalAuth: jwt.Verify failed: %v", err)
				next.ServeHTTP(w, r) // invalid/expired token, proceed as anonymous rather than erroring
				return
			}

			internalUserID, err := userService.EnsureUser(claims.Subject)
			if err != nil {
				log.Printf("OptionalAuth: EnsureUser failed: %v", err)
				next.ServeHTTP(w, r) // DB error resolving user — fail open, proceed as anonymous
				return
			}

			log.Printf("OptionalAuth: authenticated user %d (clerk: %s)", internalUserID, claims.Subject)
			ctx := auth.WithUserID(r.Context(), internalUserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(header) > len(prefix) && header[:len(prefix)] == prefix {
		return header[len(prefix):]
	}
	return ""
}
