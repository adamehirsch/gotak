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

var db *sql.DB

// InitDB will initialize a sqlite3 db
func InitDB(dataSourceName string) error {
	var err error
	db, err = sql.Open("sqlite3", dataSourceName)
	if err != nil {
		log.Panic(err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}
	if _, err = db.Exec("CREATE TABLE IF NOT EXISTS players (guid BLOB(16) PRIMARY KEY, username VARCHAR UNIQUE NOT NULL, hash VARCHAR, playedgames VARCHAR)"); err != nil {
		return err
	}
	if _, err = db.Exec("CREATE TABLE IF NOT EXISTS games (guid BLOB(16) PRIMARY KEY UNIQUE, isOver BOOL, isPublic BOOL, hasStarted BOOL, gameBlob VARCHAR)"); err != nil {
		return err
	}
	return nil
}

// StoreTakGame puts a given game into the database
func StoreTakGame(tg *TakGame) error {
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

// StorePlayer puts a given player into the database
func StorePlayer(p *TakPlayer) error {
	pg, _ := json.Marshal(p.PlayedGames)

	// this clever little two step handles INSERT OR UPDATE in sqlite3 so that one can store an existing player or have it update an existing one
	// http://stackoverflow.com/questions/15277373/sqlite-upsert-update-or-insert

	db.Exec("UPDATE players SET guid=?, username=?, hash=?, playedGames=? WHERE guid=?", p.PlayerID, p.Name, p.PasswordHash, pg, p.PlayerID)
	_, err := db.Exec("INSERT INTO players(guid, username, hash, playedGames) SELECT ?, ?, ?, ? WHERE (SELECT CHANGES() = 0)", p.PlayerID, p.Name, p.PasswordHash, pg)

	if err != nil {
		return err
	}
	return nil
}

// RetrievePlayerByName gets a player from the db
func RetrievePlayerByName(name string) (*TakPlayer, error) {
	var (
		player      TakPlayer
		playedGames sql.NullString
		npg         []uuid.UUID
	)

	queryErr := db.QueryRow("SELECT * from players WHERE username = ?", name).Scan(&player.PlayerID, &player.Name, &player.PasswordHash, &playedGames)

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

// DeleteTakGame deletes a game from the db
func DeleteTakGame(id uuid.UUID) error {
	_, err := db.Exec("DELETE FROM games WHERE guid=?", id)

	switch {
	case err == sql.ErrNoRows:
		return errors.New("No such game found")
	case err != nil:
		// problem with running the query? Yell.
		return err
	default:
		return nil
	}

}
