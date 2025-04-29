package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"rateLimiting/pkg/token"

	_ "github.com/lib/pq"
)

var (
	ErrCantWriteInDB    = errors.New("ошибка при записи в БД")
	ErrCantDeleteFromDB = errors.New("ошибка при удалении записи в БД")
)

type DB struct {
	Db *sql.DB
}

func NewDB(dbUser, dbPassword, dbName, dbHost, dbPort string) *DB {
	db, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", dbHost, dbPort, dbUser, dbPassword, dbName))
	if err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}

	return &DB{Db: db}
}

func (db *DB) UpdateOrInsertClient(clientIP string, capacity, refillRate float64) error {
	query := `
		INSERT INTO clients_info (client_ip, capacity, rate)
		VALUES ($1, $2, $3)
		ON CONFLICT (client_ip)
		DO UPDATE SET capacity = EXCLUDED.capacity, rate = EXCLUDED.rate;
	`
	_, err := db.Db.Exec(query, clientIP, capacity, refillRate)
	if err != nil {
		return ErrCantWriteInDB
	}

	return nil
}

func (db *DB) DeleteClient(clientIP string) error {
	query := `
		DELETE FROM clients_info
		WHERE client_ip = $1;
	`
	_, err := db.Db.Exec(query, clientIP)
	if err != nil {
		return ErrCantDeleteFromDB
	}
	return nil

}

func (db *DB) LoadClientsFromDB(rateLimiter *token.RateLimiter) error {
	rows, err := db.Db.Query("SELECT client_ip, capacity, rate FROM clients_info")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var clientIP string
		var capacity, rate float64
		if err := rows.Scan(&clientIP, &capacity, &rate); err != nil {
			return err
		}
		rateLimiter.GetOrCreateBucket(clientIP, capacity, rate)
	}

	return rows.Err()
}
