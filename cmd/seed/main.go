package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"

	"github.com/arimotearipo/ggmp/internal/encryption"
)

const ACCOUNTS_LENGTH = 23
const MASTER_LOGIN = "123"
const MASTER_PASSWORD = "123"

func main() {
	// Open the database
	fmt.Println("Opening database...")
	db, err := sql.Open("sqlite", "ggmp.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	fmt.Println("Database opened successfully")
	defer db.Close()

	// Create tables if they don't exist
	fmt.Println("Creating tables...")
	err = createTables(db)
	if err != nil {
		log.Fatal("Failed to create tables:", err)
	}
	fmt.Println("Tables created successfully")

	// Seed master account
	fmt.Println("Seeding master account...")
	err = seedMasterAccount(db)
	if err != nil {
		log.Fatal("Failed to seed master account:", err)
	}

	// Seed accounts
	fmt.Println("Seeding accounts...")
	err = seedAccounts(db)
	if err != nil {
		log.Fatal("Failed to seed accounts:", err)
	}
}

func createTables(db *sql.DB) error {
	// Create master_account table
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS master_account (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"username" TEXT UNIQUE,
		"hashed_password" TEXT(60),
		"salt" BLOB
	)`)
	if err != nil {
		return fmt.Errorf("failed to create master_account table: %v", err)
	}

	// Create accounts table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS accounts (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"owner" INTEGER REFERENCES master_account (id),
		"uri" TEXT,
		"username" TEXT,
		"password" TEXT
	)`)
	if err != nil {
		return fmt.Errorf("failed to create accounts table: %v", err)
	}

	fmt.Println("Tables created successfully")
	return nil
}

func seedMasterAccount(db *sql.DB) error {
	// Check if master account already exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM master_account WHERE username = ?", MASTER_LOGIN).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		fmt.Println("Master account already exists, skipping creation")
		return nil
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(MASTER_PASSWORD), 12)
	if err != nil {
		return err
	}

	// Generate salt
	salt, err := encryption.GenerateSalt()
	if err != nil {
		return err
	}

	// Insert master account
	_, err = db.Exec("INSERT INTO master_account (username, hashed_password, salt) VALUES (?, ?, ?)",
		MASTER_LOGIN, hashedPassword, salt)
	if err != nil {
		return err
	}

	fmt.Println("Master account seeded successfully")
	return nil
}

func generateRandomText() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length := rand.Intn(4) + 5 // Random length between 1 and 9

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}

	return string(result)
}

func seedAccounts(db *sql.DB) error {
	// Get master account details
	var id int
	var salt []byte
	err := db.QueryRow("SELECT id, salt FROM master_account WHERE username = ?", MASTER_LOGIN).Scan(&id, &salt)
	if err != nil {
		return err
	}

	// Derive master key
	masterKey := encryption.DeriveKey([]byte(MASTER_PASSWORD), salt)

	for i := 0; i < ACCOUNTS_LENGTH; i++ {
		uri := generateRandomText()
		username := generateRandomText()
		password := generateRandomText()

		encryptedURI, err := encryption.Encrypt(uri, masterKey)
		if err != nil {
			return err
		}
		encryptedUsername, err := encryption.Encrypt(username, masterKey)
		if err != nil {
			return err
		}
		encryptedPassword, err := encryption.Encrypt(password, masterKey)
		if err != nil {
			return err
		}

		_, err = db.Exec("INSERT INTO accounts (owner, uri, username, password) VALUES (?, ?, ?, ?)",
			id, encryptedURI, encryptedUsername, encryptedPassword)
		if err != nil {
			return err
		}

		fmt.Printf("\rSeeded %d of %d accounts", i+1, ACCOUNTS_LENGTH)
	}

	fmt.Printf("\nAll %d accounts seeded successfully\n", ACCOUNTS_LENGTH)
	return nil
}
