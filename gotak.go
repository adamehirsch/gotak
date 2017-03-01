package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
)

// Piece is the most basic element of a Tak game. One of two colors; one of three types.
type Piece struct {
	// one of "black" or "white"
	color string
	// Type can be one of "flat", "standing", or "capstone"
	orientation string
}

// Stack is potentially a needless piece of structure; it's just a slice of Pieces... so maybe I could/should just refer to []Piece instead of having a distinct struct for it
type Stack struct {
	// the top of the stack is at [0]
	pieces []Piece
}

// Board is an NxN grid of spaces, optionally occupied by Stacks of Pieces. A given Board has a guaranteed unique uuid
type Board struct {
	uuid uuid.UUID
	grid [][]Stack
}

// MakeGameBoard takes an integer size and returns a Board
func MakeGameBoard(size int) Board {
	// each board gets a unique, random UUIDv4
	newUUID := uuid.NewV4()

	// first make the rows...
	newGrid := make([][]Stack, size, size)

	// ... then populate with the columns of spaces
	for i := 0; i < size; i++ {
		row := make([]Stack, size, size)
		newGrid[i] = row
	}

	newBoard := Board{newUUID, newGrid}

	return newBoard
}

// NewGameHandler will generate a new board with a specified size and return the UUID by which will be known throughout its short, happy life.
func NewGameHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if boardsize, err := strconv.Atoi(vars["boardsize"]); err == nil {
		newGame := MakeGameBoard(boardsize)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Size: %v\n", vars["boardsize"])
		fmt.Fprintf(w, "UUID: %v\n", newGame.uuid)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}

}

func main() {
	// I'll need some way to keep multiple boards stored and accessible; a map between UUID and Board might be just the ticket.
	gameIndex := make(map[uuid.UUID]Board)

	testBoard := MakeGameBoard(2)
	firstPiece := Piece{"white", "flat"}
	secondPiece := Piece{"black", "flat"}
	testBoard.grid[0][0] = Stack{[]Piece{firstPiece, secondPiece}}

	gameIndex[testBoard.uuid] = testBoard
	fmt.Println(gameIndex[testBoard.uuid])

	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", SlashHandler)
	r.HandleFunc("/newgame/{boardsize}", NewGameHandler)

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}

// SlashHandler will be a slim handler to present some canned HTML for humans to read
func SlashHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("GOTAK!\n"))
}
