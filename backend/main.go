package main

import (
	"backend/db"
	"backend/handlers"
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Erreur lors du chargement du fichier .env: %v", err)
	}

	db.InitDB()
	defer db.CloseDB()

	r := gin.Default()

	// Configuration des sessions
	store := cookie.NewStore([]byte(os.Getenv("SESSION_SECRET")))
	store.Options(sessions.Options{
		Path:     "/",
		Domain:   os.Getenv("DOMAIN"),
		MaxAge:   3600 * 24, // 24h pour les CTF
		HttpOnly: true,
		Secure:   os.Getenv("ENV") == "production",
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("ctf_session", store))

	// Configuration CORS
	allowedOrigins := []string{"http://localhost:3000", "http://localhost:5173"}
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL != "" {
		allowedOrigins = append(allowedOrigins, frontendURL)
	}
	
	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// --------------------------------------------
	// Static files
	// --------------------------------------------
	// Servir les fichiers statiques du frontend
	r.Static("/static", "./frontend/templates")
	r.StaticFile("/", "../frontend/templates/index.html")
	r.StaticFile("/login","../frontend/templates/login.html")
	r.StaticFile("/signin","../frontend/templates/signin.html")
	r.StaticFile("/dashboard-web","../frontend/templates/dashboard-web.html")
	r.StaticFile("/dashboard-reverse","../frontend/templates/dashboard-reverse.html")
	r.StaticFile("/dashboard-forensics","../frontend/templates/dashboard-forensics.html")

	// --------------------------------------------
	// Public routes
	// --------------------------------------------
	// Authentication routes (publiques)
	r.POST("/auth/register", handlers.Register) //prob quand on se connecte à la page d'accueil 
	r.POST("/auth/login", handlers.Login)
	r.POST("/auth/logout", handlers.Logout)

	// Challenges (lecture seule pour tous)
	r.GET("/challenges", handlers.GetAllChallenges)
	r.GET("/challenges/:id", handlers.GetChallengeByID)
	
	// 🆕 NOUVELLE ROUTE : Statistiques des challenges (publique)
	r.GET("/api/stats", handlers.GetStatsHandler)
	
	// ✨ ROUTE POUR INITIALISER LES CHALLENGES (décommentée si nécessaire)
	//r.POST("/challenges/init-chall", handlers.InitDefaultChallengesHandler)
	
	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// --------------------------------------------
	// Protected routes
	// --------------------------------------------
	protected := r.Group("/api")
	protected.Use(handlers.AuthMiddleware())
	{
		// User info
		protected.GET("/auth/me", handlers.GetMe)
		
		// Challenge instances
		protected.GET("/instances/:user_id", handlers.GetUserInstances)
		protected.POST("/instances", handlers.CreateInstance)
		protected.GET("/challenges", handlers.GetChallengesHandler)
	}
	// --------------------------------------------
	// Admin routes
	// --------------------------------------------
	admin := r.Group("/api/admin")
	admin.Use(handlers.AuthMiddleware(), handlers.AdminMiddleware())
	{
		// Challenge management
		admin.POST("/challenges", handlers.CreateChallenge)
		admin.DELETE("/challenges/:id", handlers.DeleteChallenge)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("CTF Platform starting on port %s", port)
	r.Run(":" + port)
}