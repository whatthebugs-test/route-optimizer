package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// RequireServiceToken checks the same HS256 service JWTs that shipment-api issues.
func RequireServiceToken(secret []byte, scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		claims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (any, error) { return secret, nil },
			jwt.WithValidMethods([]string{"HS256"}),
			jwt.WithIssuer("shipment-api"),
			jwt.WithExpirationRequired(),
		)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		scopes, _ := claims["scope"].(string)
		for _, s := range strings.Fields(scopes) {
			if s == scope {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "missing scope " + scope})
	}
}
