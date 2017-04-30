package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	// sql backend for this deployment
	_ "github.com/mattn/go-sqlite3"
	uuid "github.com/satori/go.uuid"
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
	makeUsers, _ := db.Prepare("CREATE TABLE IF NOT EXISTS users (guid BLOB(16) PRIMARY KEY, username VARCHAR UNIQUE NOT NULL, hash VARCHAR, playedgames VARCHAR)")
	makeUsers.Exec()
	makeGames, _ := db.Prepare("CREATE TABLE IF NOT EXISTS games (guid BLOB(16) PRIMARY KEY UNIQUE, gameBlob VARCHAR)")
	makeGames.Exec()
}

// StoreTakGame puts a given game into the database
func StoreTakGame(tg *TakGame) error {
	textGame, _ := json.Marshal(tg)
	// this clever little two step handles INSERT OR UPDATE in sqlite3
	// http://stackoverflow.com/questions/15277373/sqlite-upsert-update-or-insert
	db.Exec("UPDATE games SET guid=?, gameBlob=? WHERE guid=?", tg.GameID, textGame, tg.GameID)
	_, err := db.Exec("INSERT INTO games(guid, gameBlob) SELECT ?, ? WHERE (SELECT CHANGES() = 0)", tg.GameID, textGame)

	if err != nil {
		return err
	}
	return nil
}

// RetrieveTakGame gets a game from the db
func RetrieveTakGame(id uuid.UUID) (*TakGame, error) {
	var gameBlob string
	queryErr := db.QueryRow("SELECT gameBlob from games WHERE guid = ?", id).Scan(&gameBlob)
	switch {
	case queryErr == sql.ErrNoRows:
		return nil, errors.New("No such game found")
	case queryErr != nil:
		// problem with running the query? Yell.
		log.Fatal(queryErr)
	}
	retrievedGame := TakGame{}
	if unmarshalError := json.Unmarshal([]byte(gameBlob), &retrievedGame); unmarshalError != nil {
		return nil, errors.New("Problem decoding JSON")
	}
	return &retrievedGame, nil
}
