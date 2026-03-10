package middleware

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// AdminAuth returns middleware that requires a valid X-Admin-Secret header.
func AdminAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if c.Request().Header.Get("X-Admin-Secret") != secret {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}
			return next(c)
		}
	}
}
