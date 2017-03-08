package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
)

// Piece is the most basic element of a Tak game. One of two colors; one of three types.
type Piece struct {
	// one of "black" or "white"
	Color string `json:"color"`
	// Type can be one of "flat", "standing", or "capstone"
	Orientation string `json:"orientation"`
}

// Stack is potentially a needless piece of structure; it's just a slice of Pieces... so maybe I could/should just refer to []Piece instead of having a distinct struct for it
type Stack struct {
	// the top of the stack is at [0]
	Pieces []Piece
}

// Board is an NxN grid of spaces, optionally occupied by Stacks of Pieces. A given Board has a guaranteed unique uuid
type Board struct {
	BoardID uuid.UUID
	Grid    [][]Stack
}

// I'll need some way to keep multiple boards stored and accessible; a map between UUID and Board might be just the ticket.
var gameIndex = make(map[uuid.UUID]Board)

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

	newBoard := Board{BoardID: newUUID, Grid: newGrid}

	gameIndex[newUUID] = newBoard
	return newBoard
}

// LetterMap converts Tak files to their index value
var LetterMap = map[string]int{
	"a": 0,
	"b": 1,
	"c": 2,
	"d": 3,
	"e": 4,
	"f": 5,
	"g": 6,
	"h": 7,
	"i": 8,
	"j": 9,
	"k": 10,
	"l": 11,
	"m": 12,
}

// Place descripts the necessary aspects to describe an action that places a new piece on the board
type Place struct {
	Piece  Piece  `json:"piece"`
	Coords string `json:"coords"`
}

// Move contains the necessary aspects to describe an action that moves a stack.
type Move struct {
	Coords     string `json:"coords"`
	Direction  string `json:"direction"`
	Carry      int    `json:"carry"`
	Deliveries []int  `json:"deliveries"`
}

// CheckSquare looks at a given spot on a given board and returns either a Stack, nil, or an err
func (board Board) CheckSquare(coords string) (Stack, error) {
	grid := board.Grid

	// looking for coordinates in the form LetterNumber
	r := regexp.MustCompile("^([a-zA-Z])([1-9]|[12][0-9])$")

	validcoords := r.FindAllStringSubmatch(coords, -1)

	if len(validcoords) > 0 {

		// Assuming we've got a valid looking set of coordinates, look them up on the provided board
		rank := LetterMap[validcoords[0][1]]
		file, err := strconv.Atoi(validcoords[0][2])

		switch {
		case err != nil:
			return Stack{}, errors.New("cannot interpret coordinates")
		case rank >= len(grid) || file-1 >= len(grid):
			return Stack{}, fmt.Errorf("coordinates '%v' larger than board size: %v", validcoords[0][0], len(grid))
		}

		// arrays start with [0], so subtract one from the human-readable rank
		file = file - 1

		foundStack := grid[rank][file]
		return foundStack, nil
	}
	return Stack{}, fmt.Errorf("Could not interpret coordinates '%v'", coords)
}

// SquareIsEmpty returns a simple boolean to test if a square is empty
func (board Board) SquareIsEmpty(coords string) (bool, error) {
	if foundStack, err := board.CheckSquare(coords); err == nil {
		if reflect.DeepEqual(foundStack, Stack{}) {
			return true, nil
		} else {
			return false, nil
		}
	}
	return false, fmt.Errorf("Could not interpret coordinates '%v'", coords)

}

func main() {

	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", SlashHandler)
	r.HandleFunc("/newgame/{boardSize}", NewGameHandler)
	r.HandleFunc("/showgame/{gameID}", ShowGameHandler)
	r.HandleFunc("/place/{gameID}", PlaceMoveHandler).Methods("PUT")

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}
