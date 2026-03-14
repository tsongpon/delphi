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
	teamRepo := repository.NewTeamFirestoreRepository(firestoreClient)
	inviteLinkRepo := repository.NewInviteLinkFirestoreRepository(firestoreClient)

	userService := service.NewUserService(userRepo, teamRepo, jwtSecret)
	feedbackService := service.NewFeedbackService(feedbackRepo, userRepo)
	passwordResetService := service.NewPasswordResetService(tokenRepo, userRepo, appBaseURL)
	teamService := service.NewTeamService(teamRepo)
	inviteLinkService := service.NewInviteLinkService(inviteLinkRepo, teamRepo, jwtSecret, appBaseURL)

	authHandler := handler.NewAuthHandler(userService, inviteLinkService)
	userHandler := handler.NewUserHandler(userService)
	feedbackHandler := handler.NewFeedbackHandler(feedbackService)
	passwordResetHandler := handler.NewPasswordResetHandler(passwordResetService)
	adminHandler := handler.NewAdminHandler(userService, teamService)
	inviteLinkHandler := handler.NewInviteLinkHandler(inviteLinkService)

	e := echo.New()
	e.Use(middleware.CORS("*"))

	// Public routes
	e.GET("/ping", func(c *echo.Context) error {
		return c.String(http.StatusOK, "pong")
	})
	e.POST("/register", authHandler.RegisterUser)
	e.POST("/login", authHandler.LoginUser)
	e.POST("/reset-password", passwordResetHandler.ResetPassword)
	e.GET("/invite-links/validate", inviteLinkHandler.ValidateInviteToken)

	// Protected routes (JWT required)
	api := e.Group("", custommiddleware.JWTAuth(jwtSecret))
	api.GET("/me/teammates", userHandler.GetTeammates)
	api.GET("/me/feedbacks", feedbackHandler.GetMyFeedbacks)
	api.GET("/me/given-feedbacks", feedbackHandler.GetMyGivenFeedbacks)
	api.POST("/feedbacks", feedbackHandler.CreateFeedback)
	api.GET("/teams/:teamId/feedbacks", feedbackHandler.GetTeamFeedbacks, custommiddleware.RequireRole("manager"))
	api.GET("/teams/:teamId/dashboard", feedbackHandler.GetTeamDashboard, custommiddleware.RequireRole("manager"))
	api.GET("/teams/:teamId/members/:memberId/feedbacks", feedbackHandler.GetMemberFeedbacks, custommiddleware.RequireRole("manager"))
	api.POST("/teams/:teamId/invite-links", inviteLinkHandler.CreateInviteLink, custommiddleware.RequireRole("manager"))
	api.GET("/teams/:teamId/invite-links", inviteLinkHandler.ListInviteLinks, custommiddleware.RequireRole("manager"))
	api.DELETE("/teams/:teamId/invite-links/:linkId", inviteLinkHandler.RevokeInviteLink, custommiddleware.RequireRole("manager"))

	// Admin routes (ADMIN_SECRET header required)
	admin := e.Group("/admin", custommiddleware.AdminAuth(adminSecret))
	admin.POST("/users/:userID/reset-link", passwordResetHandler.GenerateResetLink)
	admin.PUT("/users/:userID/role", adminHandler.UpdateUserRole)
	admin.POST("/teams", adminHandler.CreateTeam)

	if err := e.Start(":8080"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}
