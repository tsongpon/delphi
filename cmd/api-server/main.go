package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/tsongpon/delphi/internal/handler"
	custommiddleware "github.com/tsongpon/delphi/internal/middleware"
	"github.com/tsongpon/delphi/internal/repository"
	"github.com/tsongpon/delphi/internal/service"
)

func main() {
	// Load .env file if present (ignored if not found)
	_ = godotenv.Load()

	ctx := context.Background()

	projectID := os.Getenv("GCP_PROJECT_ID")
	databaseID := os.Getenv("GCP_FIRESTORE_DATABASE_ID")
	if projectID == "" {
		log.Fatal("GCP_PROJECT_ID environment variable is required")
	}

	firestoreClient, err := firestore.NewClientWithDatabase(ctx, projectID, databaseID)
	if err != nil {
		log.Fatalf("failed to create firestore client: %v", err)
	}
	defer firestoreClient.Close()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	adminSecret := os.Getenv("ADMIN_SECRET")
	if adminSecret == "" {
		log.Fatal("ADMIN_SECRET environment variable is required")
	}

	appBaseURL := os.Getenv("APP_BASE_URL")
	if appBaseURL == "" {
		log.Fatal("APP_BASE_URL environment variable is required")
	}

	userRepo := repository.NewUserFirestoreRepository(firestoreClient)
	feedbackRepo := repository.NewFeedbackFirestoreRepository(firestoreClient)
	tokenRepo := repository.NewTokenFirestoreRepository(firestoreClient)

	userService := service.NewUserService(userRepo, jwtSecret)
	feedbackService := service.NewFeedbackService(feedbackRepo, userRepo)
	passwordResetService := service.NewPasswordResetService(tokenRepo, userRepo, appBaseURL)

	authHandler := handler.NewAuthHandler(userService)
	userHandler := handler.NewUserHandler(userService)
	feedbackHandler := handler.NewFeedbackHandler(feedbackService)
	passwordResetHandler := handler.NewPasswordResetHandler(passwordResetService)

	e := echo.New()
	e.Use(middleware.CORS("*"))

	// Public routes
	e.GET("/ping", func(c *echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})
	e.POST("/register", authHandler.RegisterUser)
	e.POST("/login", authHandler.LoginUser)
	e.POST("/reset-password", passwordResetHandler.ResetPassword)

	// Protected routes (JWT required)
	api := e.Group("", custommiddleware.JWTAuth(jwtSecret))
	api.GET("/me/teammates", userHandler.GetTeammates)
	api.GET("/me/feedbacks", feedbackHandler.GetMyFeedbacks)
	api.POST("/feedbacks", feedbackHandler.CreateFeedback)

	// Admin routes (ADMIN_SECRET header required)
	admin := e.Group("/admin", custommiddleware.AdminAuth(adminSecret))
	admin.POST("/users/:userID/reset-link", passwordResetHandler.GenerateResetLink)

	if err := e.Start(":8080"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}
