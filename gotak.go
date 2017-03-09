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

// Placement descripts the necessary aspects to describe an action that places a new piece on the board
type Placement struct {
	Piece  Piece  `json:"piece"`
	Coords string `json:"coords"`
}

// Movement contains the necessary aspects to describe an action that moves a stack.
type Movement struct {
	Coords     string `json:"coords"`
	Direction  string `json:"direction"`
	Carry      int    `json:"carry"`
	Deliveries []int  `json:"deliveries"`
}

// WebError is a custom error type for reporting bad events when making an HTTP request
type WebError struct {
	Error   error
	Message string
	Code    int
}

// Let's try something out.
type webHandler func(http.ResponseWriter, *http.Request) *WebError

func (fn webHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil { // e is *webError, not os.Error.
		http.Error(w, e.Message, e.Code)
	}
}

// TranslateCoords turns human-submitted coordinates and turns them into actual slice positions on a given board's grid
func (b *Board) TranslateCoords(coords string) (rank int, file int, error error) {
	grid := b.Grid

	// looking for coordinates in the form LetterNumber
	r := regexp.MustCompile("^([a-zA-Z])([1-9]|[12][0-9])$")

	validcoords := r.FindAllStringSubmatch(coords, -1)

	if len(validcoords) > 0 {
		// Assuming we've got a valid looking set of coordinates, look them up on the provided board
		rank := LetterMap[validcoords[0][1]]
		file, err := strconv.Atoi(validcoords[0][2])
		switch {
		case err != nil:
			return -1, -1, errors.New("cannot interpret coordinates")
		case rank >= len(grid) || file-1 >= len(grid):
			return -1, -1, fmt.Errorf("coordinates '%v' larger than board size: %v", validcoords[0][0], len(grid))
		}
		// arrays start with [0], so subtract one from the human-readable rank
		file = file - 1
		return rank, file, nil
	}
	return -1, -1, fmt.Errorf("Could not interpret coordinates '%v'", coords)
}

// CheckSquare looks at a given spot on a given board and returns what's there
func (b *Board) CheckSquare(coords string) (Stack, error) {
	grid := b.Grid
	rank, file, err := b.TranslateCoords(coords)
	if err != nil {
		return Stack{}, err
	}
	foundStack := grid[rank][file]
	return foundStack, nil
}

// SquareIsEmpty returns a simple boolean to signal if ... wait for it ... a square is empty
func (b *Board) SquareIsEmpty(coords string) (bool, error) {
	if foundStack, err := b.CheckSquare(coords); err == nil {
		// is there only an empty Stack{} on that square? If so, it's empty.
		if reflect.DeepEqual(foundStack, Stack{}) {
			return true, nil
		}
		return false, nil
	}
	return false, fmt.Errorf("Could not interpret coordinates '%v'", coords)
}

// PlacePiece should put a Piece at a valid board position and return the updated board
func (b *Board) PlacePiece(coords string, pieceToPlace Piece) error {
	if empty, err := b.SquareIsEmpty(coords); err == nil {
		if empty == false {
			return fmt.Errorf("Could not place piece at occupied square %v", coords)
		}
		if rank, file, err := b.TranslateCoords(coords); err == nil {
			square := &b.Grid[rank][file]
			square.Pieces = append([]Piece{pieceToPlace}, square.Pieces...)
			return nil
		}
	}
	return fmt.Errorf("Could not place piece at %v", coords)
}

func main() {

	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", SlashHandler)
	r.HandleFunc("/newgame/{boardSize}", NewGameHandler)
	r.HandleFunc("/showgame/{gameID}", ShowGameHandler)
	r.Handle("/place/{gameID}", webHandler(PlaceMoveHandler)).Methods("PUT")

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}
