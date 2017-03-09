package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
)

// SlashHandler is a slim handler to present some canned HTML for humans to read
func SlashHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("GOTAK!\n"))
}

// NewGameHandler will generate a new board with a specified size and return the UUID by which it will be known throughout its short, happy life.
func NewGameHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	if boardsize, err := strconv.Atoi(vars["boardSize"]); err == nil {
		newGame := MakeGameBoard(boardsize)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "grid size: %v\n", vars["boardSize"])
		fmt.Fprintf(w, "UUID: %v\n", newGame.BoardID)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Could not understand requested board size: %v\n", vars["boardSize"])
	}

}

// ShowGameHandler takes a given UUID, looks up the game (if it exists) and returns the current grid
func ShowGameHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if gameID, err := uuid.FromString(vars["gameID"]); err == nil {

		if requestedGame, ok := gameIndex[gameID]; ok == true {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(requestedGame.Grid); err != nil {
				panic(err)
			}

		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "requested game '%v' not found.", gameID)
		}

	} else {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "requested game ID '%v' not understood.", gameID)

	}
}

// PlaceMoveHandler will accept a JSON Placement for a particular game, execute it if the space is empty, and then return the updated grid
func PlaceMoveHandler(w http.ResponseWriter, r *http.Request) {

	// read in only up to 1MB of data from the client. Come on, now.
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		log.Fatal(err)
	}
	if closeError := r.Body.Close(); closeError != nil {
		log.Fatal(closeError)
	}

	var placement Placement
	if unmarshalError := json.Unmarshal(body, &placement); unmarshalError != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if encodeErr := json.NewEncoder(w).Encode(unmarshalError); encodeErr != nil {
			panic(encodeErr)
		}
		return
	}

	vars := mux.Vars(r)
	gameID, err := uuid.FromString(vars["gameID"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "requested game ID not understood: %v", gameID)
		return
	}

	requestedGame, ok := gameIndex[gameID]
	if ok != true {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "requested game not found: %v", gameID)
		return
	}

	if err := requestedGame.PlacePiece(placement.Coords, placement.Piece); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "problem placing piece at %v: %v", placement.Coords, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(requestedGame.Grid)

}

// // testing: write back placement order
//  {
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	if err := json.NewEncoder(w).Encode(placement); err != nil {
// 		panic(err)
// 	}
// }
