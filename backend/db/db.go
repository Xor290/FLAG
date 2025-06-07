package db

import (
	"backend/models"
	"database/sql"
	"errors"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

// InitDB ouvre la connexion et crée les tables si elles n'existent pas
func InitDB() {
	var err error
	DB, err = sql.Open("sqlite3", "./ctf.sqlite")
	if err != nil {
		log.Fatal("Failed to open DB:", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}

	createTables()
	log.Println("CTF Database initialized.")
}

// CloseDB ferme la connexion à la base de données
func CloseDB() {
	if DB != nil {
		err := DB.Close()
		if err != nil {
			log.Printf("Error closing DB: %v\n", err)
		} else {
			log.Println("Database closed.")
		}
	}
}

// createTables crée les tables nécessaires pour la plateforme CTF
func createTables() {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT DEFAULT 'user',
			score INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_login DATETIME
		);`,

		`CREATE TABLE IF NOT EXISTS challenges (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL,
			category TEXT NOT NULL,
			difficulty TEXT NOT NULL,
			points INT NOT NULL,
			flag TEXT NOT NULL,
			docker_image TEXT NOT NULL,
			port INTEGER NOT NULL,
			cpu_limit TEXT DEFAULT '0.5',
			memory_limit TEXT DEFAULT '512Mi',
			time_limit INTEGER DEFAULT 3600,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (created_by) REFERENCES users(id)
		);`,

		`CREATE TABLE IF NOT EXISTS instances (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			challenge_id INTEGER NOT NULL,
			container_name TEXT NOT NULL,
			pod_name TEXT,
			service_name TEXT,
			namespace TEXT DEFAULT 'ctf-instances',
			external_port INTEGER,
			internal_port INTEGER,
			access_url TEXT,
			status TEXT DEFAULT 'creating',
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME,
			last_accessed DATETIME,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (challenge_id) REFERENCES challenges(id)
		);`,

		`CREATE TABLE IF NOT EXISTS submissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			challenge_id INTEGER NOT NULL,
			instance_id INTEGER,
			submitted_flag TEXT NOT NULL,
			is_valid BOOLEAN NOT NULL,
			points_awarded INTEGER DEFAULT 0,
			submitted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (challenge_id) REFERENCES challenges(id),
			FOREIGN KEY (instance_id) REFERENCES instances(id)
		);`,

		`CREATE TABLE IF NOT EXISTS challenge_solves (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			challenge_id INTEGER NOT NULL,
			solved_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, challenge_id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (challenge_id) REFERENCES challenges(id)
		);`,

		`CREATE TABLE IF NOT EXISTS user_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			session_token TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,
	}

	for _, table := range tables {
		_, err := DB.Exec(table)
		if err != nil {
			log.Printf("Error creating table: %v\n", err)
		}
	}

	// Créer un utilisateur admin par défaut
	createDefaultAdmin()
	insertDefaultChallenges()
}

func insertDefaultChallenges() {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM challenges").Scan(&count)
	if err != nil {
		log.Printf("Error checking challenge count: %v", err)
		return
	}

	if count > 0 {
		log.Println("Challenges already exist, skipping default insertion.")
		return
	}

	challenges := []struct {
		Name        string
		Description string
		Category    string
		Difficulty  string
		Points      int
		Flag        string
		Image       string
		Port        int
	}{
		{
			Name:        "XSS Reflective",
			Description: "Trouvez une faille XSS réfléchie dans ce mini site web.",
			Category:    "Web",
			Difficulty:  "Easy",
			Points:      75,
			Flag:        "CTF{xss_reflected_pwned}",
			Image:       "xss-vuln", // ou "utilisateur/xss-vuln:latest" si hébergée sur Docker Hub
			Port:        80,
		},
		{
			Name:        "CSRF Reflective",
			Description: "Trouvez une faille CSRF réfléchie dans ce mini site web.",
			Category:    "Web",
			Difficulty:  "Easy",
			Points:      75,
			Flag:        "CTF{csrf_reflected_pwned}",
			Image:       "xss-vuln", // ou "utilisateur/xss-vuln:latest" si hébergée sur Docker Hub
			Port:        81,
		},
	}

	for _, c := range challenges {
		_, err := DB.Exec(`
			INSERT INTO challenges (name, description, category, difficulty, points, flag, docker_image, port)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			c.Name, c.Description, c.Category, c.Difficulty, c.Points, c.Flag, c.Image, c.Port)
		if err != nil {
			log.Printf("Error inserting challenge '%s': %v", c.Name, err)
		} else {
			log.Printf("Challenge '%s' added successfully.", c.Name)
		}
	}
}


// createDefaultAdmin crée un utilisateur admin par défaut
func createDefaultAdmin() {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
	if err != nil {
		log.Printf("Error checking admin users: %v", err)
		return
	}

	if count == 0 {
		// Hash du mot de passe "admin123"
		adminHash := "$2a$10$8K1p/a9g4fDH8k1qD6dJI.vN9lqCjHZr8X1eqQXWJYvnLq8P3P0PW"
		_, err := DB.Exec(`INSERT INTO users (username, email, password_hash, role) 
			VALUES ('admin', 'admin@ctf.local', ?, 'admin')`, adminHash)
		if err != nil {
			log.Printf("Error creating default admin: %v", err)
		} else {
			log.Println("Default admin user created (admin/admin123)")
		}
	}
}

// GetUserByUsername récupère un utilisateur par son username
func GetUserByUsername(username string) (*models.User, error) {
	row := DB.QueryRow(`SELECT id, username, email, password_hash, role, score, created_at, last_login 
		FROM users WHERE username = ?`, username)
	var user models.User
	var createdAtStr string
	var lastLoginStr sql.NullString

	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, 
		&user.Role, &user.Score, &createdAtStr, &lastLoginStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Parse dates
	user.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
	if err != nil {
		log.Printf("Warning: failed to parse created_at date: %v", err)
	}

	if lastLoginStr.Valid {
		user.LastLogin, err = time.Parse("2006-01-02 15:04:05", lastLoginStr.String)
		if err != nil {
			log.Printf("Warning: failed to parse last_login date: %v", err)
		}
	}

	return &user, nil
}

// InsertUser ajoute un nouvel utilisateur
func InsertUser(user *models.User) error {
	stmt, err := DB.Prepare(`INSERT INTO users (username, email, password_hash, role) 
		VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	result, err := stmt.Exec(user.Username, user.Email, user.PasswordHash, user.Role)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	user.ID = int(id)
	return nil
}

// UpdateUserLastLogin met à jour la dernière connexion
func UpdateUserLastLogin(userID int) error {
	_, err := DB.Exec("UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = ?", userID)
	return err
}

// GetUserByID récupère un utilisateur par son ID
func GetUserByID(id int) (*models.User, error) {
	row := DB.QueryRow(`SELECT id, username, email, password_hash, role, score, created_at, last_login 
		FROM users WHERE id = ?`, id)
	var user models.User
	var createdAtStr string
	var lastLoginStr sql.NullString

	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, 
		&user.Role, &user.Score, &createdAtStr, &lastLoginStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Parse dates
	user.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAtStr)
	if err != nil {
		log.Printf("Warning: failed to parse created_at date: %v", err)
	}

	if lastLoginStr.Valid {
		user.LastLogin, err = time.Parse("2006-01-02 15:04:05", lastLoginStr.String)
		if err != nil {
			log.Printf("Warning: failed to parse last_login date: %v", err)
		}
	}

	return &user, nil
}

// GetAllChallenges récupère tous les challenges actifs
func GetAllChallenges() ([]models.Challenge, error) {
	rows, err := DB.Query(`SELECT id, name, description, docker_image, port, cpu_limit, memory_limit, created_at 
		FROM challenges WHERE is_active = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var challenges []models.Challenge
	for rows.Next() {
		var challenge models.Challenge
		var createdAtStr string
		
		err := rows.Scan(&challenge.ID, &challenge.Name, &challenge.Description, 
			&challenge.DockerImage, &challenge.Port, &challenge.CPULimit, 
			&challenge.MemoryLimit, &createdAtStr)
		if err != nil {
			log.Printf("Error scanning challenge: %v", err)
			continue
		}
		
		challenge.CreatedAt = createdAtStr
		challenges = append(challenges, challenge)
	}

	return challenges, nil
}