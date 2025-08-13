package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

type User struct {
	ClientId      string
	ApiKey        string
	AllowedOrigin string
}

func InitDB() {
	var err error
	DB, err = sql.Open("sqlite", "./clients/clients.db")
	if err != nil {
		panic(err)
	}
	_, err = DB.Exec(`
		CREATE TABLE IF NOT EXISTS clients (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id TEXT UNIQUE NOT NULL,
			api_key TEXT NOT NULL,
			allowed_origin TEXT
		)`,
	)

	if err != nil {
		panic(err)
	}
}

func Close() {
	DB.Close()
}

func CheckAuth(clientID, apiKey string) (string, error) {
	var allowedOrigin string
	err := DB.QueryRow(
		`SELECT allowed_origin FROM clients WHERE client_id = ? AND api_key = ?`,
		clientID, apiKey,
	).Scan(&allowedOrigin)

	if err != nil {
		return "", err
	}

	return allowedOrigin, nil
}

func CheckOrigin(clientID string) (string, error) {
	var allowedOrigin string
	err := DB.QueryRow(
		`SELECT allowed_origin FROM clients WHERE client_id = ?`,
		clientID,
	).Scan(&allowedOrigin)

	if err != nil {
		return "", err
	}

	return allowedOrigin, nil
}

func AddUser(clientID, apiKey, origin string) error {
	_, err := DB.Exec(`
		INSERT INTO clients (client_id, api_key, allowed_origin) 
		VALUES (?, ?, ?)`,
		clientID, apiKey, origin)

	if err != nil {
		return err
	}

	return nil
}

func AllUsers() ([]User, error) {
	rows, err := DB.Query("SELECT client_id, api_key, allowed_origin FROM clients")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		rows.Scan(&user.ClientId, &user.ApiKey, &user.AllowedOrigin)
		users = append(users, user)
	}

	return users, nil
}

func DeleteUser(clientID string) error {
	_, err := DB.Exec("DELETE FROM clients WHERE client_id = ?", clientID)
	if err != nil {
		return err
	}

	return nil
}
