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
	// of the board, so to get the right slice position for the ranks, I've got to do the math below.
	file = LetterMap[validcoords[0][1]]
	rank, err := strconv.Atoi(validcoords[0][2])
	boardSize := len(b.Grid)
	rank = (boardSize - 1) - (rank - 1)

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
		return false, fmt.Errorf("Problem checking coordinates '%v': %v", coords, err)
	}
	// is there only an empty Stack{} on that square? If so, it's empty.
	if reflect.DeepEqual(foundStack, Stack{}) {
		return true, nil
	}
	return false, nil
}

// PlacePiece should put a Piece at a valid board position and return the updated board
func (b *Board) PlacePiece(p Placement) error {
	if err := b.validatePlacement(p); err != nil {
		return fmt.Errorf("bad placement request: %v", err)
	}
	rank, file, _ := b.TranslateCoords(p.Coords)

	square := &b.Grid[rank][file]
	square.Pieces = append([]Piece{p.Piece}, square.Pieces...)
	return nil
}

//validatePlacement checks to see if a Placement order is okay to run
func (b *Board) validatePlacement(p Placement) error {
	_, _, translateErr := b.TranslateCoords(p.Coords)
	if translateErr != nil {
		return fmt.Errorf("%v: %v", p.Coords, translateErr)
	}
	squareIsEmpty, emptyErr := b.SquareIsEmpty(p.Coords)

	rBlack := regexp.MustCompile("^(?i)black$")
	rWhite := regexp.MustCompile("^(?i)white$")
	switch {
	case emptyErr != nil:
		return fmt.Errorf("Problem checking square %v: %v", p.Coords, emptyErr)
	case squareIsEmpty != true:
		return fmt.Errorf("Cannot place piece on occupied square %v", p.Coords)
	case b.IsDarkTurn && rWhite.MatchString(p.Piece.Color):
		return errors.New("Cannot place white piece on black turn")
	case (b.IsDarkTurn == false) && rBlack.MatchString(p.Piece.Color):
		return errors.New("Cannot place black piece on white turn")
	}
	return nil
}

// WhatColor returns a valid color or an error
func (p *Piece) WhatColor() (string, error) {
	r := regexp.MustCompile("^((?i)black|white)$")
	goodPieceColor := r.FindString(p.Color)
	if goodPieceColor == "" {
		return "", fmt.Errorf("Invalid piece color '%v'", p.Color)
	}
	return goodPieceColor, nil
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
		return fmt.Errorf("Requested drops (%v) exceed number of pieces carried (%v)", m.Drops, m.Carry)
	case minDrop < 1:
		return fmt.Errorf("Stack movements (%v) include a drop less than 1: %v", m.Drops, minDrop)
	case moveTooBig != nil:
		return moveTooBig
	case unparsableDirection != nil:
		return unparsableDirection
	}
	return nil
}

// MoveStack moves a stack from a valid board position and return the updated board
func (b *Board) MoveStack(movement Movement) error {

	if err := b.validateMovement(movement); err != nil {
		return fmt.Errorf("bad movement request: %v", err)
	}

	// I've already validated the move above explicitly; assume no error
	rank, file, _ := b.TranslateCoords(movement.Coords)
	// pointer to the square where the movement originates
	square := &b.Grid[rank][file]
	// set up for the sequence of next squares the move will cover
	var nextSquare *Stack
	// create a new slice for the pieces in motion, and copy the "top" pieces from the origin square
	movingStack := make([]Piece, movement.Carry)
	copy(movingStack, square.Pieces[0:movement.Carry])

	// take the carried pieces off the origin
	square.Pieces = square.Pieces[movement.Carry:]

	// cycle through the array of drop orders
	for _, DropCount := range movement.Drops {

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
