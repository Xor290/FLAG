package db

import (
	"backend/models"
	"database/sql"
	"errors"
	"log"
	"time"
)



// GetChallengeByID récupère un challenge par son ID
func GetChallengeByID(id int) (*models.Challenge, error) {
	row := DB.QueryRow(`SELECT id, name, description, docker_image, port, cpu_limit, memory_limit, created_at 
		FROM challenges WHERE id = ? AND is_active = 1`, id)
	
	var challenge models.Challenge
	var createdAtStr string
	
	err := row.Scan(&challenge.ID, &challenge.Name, &challenge.Description, 
		&challenge.DockerImage, &challenge.Port, &challenge.CPULimit, 
		&challenge.MemoryLimit, &createdAtStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	
	challenge.CreatedAt = createdAtStr
	return &challenge, nil
}

// CreateChallenge crée un nouveau challenge
func CreateChallenge(challenge *models.Challenge) (int, error) {
	stmt, err := DB.Prepare(`INSERT INTO challenges (name, description, category, difficulty, points, flag, docker_image, port, cpu_limit, memory_limit, time_limit) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	// Valeurs par défaut si non définies
	category := "general"
	difficulty := "medium"
	points := 100
	flag := "CTF{placeholder}"
	dockerImage := "nginx:latest"
	cpuLimit := "0.5"
	memoryLimit := "512Mi"
	timeLimit := 3600

	result, err := stmt.Exec(challenge.Name, challenge.Description, category, difficulty, points, flag, dockerImage, challenge.Port, cpuLimit, memoryLimit, timeLimit)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

// DeleteChallenge supprime un challenge (soft delete)
func DeleteChallenge(id int) error {
	_, err := DB.Exec("UPDATE challenges SET is_active = 0 WHERE id = ?", id)
	return err
}

// GetUserInstances récupère toutes les instances d'un utilisateur
func GetUserInstances(userID int) ([]models.Instance, error) {
	rows, err := DB.Query(`SELECT i.id, i.user_id, i.challenge_id, i.container_name, i.pod_name, i.service_name, 
		i.namespace, i.external_port, i.internal_port, i.access_url, i.status, i.started_at, i.expires_at, i.last_accessed,
		c.id, c.name, c.description, c.docker_image, c.port, c.cpu_limit, c.memory_limit, c.created_at
		FROM instances i 
		LEFT JOIN challenges c ON i.challenge_id = c.id 
		WHERE i.user_id = ? AND i.status != 'deleted'
		ORDER BY i.started_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []models.Instance
	for rows.Next() {
		var instance models.Instance
		var challenge models.Challenge
		var startedAtStr, expiresAtStr, challengeCreatedAtStr string
		var podName, serviceName, accessURL sql.NullString
		var externalPort, internalPort sql.NullInt64
		var lastAccessed sql.NullString

		err := rows.Scan(&instance.ID, &instance.UserID, &instance.ChallengeID, &instance.ContainerName,
			&podName, &serviceName, &instance.Namespace, &externalPort, &internalPort, &accessURL,
			&instance.Status, &startedAtStr, &expiresAtStr, &lastAccessed,
			&challenge.ID, &challenge.Name, &challenge.Description, &challenge.DockerImage,
			&challenge.Port, &challenge.CPULimit, &challenge.MemoryLimit, &challengeCreatedAtStr)
		if err != nil {
			log.Printf("Error scanning instance: %v", err)
			continue
		}

		// Parse les dates
		if instance.StartedAt, err = time.Parse("2006-01-02 15:04:05", startedAtStr); err != nil {
			log.Printf("Warning: failed to parse started_at date: %v", err)
		}
		if instance.ExpiresAt, err = time.Parse("2006-01-02 15:04:05", expiresAtStr); err != nil {
			log.Printf("Warning: failed to parse expires_at date: %v", err)
		}
		if lastAccessed.Valid {
			if instance.LastAccessed, err = time.Parse("2006-01-02 15:04:05", lastAccessed.String); err != nil {
				log.Printf("Warning: failed to parse last_accessed date: %v", err)
			}
		}

		// Parse les valeurs nullables
		if podName.Valid {
			instance.PodName = podName.String
		}
		if serviceName.Valid {
			instance.ServiceName = serviceName.String
		}
		if accessURL.Valid {
			instance.AccessURL = accessURL.String
		}
		if externalPort.Valid {
			instance.ExternalPort = int(externalPort.Int64)
		}
		if internalPort.Valid {
			instance.InternalPort = int(internalPort.Int64)
		}

		challenge.CreatedAt = challengeCreatedAtStr
		instance.Challenge = &challenge
		instances = append(instances, instance)
	}

	return instances, nil
}

// GetUserChallengeInstances récupère les instances d'un utilisateur pour un challenge spécifique
func GetUserChallengeInstances(userID, challengeID int) ([]models.Instance, error) {
	rows, err := DB.Query(`SELECT id, user_id, challenge_id, container_name, pod_name, service_name, 
		namespace, external_port, internal_port, access_url, status, started_at, expires_at, last_accessed
		FROM instances 
		WHERE user_id = ? AND challenge_id = ? AND status IN ('creating', 'running', 'ready')`, userID, challengeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []models.Instance
	for rows.Next() {
		var instance models.Instance
		var startedAtStr, expiresAtStr string
		var podName, serviceName, accessURL sql.NullString
		var externalPort, internalPort sql.NullInt64
		var lastAccessed sql.NullString

		err := rows.Scan(&instance.ID, &instance.UserID, &instance.ChallengeID, &instance.ContainerName,
			&podName, &serviceName, &instance.Namespace, &externalPort, &internalPort, &accessURL,
			&instance.Status, &startedAtStr, &expiresAtStr, &lastAccessed)
		if err != nil {
			log.Printf("Error scanning instance: %v", err)
			continue
		}

		// Parse les dates
		if instance.StartedAt, err = time.Parse("2006-01-02 15:04:05", startedAtStr); err != nil {
			log.Printf("Warning: failed to parse started_at date: %v", err)
		}
		if instance.ExpiresAt, err = time.Parse("2006-01-02 15:04:05", expiresAtStr); err != nil {
			log.Printf("Warning: failed to parse expires_at date: %v", err)
		}
		if lastAccessed.Valid {
			if instance.LastAccessed, err = time.Parse("2006-01-02 15:04:05", lastAccessed.String); err != nil {
				log.Printf("Warning: failed to parse last_accessed date: %v", err)
			}
		}

		// Parse les valeurs nullables
		if podName.Valid {
			instance.PodName = podName.String
		}
		if serviceName.Valid {
			instance.ServiceName = serviceName.String
		}
		if accessURL.Valid {
			instance.AccessURL = accessURL.String
		}
		if externalPort.Valid {
			instance.ExternalPort = int(externalPort.Int64)
		}
		if internalPort.Valid {
			instance.InternalPort = int(internalPort.Int64)
		}

		instances = append(instances, instance)
	}

	return instances, nil
}

// CreateInstance crée une nouvelle instance
func CreateInstance(instance *models.Instance) (int, error) {
	// Définir une date d'expiration (par exemple, 2 heures après la création)
	expiresAt := time.Now().Add(2 * time.Hour)

	stmt, err := DB.Prepare(`INSERT INTO instances (user_id, challenge_id, container_name, pod_name, service_name, 
		namespace, external_port, internal_port, access_url, status, expires_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	result, err := stmt.Exec(instance.UserID, instance.ChallengeID, instance.ContainerName,
		instance.PodName, instance.ServiceName, instance.Namespace, instance.ExternalPort,
		instance.InternalPort, instance.AccessURL, instance.Status, expiresAt)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

// GetInstanceByID récupère une instance par son ID
func GetInstanceByID(id int) (*models.Instance, error) {
	row := DB.QueryRow(`SELECT id, user_id, challenge_id, container_name, pod_name, service_name, 
		namespace, external_port, internal_port, access_url, status, started_at, expires_at, last_accessed
		FROM instances WHERE id = ?`, id)

	var instance models.Instance
	var startedAtStr, expiresAtStr string
	var podName, serviceName, accessURL sql.NullString
	var externalPort, internalPort sql.NullInt64
	var lastAccessed sql.NullString

	err := row.Scan(&instance.ID, &instance.UserID, &instance.ChallengeID, &instance.ContainerName,
		&podName, &serviceName, &instance.Namespace, &externalPort, &internalPort, &accessURL,
		&instance.Status, &startedAtStr, &expiresAtStr, &lastAccessed)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Parse les dates
	if instance.StartedAt, err = time.Parse("2006-01-02 15:04:05", startedAtStr); err != nil {
		log.Printf("Warning: failed to parse started_at date: %v", err)
	}
	if instance.ExpiresAt, err = time.Parse("2006-01-02 15:04:05", expiresAtStr); err != nil {
		log.Printf("Warning: failed to parse expires_at date: %v", err)
	}
	if lastAccessed.Valid {
		if instance.LastAccessed, err = time.Parse("2006-01-02 15:04:05", lastAccessed.String); err != nil {
			log.Printf("Warning: failed to parse last_accessed date: %v", err)
		}
	}

	// Parse les valeurs nullables
	if podName.Valid {
		instance.PodName = podName.String
	}
	if serviceName.Valid {
		instance.ServiceName = serviceName.String
	}
	if accessURL.Valid {
		instance.AccessURL = accessURL.String
	}
	if externalPort.Valid {
		instance.ExternalPort = int(externalPort.Int64)
	}
	if internalPort.Valid {
		instance.InternalPort = int(internalPort.Int64)
	}

	return &instance, nil
}

// DeleteInstance supprime une instance (soft delete)
func DeleteInstance(id int) error {
	_, err := DB.Exec("UPDATE instances SET status = 'deleted' WHERE id = ?", id)
	return err
}