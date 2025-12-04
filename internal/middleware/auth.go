package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"rainchanel.com/internal/api/response"
	"rainchanel.com/internal/auth"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.JSON(http.StatusUnauthorized, response.Response{
				Error: &response.Error{
					Code:    http.StatusUnauthorized,
					Message: "Authorization header required",
				},
			})
			ctx.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			ctx.JSON(http.StatusUnauthorized, response.Response{
				Error: &response.Error{
					Code:    http.StatusUnauthorized,
					Message: "Invalid authorization header format",
				},
			})
			ctx.Abort()
			return
		}

		token := parts[1]
		claims, err := auth.ValidateToken(token)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, response.Response{
				Error: &response.Error{
					Code:    http.StatusUnauthorized,
					Message: "Invalid or expired token",
				},
			})
			ctx.Abort()
			return
		}

		ctx.Set("user_id", claims.UserID)
		ctx.Set("username", claims.Username)

		ctx.Next()
	}
}
