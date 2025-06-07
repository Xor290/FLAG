package models

import "time"

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Ne pas exposer le hash
	Role         string    `json:"role"`
	Score        int       `json:"score"`
	CreatedAt    time.Time `json:"created_at"`
	LastLogin    time.Time `json:"last_login,omitempty"`
}

// models/challenge.go
type Challenge struct {
	ID          int    `json:"id"`
	Name        string `json:"name_chall"`
	Description string `json:"description"`
	DockerImage string `json:"docker_image"`
	Port        int    `json:"port"`
	CPULimit    string `json:"cpu_limit"`
	MemoryLimit string `json:"memory_limit"`
	CreatedAt   string `json:"created_at"`
	Category    string `json:"category"`
	Points      int    `json:"points"`
}



type Instance struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	ChallengeID  int       `json:"challenge_id"`
	ContainerName string   `json:"container_name"`
	PodName      string    `json:"pod_name,omitempty"`
	ServiceName  string    `json:"service_name,omitempty"`
	Namespace    string    `json:"namespace"`
	ExternalPort int       `json:"external_port,omitempty"`
	InternalPort int       `json:"internal_port,omitempty"`
	AccessURL    string    `json:"access_url,omitempty"`
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"started_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	LastAccessed time.Time `json:"last_accessed,omitempty"`
	Challenge    *Challenge `json:"challenge,omitempty"`
}

type Submission struct {
	ID            int       `json:"id"`
	UserID        int       `json:"user_id"`
	ChallengeID   int       `json:"challenge_id"`
	InstanceID    *int      `json:"instance_id,omitempty"`
	SubmittedFlag string    `json:"submitted_flag"`
	IsValid       bool      `json:"is_valid"`
	PointsAwarded int       `json:"points_awarded"`
	SubmittedAt   time.Time `json:"submitted_at"`
	Challenge     *Challenge `json:"challenge,omitempty"`
	User          *User     `json:"user,omitempty"`
}

type ChallengeSolve struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	ChallengeID int       `json:"challenge_id"`
	SolvedAt    time.Time `json:"solved_at"`
	User        *User     `json:"user,omitempty"`
	Challenge   *Challenge `json:"challenge,omitempty"`
}

type UserSession struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	SessionToken string    `json:"session_token"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// DTOs pour les requêtes API
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=20"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type CreateInstanceRequest struct {
	ChallengeID int `json:"challenge_id" binding:"required"`
}

type SubmitFlagRequest struct {
	Flag string `json:"flag" binding:"required"`
}

type CreateChallengeRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
	Category    string `json:"category" binding:"required"`
	Difficulty  string `json:"difficulty" binding:"required,oneof=easy medium hard"`
	Points      int    `json:"points" binding:"required,min=1"`
	Flag        string `json:"flag" binding:"required"`
	DockerImage string `json:"docker_image" binding:"required"`
	Port        int    `json:"port" binding:"required"`
	CPULimit    string `json:"cpu_limit"`
	MemoryLimit string `json:"memory_limit"`
	TimeLimit   int    `json:"time_limit"`
}

type UpdateChallengeRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Difficulty  string `json:"difficulty" binding:"omitempty,oneof=easy medium hard"`
	Points      int    `json:"points" binding:"omitempty,min=1"`
	Flag        string `json:"flag"`
	DockerImage string `json:"docker_image"`
	Port        int    `json:"port"`
	CPULimit    string `json:"cpu_limit"`
	MemoryLimit string `json:"memory_limit"`
	TimeLimit   int    `json:"time_limit"`
	IsActive    *bool  `json:"is_active"`
}

type LeaderboardEntry struct {
	Rank     int    `json:"rank"`
	Username string `json:"username"`
	Score    int    `json:"score"`
	Solves   int    `json:"solves"`
}

type PlatformStats struct {
	TotalUsers      int `json:"total_users"`
	TotalChallenges int `json:"total_challenges"`
	ActiveInstances int `json:"active_instances"`
	TotalSolves     int `json:"total_solves"`
}