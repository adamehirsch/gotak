package main

import (
	"encoding/json"
	"fmt"
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

			if jBoard, err := json.Marshal(requestedGame.Grid); err == nil {
				w.Write(jBoard)

				// fmt.Fprintf(w, "requested game: %v\n", requestedGame)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "requested game not found: %v", gameID)
		}

	} else {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "requested game ID not understood: %v", gameID)

	}
}
