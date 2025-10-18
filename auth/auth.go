// auth middleware to defance targets
package auth

import (
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/osamikoyo/orion/config"
	"github.com/rs/zerolog"
)

type AuthMW struct {
	cfg    *config.AuthConfig
	logger *zerolog.Logger
}

func NewAuthMW(cfg *config.AuthConfig, logger *zerolog.Logger) *AuthMW {
	return &AuthMW{
		cfg:    cfg,
		logger: logger,
	}
}

func (a *AuthMW) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenStr := r.Header.Get("Authorization")
		if tokenStr == "" {
			http.Error(w, "empty auth token", http.StatusNonAuthoritativeInfo)
			return
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return []byte(a.cfg.Key), nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "failed to parse token", http.StatusBadGateway)
			return
		}

		next.ServeHTTP(w, r)
	})
}
