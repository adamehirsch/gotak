package main

import (
	"database/sql"
	"log"
	// sql backend for this deployment
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// InitDB will initialize a sqlite3 db
func InitDB(dataSourceName string) {
	var err error

	db, err = sql.Open("sqlite3", dataSourceName)
	if err != nil {
		log.Panic(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	makeUsers, _ := db.Prepare("CREATE TABLE IF NOT EXISTS users (guid CHARACTER(37) PRIMARY KEY, username VARCHAR UNIQUE NOT NULL, hash VARCHAR, playedgames VARCHAR)")
	makeUsers.Exec()
	makeGames, _ := db.Prepare("CREATE TABLE IF NOT EXISTS games (guid CHARACTER(37) PRIMARY KEY, gameBlob VARCHAR)")
	makeGames.Exec()
}

// StoreTakGame puts a given game into the database
func StoreTakGame(tg *TakGame) error {

	return nil
}
