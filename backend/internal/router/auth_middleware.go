package router

import (
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

func extractBearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(header) > len(prefix) && header[:len(prefix)] == prefix {
		return header[len(prefix):]
	}
	return ""
}
