package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	// sql backend for this deployment
	_ "github.com/mattn/go-sqlite3"
	uuid "github.com/satori/go.uuid"
)

// Datastore contains any methods that are going to touch the backend database
// Cool technique, inspired from http://www.alexedwards.net/blog/organising-database-access
type Datastore interface {
	StoreTakGame(tg *TakGame) error
	RetrieveTakGame(id uuid.UUID) (*TakGame, error)
	StorePlayer(p *TakPlayer) error
	RetrievePlayer(name string) (*TakPlayer, error)
	PlayerExists(n string) bool
}

// DB is simply a self-contained
type DB struct {
	*sql.DB
}

// InitSQLiteDB will initialize a sqlite3 db
func InitSQLiteDB(dataSourceName string) (*DB, error) {
	var err error
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		log.Panic(err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}
	if _, err = db.Exec("CREATE TABLE IF NOT EXISTS players (guid BLOB(16) PRIMARY KEY, username VARCHAR UNIQUE NOT NULL, hash VARCHAR, playedgames VARCHAR)"); err != nil {
		return nil, err
	}
	if _, err = db.Exec("CREATE TABLE IF NOT EXISTS games (guid BLOB(16) PRIMARY KEY UNIQUE, isOver BOOL, isPublic BOOL, hasStarted BOOL, gameBlob VARCHAR)"); err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

// StoreTakGame puts a given game into the database
func (db *DB) StoreTakGame(tg *TakGame) error {
	textGame, _ := json.Marshal(tg)
	// this clever little two step handles INSERT OR UPDATE in sqlite3 so that one can store an existing game and have it update the row in the db
	// http://stackoverflow.com/questions/15277373/sqlite-upsert-update-or-insert
	db.Exec("UPDATE games SET guid=?, isOver=?, isPublic=?, hasStarted=?, gameBlob=? WHERE guid=?", tg.GameID, tg.GameOver, tg.IsPublic, tg.HasStarted, textGame, tg.GameID)
	_, err := db.Exec("INSERT INTO games(guid, isOver, isPublic, hasStarted, gameBlob) SELECT ?, ?, ?, ?, ? WHERE (SELECT CHANGES() = 0)", tg.GameID, tg.GameOver, tg.IsPublic, tg.HasStarted, textGame)

	if err != nil {
		return err
	}
	return nil
}

// RetrieveTakGame gets a game from the db
func (db *DB) RetrieveTakGame(id uuid.UUID) (*TakGame, error) {
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

// StorePlayer puts a given player into the database
func (db *DB) StorePlayer(p *TakPlayer) error {
	pg, _ := json.Marshal(p.PlayedGames)

	// this clever little two step handles INSERT OR UPDATE in sqlite3 so that one can store an existing player or have it update an existing one
	// http://stackoverflow.com/questions/15277373/sqlite-upsert-update-or-insert

	db.Exec("UPDATE players SET guid=?, username=?, hash=?, playedGames=? WHERE guid=?", p.PlayerID, p.Username, p.passwordHash, pg, p.PlayerID)
	_, err := db.Exec("INSERT INTO players(guid, username, hash, playedGames) SELECT ?, ?, ?, ? WHERE (SELECT CHANGES() = 0)", p.PlayerID, p.Username, p.passwordHash, pg)

	if err != nil {
		return err
	}
	return nil
}

// RetrievePlayer gets a player from the db by name
func (db *DB) RetrievePlayer(name string) (*TakPlayer, error) {
	var (
		player      TakPlayer
		playedGames sql.NullString
		npg         []uuid.UUID
	)

	queryErr := db.QueryRow("SELECT * from players WHERE username = ?", name).Scan(&player.PlayerID, &player.Username, &player.passwordHash, &playedGames)

	switch {
	case queryErr == sql.ErrNoRows:
		return nil, errors.New("No such player found")
	case queryErr != nil:
		// problem with running the query? Yell.
		log.Fatal(queryErr)
	}

	// json.Unmarshal did unexpected things when presented with an empty column. workaround.
	if playedGames.String != "" {
		if unmarshalError := json.Unmarshal([]byte(playedGames.String), &npg); unmarshalError != nil {
			return nil, fmt.Errorf("problem decoding played games: %v", playedGames.String)
		}
		player.PlayedGames = npg

	}
	return &player, nil
}

// PlayerExists checks to see if a username is already taken
func (db *DB) PlayerExists(n string) bool {
	// check to see if the name conflicts in the DB
	var matchName string

	queryErr := db.QueryRow("SELECT username FROM players WHERE username = ?", n).Scan(&matchName)
	if queryErr == sql.ErrNoRows {
		// that's what we want to see: no rows.
		return false
	}
	return true

}
