package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"backend/models"
	"backend/db"
)

func GetUserInstances(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	rows, err := db.DB.Query("SELECT id, user_id, challenge_id, container_name, started_at, expires_at, status FROM instances WHERE user_id=?", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	instances := []models.Instance{}
	for rows.Next() {
		var inst models.Instance
		err := rows.Scan(&inst.ID, &inst.UserID, &inst.ChallengeID, &inst.ContainerName, &inst.StartedAt, &inst.ExpiresAt, &inst.Status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		instances = append(instances, inst)
	}

	c.JSON(http.StatusOK, instances)
}
