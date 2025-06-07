package handlers

import (
	"backend/db"
	"backend/models"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey = []byte(os.Getenv("JWT_SECRET"))

func Register(c *gin.Context) {
	var credentials struct {
		Username string `json:"username" binding:"required,min=3,max=50"`
		Password string `json:"password" binding:"required,min=6"`
		Email    string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides", "details": err.Error()})
		return
	}

	// Vérifier si l'utilisateur existe déjà
	existingUser, err := db.GetUserByUsername(credentials.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
		return
	}
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Nom d'utilisateur déjà utilisé"})
		return
	}

	// Hasher le mot de passe
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(credentials.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors du hashage du mot de passe"})
		return
	}

	// Créer l'utilisateur en base
	user := &models.User{
		Username:     credentials.Username,
		Email:        credentials.Email,
		PasswordHash: string(hashedPassword),
		Role:         "user", // Par défaut, les nouveaux utilisateurs ne sont pas admin
		CreatedAt:    time.Now(),
	}

	err = db.InsertUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création de l'utilisateur"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Utilisateur créé avec succès"})
}

func Login(c *gin.Context) {
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Récupérer l'utilisateur en base
	user, err := db.GetUserByUsername(credentials.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
		return
	}
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identifiants invalides"})
		return
	}

	// Comparer le mot de passe hashé
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(credentials.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Identifiants invalides"})
		return
	}

	// Création du token JWT
	expirationTime := time.Now().Add(1 * time.Hour)
	claims := &jwt.StandardClaims{
		ExpiresAt: expirationTime.Unix(),
		Subject:   user.Username,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Impossible de générer le token"})
		return
	}

	session := sessions.Default(c)
	session.Set("token", tokenString)
	session.Set("user_id", user.ID)
	session.Set("username", user.Username)
	isAdmin := user.Role == "admin"
	session.Set("is_admin", isAdmin)
	session.Save()

	// Mettre à jour la dernière connexion
	db.UpdateUserLastLogin(user.ID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Connexion réussie",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"is_admin": isAdmin,
		},
	})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		tokenString := session.Get("token")
		if tokenString == nil {
			log.Println("Unauthorized: Missing token")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token manquant"})
			c.Abort()
			return
		}

		claims := &jwt.StandardClaims{}
		token, err := jwt.ParseWithClaims(tokenString.(string), claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			log.Printf("Unauthorized: Invalid token. Error: %v\n", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide"})
			c.Abort()
			return
		}

		// Ajouter les informations utilisateur au contexte
		c.Set("username", claims.Subject)
		userID := session.Get("user_id")
		if userID != nil {
			c.Set("user_id", userID)
		}
		isAdmin := session.Get("is_admin")
		if isAdmin != nil {
			c.Set("is_admin", isAdmin)
		}

		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, exists := c.Get("is_admin")
		if !exists || !isAdmin.(bool) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Accès administrateur requis"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func GetMe(c *gin.Context) {
	session := sessions.Default(c)
	token := session.Get("token")
	if token == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié"})
		return
	}

	username := session.Get("username")
	userID := session.Get("user_id")
	isAdmin := session.Get("is_admin")

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":       userID,
			"username": username,
			"is_admin": isAdmin,
		},
	})
}

func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.JSON(http.StatusOK, gin.H{"message": "Déconnexion réussie"})
}