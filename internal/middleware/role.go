package middleware

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// RequireRole returns middleware that allows only users with the specified role.
// Must be used after JWTAuth, which sets the "role" context value.
func RequireRole(role string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			userRole, _ := c.Get("role").(string)
			if userRole != role {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
			}
			return next(c)
		}
	}
}
