package router

import (
	"os"
	auth_handler "stock-logger/internal/auth/handler"
	filesxls_handler "stock-logger/internal/filesxls/handler"
	reports_handler "stock-logger/internal/reports/handler"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// JWTMiddleware checks for valid JWT in Authorization header
func JWTMiddleware(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(401).JSON(fiber.Map{
			"error": "Authorization header missing",
		})
	}

	// Check if it's a Bearer token
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid authorization header format",
		})
	}

	tokenString := tokenParts[1]
	secret := os.Getenv("JWT_SECRET")

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	// Token is valid, proceed to next handler
	return c.Next()
}

// SetupRoutes configures all application routes
func SetupRoutes(app *fiber.App, reportsHandler *reports_handler.Handler, excelHandler *filesxls_handler.Handler, authHandler *auth_handler.Handler) {
	// Add routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"Title": "Stock Logger Dashboard",
		}) // Render HTML template instead of sending plain text
	})

	// Apply JWT middleware to protect these endpoints
	app.Get("/api/stocks", JWTMiddleware, reportsHandler.GetReports)

	// Add route for Excel report generation
	app.Post("/api/excel/generate", JWTMiddleware, excelHandler.GenerateReport)

	// Add route for listing Excel report files
	app.Get("/api/excel/list", JWTMiddleware, excelHandler.ListFiles)

	// Add login route - should be public, no middleware
	app.Post("/login", authHandler.Auth)

	// Serve static files from file-reports directory for downloads
	app.Static("/file-reports", "./file-reports")

	// Add reports page route - should be public but client-side will handle auth
	app.Get("/reports", func(c *fiber.Ctx) error {
		return c.Render("reports", fiber.Map{
			"Title": "Generated Reports",
		})
	})
}
