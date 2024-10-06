package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	DB *sql.DB
}

func NewDatabase(name string) *Database {
	// connecting to database
	DB, err := sql.Open("sqlite3", name)
	if err != nil {
		log.Fatal("Fail to connect to database")
	}
	log.Println("Connected to database")

	// creating master account table
	createMasterAccountSchema := `CREATE TABLE IF NOT EXISTS master_account (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"username" TEXT,
		"hashed_password" TEXT,
		"initialization_vector" BLOB NOT NULL,
		"salt" BLOB NOT NULL
	);`
	statement, err := DB.Prepare(createMasterAccountSchema)
	if err != nil {
		log.Println("err:", err)
		log.Fatal("Fail to prepare createMasterAccountSchema SQL statement")
	}
	statement.Exec()
	log.Println("Master account table created")

	// create accounts table
	createAccountsSchema := `CREATE TABLE IF NOT EXISTS accounts (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"uri" TEXT,
		"username" TEXT,
		"hashed_password" TEXT
	);`
	statement, err = DB.Prepare(createAccountsSchema)
	if err != nil {
		log.Println("err:", err)
		log.Fatal("Fail to prepare createAccountsSchema SQL statement")
	}
	statement.Exec()
	log.Println("Account table created")

	return &Database{DB}
}

func (db *Database) Close() {
	db.DB.Close()
}

func (db *Database) AddPassword(uri string, username string, encryptedPassword string) error {
	insertAccountQuery := `INSERT INTO accounts (uri, username, hashed_password) VALUES (?, ?, ?);`
	statement, err := db.DB.Prepare(insertAccountQuery)

	if err != nil {
		return err
	}

	statement.Exec(uri, username, encryptedPassword)
	return nil
}

func (db *Database) GetPassword(uri string) (string, error) {
	selectAccountQuery := `SELECT username, hashed_password FROM accounts WHERE uri = ?;`
	statement, err := db.DB.Prepare(selectAccountQuery)

	if err != nil {
		return "", err
	}

	var username string
	var encryptedPassword string
	statement.QueryRow(uri).Scan(&username, &encryptedPassword)
	return encryptedPassword, nil
}

func (db *Database) ListURIs() ([]string, error) {
	selectAccountQuery := `SELECT uri FROM accounts;`
	statement, err := db.DB.Prepare(selectAccountQuery)

	if err != nil {
		return nil, err
	}

	rows, err := statement.Query()

	if err != nil {
		return nil, err
	}

	var uris []string
	for rows.Next() {
		var uri string
		rows.Scan(&uri)
		uris = append(uris, uri)
	}
	return uris, nil
}

func (db *Database) DeleteAccount(uri string) error {
	deleteAccountQuery := `DELETE FROM accounts WHERE uri = ?;`
	statement, err := db.DB.Prepare(deleteAccountQuery)

	if err != nil {
		return err
	}

	statement.Exec(uri)
	return nil
}

func (db *Database) UpdatePassword(uri string, username string, encryptedPassword string) error {
	updateAccountQuery := `UPDATE accounts SET username = ?, hashed_password = ? WHERE uri = ?;`
	statement, err := db.DB.Prepare(updateAccountQuery)

	if err != nil {
		return err
	}

	statement.Exec(username, encryptedPassword, uri)
	return nil
}

func (db *Database) AddMasterAccount(username string, hashedPassword string, initializationVector, salt []byte) error {
	insertMasterAccountQuery := `INSERT INTO master_account (username, hashed_password, initialization_vector, salt) VALUES (?, ?, ?, ?);`
	statement, err := db.DB.Prepare(insertMasterAccountQuery)
	if err != nil {
		fmt.Println(err)
		return err
	}

	statement.Exec(username, hashedPassword, initializationVector, salt)
	return nil
}

func (db *Database) GetMasterAccount(username string) (string, []byte, []byte, error) {
	selectMasterAccountQuery := `SELECT COUNT(*) FROM master_account WHERE username = ?;`
	statement, err := db.DB.Prepare(selectMasterAccountQuery)
	if err != nil {
		fmt.Println(err)
		return "", nil, nil, err
	}

	var count int
	statement.QueryRow(username).Scan(&count)
	if count == 0 {
		return "", nil, nil, errors.New("username does not exist")
	}

	selectMasterAccountQuery = `SELECT hashed_password, initialization_vector, salt FROM master_account WHERE username = ?;`
	statement, err = db.DB.Prepare(selectMasterAccountQuery)
	if err != nil {
		fmt.Println(err)
		return "", nil, nil, err
	}

	var hashedPassword string
	var initializationVector, salt []byte
	statement.QueryRow(username).Scan(&hashedPassword, &initializationVector, salt)
	return hashedPassword, initializationVector, salt, nil
}
