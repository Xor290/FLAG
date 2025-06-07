package handlers

import (
	"backend/db"
	"backend/models"
	"backend/services"
	"net/http"
	"strconv"
	"log"
	"github.com/gin-gonic/gin"
)

type ChallengeResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Difficulty  string `json:"difficulty"`
	Points      int    `json:"points"`
	DockerImage string `json:"docker_image"`
	Port        int    `json:"port"`
	CPULimit    string `json:"cpu_limit"`
	MemoryLimit string `json:"memory_limit"`
	CreatedAt   string `json:"created_at"`
}


func GetStatsHandler(c *gin.Context) {
	var stats struct {
		TotalChallenges int `json:"total_challenges"`
		WebChallenges   int `json:"web_challenges"`
		CryptoChallenges int `json:"crypto_challenges"`
		ReverseChallenges int `json:"reverse_challenges"`
		ForensicsChallenges int `json:"forensics_challenges"`
		NetworkChallenges int `json:"network_challenges"`
		StegoChallenges int `json:"stego_challenges"`
		TotalUsers      int `json:"total_users"`
		TotalSubmissions int `json:"total_submissions"`
	}

	// Compter le nombre total de challenges
	err := db.DB.QueryRow("SELECT COUNT(*) FROM challenges").Scan(&stats.TotalChallenges)
	if err != nil {
		log.Printf("Erreur lors du comptage des challenges: %v", err)
		stats.TotalChallenges = 0
	}

	// Compter les challenges par catégorie
	categories := map[string]*int{
		"Web":       &stats.WebChallenges,
		"Crypto":    &stats.CryptoChallenges,
		"Reverse":   &stats.ReverseChallenges,
		"Forensics": &stats.ForensicsChallenges,
		"Network":   &stats.NetworkChallenges,
		"Stego":     &stats.StegoChallenges,
	}

	for category, count := range categories {
		err := db.DB.QueryRow("SELECT COUNT(*) FROM challenges WHERE category = ?", category).Scan(count)
		if err != nil {
			log.Printf("Erreur lors du comptage des challenges %s: %v", category, err)
			*count = 0
		}
	}

	// Compter le nombre total d'utilisateurs
	err = db.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	if err != nil {
		log.Printf("Erreur lors du comptage des utilisateurs: %v", err)
		stats.TotalUsers = 0
	}

	// Compter le nombre total de soumissions
	err = db.DB.QueryRow("SELECT COUNT(*) FROM submissions").Scan(&stats.TotalSubmissions)
	if err != nil {
		log.Printf("Erreur lors du comptage des soumissions: %v", err)
		stats.TotalSubmissions = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"stats":   stats,
	})
}


// GetChallengesHandler retourne tous les challenges actifs
func GetChallengesHandler(c *gin.Context) {
	// Récupérer les paramètres de requête pour le filtrage
	category := c.Query("category")
	
	// Construire la requête SQL de base
	query := `
		SELECT id, name, description, category, difficulty, points, docker_image, port, cpu_limit, memory_limit, created_at 
		FROM challenges 
		WHERE is_active = 1`
	
	var args []interface{}
	
	// Ajouter le filtre par catégorie si spécifié
	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}
	
	query += " ORDER BY difficulty ASC, points ASC"

	// Exécuter la requête
	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("Erreur lors de la récupération des challenges: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération des challenges"})
		return
	}
	defer rows.Close()

	var challenges []ChallengeResponse
	for rows.Next() {
		var challenge ChallengeResponse
		err := rows.Scan(
			&challenge.ID,
			&challenge.Name,
			&challenge.Description,
			&challenge.Category,
			&challenge.Difficulty,
			&challenge.Points,
			&challenge.DockerImage,
			&challenge.Port,
			&challenge.CPULimit,
			&challenge.MemoryLimit,
			&challenge.CreatedAt,
		)
		if err != nil {
			log.Printf("Erreur lors du scan du challenge: %v", err)
			continue
		}
		challenges = append(challenges, challenge)
	}

	// Vérifier s'il y a eu des erreurs lors de l'itération
	if err = rows.Err(); err != nil {
		log.Printf("Erreur lors de l'itération des challenges: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la lecture des challenges"})
		return
	}

	log.Printf("Nombre de challenges récupérés: %d pour la catégorie: %s", len(challenges), category)

	// Retourner la réponse JSON
	c.JSON(http.StatusOK, gin.H{
		"challenges": challenges,
		"count":      len(challenges),
		"category":   category,
		"success":    true,
	})
}


// GetAllChallenges récupère tous les challenges actifs
func GetAllChallenges(c *gin.Context) {
	challenges, err := db.GetAllChallenges()
	if err != nil {
		log.Printf("Error fetching challenges: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération des challenges"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"challenges": challenges})
}

// GetChallengeByID récupère un challenge par son ID
func GetChallengeByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}
	
	challenge, err := db.GetChallengeByID(id)
	if err != nil {
		log.Printf("Error fetching challenge %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération du challenge"})
		return
	}
	
	if challenge == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Challenge non trouvé"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"challenge": challenge})
}

// CreateChallenge crée un nouveau challenge (admin seulement)
func CreateChallenge(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Port        int    `json:"port" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Données invalides", "details": err.Error()})
		return
	}

	if req.Name == "" || req.Port == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nom et port sont obligatoires"})
		return
	}

	// Validation du port (plage valide)
	if req.Port < 1024 || req.Port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le port doit être entre 1024 et 65535"})
		return
	}

	challenge := &models.Challenge{
		Name:        req.Name,
		Description: req.Description,
		Port:        req.Port,
	}

	id, err := db.CreateChallenge(challenge)
	if err != nil {
		log.Printf("Error creating challenge: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création du challenge"})
		return
	}

	challenge.ID = id
	c.JSON(http.StatusCreated, gin.H{"challenge": challenge})
}

// DeleteChallenge supprime un challenge (admin seulement)
func DeleteChallenge(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	// Vérifier si le challenge existe
	challenge, err := db.GetChallengeByID(id)
	if err != nil {
		log.Printf("Error checking challenge %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la vérification du challenge"})
		return
	}

	if challenge == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Challenge non trouvé"})
		return
	}

	// Supprimer le challenge
	err = db.DeleteChallenge(id)
	if err != nil {
		log.Printf("Error deleting challenge %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression du challenge"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Challenge supprimé avec succès"})
}

// GetChallengeUserInstances récupère les instances d'un utilisateur (nom modifié pour éviter le conflit)
func GetChallengeUserInstances(c *gin.Context) {
	userIDStr := c.Param("user_id")
	if userIDStr == "" {
		// Si pas de paramètre, essayer de récupérer depuis le contexte (utilisateur connecté)
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur manquant"})
			return
		}
		userIDStr = strconv.Itoa(userID.(int))
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur invalide"})
		return
	}

	instances, err := db.GetUserInstances(userID)
	if err != nil {
		log.Printf("Error fetching instances for user %d: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération des instances"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"instances": instances})
}

// CreateInstance crée une nouvelle instance pour un challenge
func CreateInstance(c *gin.Context) {
	var req struct {
		UserID      int `json:"user_id"`
		ChallengeID int `json:"challenge_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Données invalides",
			"details": err.Error(),
		})
		return
	}

	// Si user_id n'est pas fourni, utiliser l'utilisateur connecté
	if req.UserID == 0 {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID utilisateur manquant"})
			return
		}
		req.UserID = userID.(int)
	}

	// Vérifier que le challenge existe
	challenge, err := db.GetChallengeByID(req.ChallengeID)
	if err != nil {
		log.Printf("Error fetching challenge %d: %v", req.ChallengeID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la vérification du challenge"})
		return
	}

	if challenge == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Challenge non trouvé"})
		return
	}

	// Vérifier si l'utilisateur a déjà une instance active pour ce challenge
	existingInstances, err := db.GetUserChallengeInstances(req.UserID, req.ChallengeID)
	if err != nil {
		log.Printf("Error checking existing instances: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la vérification des instances existantes"})
		return
	}

	if len(existingInstances) > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Une instance active existe déjà pour ce challenge",
			"instance": existingInstances[0],
		})
		return
	}

	// Créer l'instance
	instance := &models.Instance{
		UserID:      req.UserID,
		ChallengeID: req.ChallengeID,
		Status:      "creating",
		Namespace:   "ctf-instances",
	}

	// Initialiser le service Kubernetes
	k8sService, err := services.NewKubernetesService()
	if err != nil {
		log.Printf("Failed to initialize Kubernetes service: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de l'initialisation du service"})
		return
	}

	// Créer l'instance dans Kubernetes
	if err := k8sService.CreateChallengeInstance(instance, challenge); err != nil {
		log.Printf("Failed to create challenge instance: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la création de l'instance"})
		return
	}

	// Sauvegarder l'instance en base de données
	instanceID, err := db.CreateInstance(instance)
	if err != nil {
		log.Printf("Error saving instance to database: %v", err)
		
		// Essayer de nettoyer l'instance Kubernetes en cas d'erreur
		if cleanupErr := k8sService.CleanupChallengeInstance(instance); cleanupErr != nil {
			log.Printf("Failed to cleanup Kubernetes instance: %v", cleanupErr)
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la sauvegarde de l'instance"})
		return
	}

	// Mettre à jour l'ID de l'instance et le statut
	instance.ID = instanceID
	instance.Status = "running"

	// Préparer la réponse avec l'URL d'accès
	response := gin.H{
		"success": true,
		"instance": gin.H{
			"id":           instance.ID,
			"user_id":      instance.UserID,
			"challenge_id": instance.ChallengeID,
			"status":       instance.Status,
			"pod_name":     instance.PodName,
			"service_name": instance.ServiceName,
			"access_url":   instance.AccessURL,
			"external_port": instance.ExternalPort,
			"internal_port": instance.InternalPort,
			"namespace":    instance.Namespace,
			"created_at":   time.Now().Format(time.RFC3339),
		},
		"message": "Instance créée avec succès",
	}

	log.Printf("Instance créée avec succès: ID=%d, URL=%s", instance.ID, instance.AccessURL)
	c.JSON(http.StatusCreated, response)
}

// DeleteInstance supprime une instance
func DeleteInstance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID invalide"})
		return
	}

	// Récupérer l'instance
	instance, err := db.GetInstanceByID(id)
	if err != nil {
		log.Printf("Error fetching instance %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la récupération de l'instance"})
		return
	}

	if instance == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Instance non trouvée"})
		return
	}

	// Vérifier les permissions (utilisateur propriétaire ou admin)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Utilisateur non authentifié"})
		return
	}

	isAdmin, _ := c.Get("is_admin")
	if instance.UserID != userID.(int) && !isAdmin.(bool) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Accès refusé"})
		return
	}

	// Supprimer l'instance de Kubernetes
	k8sService, err := services.NewKubernetesService()
	if err != nil {
		log.Printf("Failed to initialize Kubernetes service: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de l'initialisation du service"})
		return
	}

	if err := k8sService.CleanupChallengeInstance(instance); err != nil {
		log.Printf("Failed to delete Kubernetes instance: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression de l'instance Kubernetes"})
		return
	}

	// Supprimer l'instance de la base de données
	if err := db.DeleteInstance(id); err != nil {
		log.Printf("Error deleting instance from database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors de la suppression de l'instance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Instance supprimée avec succès"})
}
