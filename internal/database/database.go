package database

import (
	"database/sql"
	"errors"
	"log"

	"github.com/arimotearipo/ggmp/internal/types"
	_ "modernc.org/sqlite"
)

type Database struct {
	DB *sql.DB
	TX *sql.Tx
}

type SQLExecutor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Prepare(query string) (*sql.Stmt, error)
}

func (db *Database) executor() SQLExecutor {
	if db.TX != nil {
		return db.TX
	}

	return db.DB
}

func NewDatabase(name string) *Database {
	// connecting to database
	DB, err := sql.Open("sqlite", name)
	if err != nil {
		log.Fatal("Fail to connect to database")
	}
	log.Println("Connected to database")

	// creating master account table
	createMasterAccountSchema := `CREATE TABLE IF NOT EXISTS master_account (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"username" TEXT UNIQUE,
		"hashed_password" TEXT(60),
		"salt" BLOB
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
		"owner" REFERENCES master_account (id),
		"uri" TEXT,
		"username" TEXT,
		"password" TEXT
	);`
	statement, err = DB.Prepare(createAccountsSchema)
	if err != nil {
		log.Println("err:", err)
		log.Fatal("Fail to prepare createAccountsSchema SQL statement")
	}
	statement.Exec()
	log.Println("Account table created")

	return &Database{DB: DB, TX: nil}
}

func (db *Database) Close() {
	db.DB.Close()
}

func (db *Database) AddPassword(id int, uri, username, password string) error {
	insertAccountQuery := `INSERT INTO accounts (owner, uri, username, password) VALUES (?, ?, ?, ?);`
	statement, err := db.executor().Prepare(insertAccountQuery)
	if err != nil {
		return err
	}

	statement.Exec(id, uri, username, password)
	return nil
}

func (db *Database) GetPassword(uriId int, ownerId int) (username string, encryptedPassword string, err error) {
	selectAccountQuery := `SELECT username, password FROM accounts WHERE id = ? AND owner = ?;`
	statement, err := db.executor().Prepare(selectAccountQuery)

	if err != nil {
		return "", "", err
	}

	err = statement.QueryRow(uriId, ownerId).Scan(&username, &encryptedPassword)
	if err != nil {
		return "", "", err
	}

	return username, encryptedPassword, nil
}

func (db *Database) ListURIs(id int) ([]types.URI, error) {
	selectAccountQuery := `SELECT id, uri FROM accounts WHERE owner = ?;`
	statement, err := db.executor().Prepare(selectAccountQuery)

	if err != nil {
		return nil, err
	}

	rows, err := statement.Query(id)

	if err != nil {
		return nil, err
	}

	var uris []types.URI
	for rows.Next() {
		var uri types.URI
		rows.Scan(&uri.Id, &uri.Uri)
		uris = append(uris, uri)
	}
	return uris, nil
}

func (db *Database) DeleteAccount(uriId int, id int) error {
	deleteAccountQuery := `DELETE FROM accounts WHERE id = ? AND owner = ?;`
	statement, err := db.executor().Prepare(deleteAccountQuery)

	if err != nil {
		return err
	}

	_, err = statement.Exec(uriId, id)
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) UpdatePassword(uriId int, username string, password string) error {
	updateAccountQuery := `UPDATE accounts SET username = ?, password = ? WHERE id = ?;`
	statement, err := db.executor().Prepare(updateAccountQuery)

	if err != nil {
		return err
	}

	statement.Exec(username, password, uriId)
	return nil
}

func (db *Database) AddMasterAccount(username string, hashedPassword string, salt []byte) error {
	insertMasterAccountQuery := `INSERT INTO master_account (username, hashed_password, salt) VALUES (?, ?, ?);`
	statement, err := db.executor().Prepare(insertMasterAccountQuery)
	if err != nil {
		return err
	}

	_, err = statement.Exec(username, hashedPassword, salt)
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) GetMasterAccount(username string) (id int, hashedPassword string, salt []byte, err error) {
	selectMasterAccountQuery := `SELECT COUNT(*) FROM master_account WHERE username = ?;`
	statement, err := db.executor().Prepare(selectMasterAccountQuery)
	if err != nil {
		return -1, "", nil, err
	}

	var count int
	statement.QueryRow(username).Scan(&count)
	if count == 0 {
		return -1, "", nil, errors.New("username does not exist")
	}

	selectMasterAccountQuery = `SELECT id, hashed_password, salt FROM master_account WHERE username = ?;`
	statement, err = db.executor().Prepare(selectMasterAccountQuery)
	if err != nil {
		return -1, "", nil, err
	}

	statement.QueryRow(username).Scan(&id, &hashedPassword, &salt)
	return id, hashedPassword, salt, nil
}

func (db *Database) DeleteMasterAccount(username string) error {
	deleteLoginsQuery := `DELETE FROM accounts WHERE owner = (SELECT id FROM master_account WHERE username = ?);`
	statement, err := db.executor().Prepare(deleteLoginsQuery)
	if err != nil {
		return err
	}
	_, err = statement.Exec(username)
	if err != nil {
		return err
	}

	deleteMasterAccountQuery := `DELETE FROM master_account WHERE username = ?;`
	statement, err = db.executor().Prepare(deleteMasterAccountQuery)
	if err != nil {
		return err
	}
	_, err = statement.Exec(username)
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) ListMasterAccounts() ([]string, error) {
	selectMasterAccountQuery := `SELECT username FROM master_account;`
	statement, err := db.executor().Prepare(selectMasterAccountQuery)
	if err != nil {
		return nil, err
	}

	rows, err := statement.Query()
	if err != nil {
		return nil, err
	}

	var usernames []string
	for rows.Next() {
		var username string
		rows.Scan(&username)
		usernames = append(usernames, username)
	}
	return usernames, nil
}

func (db *Database) GetUserId(username string) (int, error) {
	query := "SELECT id FROM master_account WHERE username = ?;"

	statement, err := db.executor().Prepare(query)
	if err != nil {
		return 0, err
	}

	var id int
	err = statement.QueryRow(username).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (db *Database) ChangeMasterPassword(userId int, hashedPassword string, salt []byte) error {
	updateMasterPasswordQuery := `UPDATE master_account SET hashed_password = ?, salt = ? WHERE id = ?;`

	stmt, err := db.executor().Prepare(updateMasterPasswordQuery)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(hashedPassword, salt, userId)
	if err != nil {
		return err
	}

	return nil
}
