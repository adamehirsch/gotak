package main

import (
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
	BoardID    uuid.UUID
	Grid       [][]Stack
	IsDarkTurn bool
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
	Coords    string `json:"coords"`
	Direction string `json:"direction"`
	Carry     int    `json:"carry"`
	Drops     []int  `json:"drops"`
}

// WebError is a custom error type for reporting bad events when making an HTTP request
type WebError struct {
	Error   error
	Message string
	Code    int
}

// TranslateCoords turns human-submitted coordinates and turns them into actual slice positions on a given board's grid
func (b *Board) TranslateCoords(coords string) (rank int, file int, error error) {

	// look for coordinates in the form LetterNumber
	r := regexp.MustCompile("^([a-mA-M])([1-9]|[1][0-3])$")
	validcoords := r.FindAllStringSubmatch(coords, -1)
	if len(validcoords) <= 0 {
		return -1, -1, fmt.Errorf("Could not interpret coordinates '%v'", coords)
	}
	// Assuming we've got a valid looking set of coordinates, look them up on the provided board
	// ranks are numbered, up the sides; files are lettered across the bottom
	// Also of note is that Tak coordinates start with "a" as the first rank at the *bottom*
	// of the board, so to get the right slice position, I've got to do the math below.
	file = LetterMap[validcoords[0][1]]
	rank, err := strconv.Atoi(validcoords[0][2])
	boardSize := len(b.Grid)
	rank = (boardSize - 1) - (rank - 1)
	// fmt.Printf("coords: %v rank: %v file: %v boardSize: %v\n", coords, rank, file, boardSize)

	switch {
	case err != nil:
		return -1, -1, fmt.Errorf("problem interpreting coordinates %v", validcoords[0][0])
	case rank < 0 || file >= boardSize:
		return -1, -1, fmt.Errorf("coordinates '%v' larger than board size: %v", validcoords[0][0], boardSize)
	}
	return rank, file, nil
}

// SquareContents looks at a given spot on a given board and returns what's there
func (b *Board) SquareContents(coords string) (Stack, error) {
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
	foundStack, err := b.SquareContents(coords)
	if err != nil {
		return false, fmt.Errorf("Problem at coordinates '%v': %v", coords, err)
	}
	// is there only an empty Stack{} on that square? If so, it's empty.
	if reflect.DeepEqual(foundStack, Stack{}) {
		return true, nil
	}
	return false, nil
}

// PlacePiece should put a Piece at a valid board position and return the updated board
func (b *Board) PlacePiece(coords string, pieceToPlace Piece) error {
	empty, err := b.SquareIsEmpty(coords)
	if err != nil {
		return fmt.Errorf("Could not place piece at %v: %v", coords, err)
	}
	if empty == false {
		return fmt.Errorf("Could not place piece at occupied square %v", coords)
	}
	rank, file, translateErr := b.TranslateCoords(coords)
	if translateErr != nil {
		return translateErr
	}
	square := &b.Grid[rank][file]
	square.Pieces = append([]Piece{pieceToPlace}, square.Pieces...)
	return nil
}

// MoveStack should move a stack from a valid board position and return the updated board
func (b *Board) MoveStack(movement Movement) error {

	if err := b.validateMovement(movement); err != nil {
		return fmt.Errorf("bad movement request: %v", err)
	}

	// I've already validated the move above; there should be no error
	rank, file, _ := b.TranslateCoords(movement.Coords)
	square := &b.Grid[rank][file]
	var nextSquare *Stack
	movingStack := make([]Piece, movement.Carry)
	copy(movingStack, square.Pieces[0:movement.Carry])
	square.Pieces = square.Pieces[movement.Carry:]

	for _, DropCount := range movement.Drops {

		square = &b.Grid[rank][file]

		switch movement.Direction {
		case ">":
			nextSquare = &b.Grid[rank][file+1]
			file++
		case "<":
			nextSquare = &b.Grid[rank][file-1]
			file--
		case "+":
			nextSquare = &b.Grid[rank-1][file]
			rank--
		case "-":
			nextSquare = &b.Grid[rank+1][file]
			rank++
		default:
			return fmt.Errorf("can't determine movement direction '%v'", movement.Direction)
		}

		nextSquare.Pieces = append(movingStack[len(movingStack)-(DropCount):], nextSquare.Pieces...)
		// for the next drop, trim off the elements of the slice that have already been dropped off
		movingStack = movingStack[:len(movingStack)-(DropCount)]
		fmt.Printf("-2- movingStack: %v\n\n", movingStack)

	}
	return nil

}

// validateMovement checks to see if a Movement order is okay to run.
func (b *Board) validateMovement(m Movement) error {

	boardSize := len(b.Grid)
	squareIsEmpty, emptyErr := b.SquareIsEmpty(m.Coords)
	rank, file, translateErr := b.TranslateCoords(m.Coords)
	if translateErr != nil {
		return fmt.Errorf("%v: %v", m.Coords, translateErr)
	}
	stackHeight := len(b.Grid[rank][file].Pieces)
	moveTooBig := b.WouldHitBoardBoundary(m)
	unparsableDirection := b.ValidMoveDirection(m)
	var totalDrops, minDrop, maxDrop int
	minDrop = 1
	for _, drop := range m.Drops {
		totalDrops += drop
		if drop < minDrop {
			minDrop = drop
		}
		if drop > maxDrop {
			maxDrop = drop
		}
	}

	switch {
	case emptyErr != nil:
		return fmt.Errorf("Problem checking square %v: %v", m.Coords, emptyErr)
	case squareIsEmpty == true:
		return fmt.Errorf("Cannot move non-existent stack: unoccupied square %v", m.Coords)
	case m.Carry > stackHeight:
		return fmt.Errorf("Stack at %v is %v high - cannot carry %v pieces", m.Coords, stackHeight, m.Carry)
	case m.Carry > len(b.Grid):
		return fmt.Errorf("Requested carry of %v pieces exceeds board carry limit: %v", m.Carry, boardSize)
	case totalDrops > m.Carry:
		return fmt.Errorf("Requested drops (%v) exceed carry of %v pieces", m.Drops, m.Carry)
	case minDrop < 1:
		return fmt.Errorf("Stack movements (%v) include a drop less than 1: %v", m.Drops, minDrop)
	case moveTooBig != nil:
		return moveTooBig
	case unparsableDirection != nil:
		return unparsableDirection
	}
	return nil
}

// WouldHitBoardBoundary checks whether a given move exceeds the board size
func (b *Board) WouldHitBoardBoundary(m Movement) error {
	boardSize := len(b.Grid)
	badMove := b.ValidMoveDirection(m)
	rank, file, translateError := b.TranslateCoords(m.Coords)
	if badMove != nil {
		return fmt.Errorf("can't parse move direction '%v'", m.Direction)
	}
	if translateError != nil {
		return fmt.Errorf("can't parse coordinates '%v'", m.Coords)
	}
	switch {
	case (m.Direction == "<") && (file-len(m.Drops)) < 0:
		return fmt.Errorf("Stack movement (%v) would exceed left board edge", m.Drops)
	case (m.Direction == ">") && (file+len(m.Drops)) >= boardSize:
		return fmt.Errorf("Stack movement (%v) would exceed right board edge", m.Drops)
	case (m.Direction == "+") && (rank-len(m.Drops)) < 0:
		return fmt.Errorf("Stack movement (%v) would exceed top board edge", m.Drops)
	case (m.Direction == "-") && (rank+len(m.Drops)) >= boardSize:
		return fmt.Errorf("Stack movement (%v) would exceed bottom board edge", m.Drops)
	}
	return nil
}

// ValidMoveDirection checks that the move direction is correct
func (b *Board) ValidMoveDirection(m Movement) error {
	r := regexp.MustCompile("^[+-<>]$")
	goodDirection := r.MatchString(m.Direction)
	if goodDirection == false {
		return fmt.Errorf("Invalid movement direction '%v'", m.Direction)
	}
	return nil
}

func main() {
	testBoard := MakeGameBoard(5)
	testBoard.BoardID, _ = uuid.FromString("3fc74809-93eb-465d-a942-ef12427f83c5")
	gameIndex[testBoard.BoardID] = testBoard

	whiteFlat := Piece{"A", "flat"}
	blackFlat := Piece{"B", "flat"}
	cowFlat := Piece{"C", "flat"}
	dogFlat := Piece{"D", "flat"}
	eggFlat := Piece{"E", "flat"}

	// whiteCapstone := Piece{"white", "capstone"}
	// blackCapstone := Piece{"black", "capstone"}

	// b2
	// testBoard.Grid[4][1] = Stack{[]Piece{whiteCapstone, whiteFlat, blackFlat}}
	// a1
	testBoard.Grid[4][0] = Stack{[]Piece{whiteFlat, blackFlat, cowFlat, dogFlat, eggFlat}}
	// d4
	// testBoard.Grid[1][3] = Stack{[]Piece{blackCapstone, whiteFlat, blackFlat, whiteFlat, blackFlat}}

	fmt.Printf("testboard: %v\n", testBoard.BoardID)

	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", SlashHandler)
	r.HandleFunc("/newgame/{boardSize}", NewGameHandler)
	r.HandleFunc("/showgame/{gameID}", ShowGameHandler)
	// r.Handle("/place/{gameID}", webHandler(PlaceMoveHandler)).Methods("PUT")
	r.Handle("/action/{action}/{gameID}", webHandler(ActionHandler)).Methods("PUT")

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}
