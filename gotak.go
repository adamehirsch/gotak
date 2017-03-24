package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
)

// PlacePiece should put a Piece at a valid board position and return the updated board
func (b *TakGame) PlacePiece(p Placement) error {
	if err := b.ValidatePlacement(p); err != nil {
		return fmt.Errorf("bad placement request: %v", err)
	}
	p.Piece.Color = strings.ToLower(p.Piece.Color)
	rank, file, _ := b.TranslateCoords(p.Coords)
	square := &b.GameBoard[rank][file]
	// Place That Piece!
	square.Pieces = append([]Piece{p.Piece}, square.Pieces...)
	if b.IsBlackTurn == true {
		b.IsBlackTurn = false
	} else {
		b.IsBlackTurn = true
	}
	return nil
}

// MoveStack moves a stack from a valid board position and return the updated board
func (b *TakGame) MoveStack(movement Movement) error {

	if err := b.ValidateMovement(movement); err != nil {
		return fmt.Errorf("invalid move: %v", err)
	}

	// I've already validated the move above explicitly; assume no error
	rank, file, _ := b.TranslateCoords(movement.Coords)
	// pointer to the square where the movement originates
	square := &b.GameBoard[rank][file]
	// set up for the sequence of next squares the move will cover
	var nextSquare *Stack
	// create a new slice for the pieces in motion, and copy the top pieces from the origin square
	movingStack := make([]Piece, movement.Carry)
	copy(movingStack, square.Pieces[0:movement.Carry])

	// remove the carried pieces off the origin stack
	square.Pieces = square.Pieces[movement.Carry:]

	// Move That Stack!
	for _, DropCount := range movement.Drops {

		switch movement.Direction {
		case ">":
			nextSquare = &b.GameBoard[rank][file+1]
			file++
		case "<":
			nextSquare = &b.GameBoard[rank][file-1]
			file--
		case "+":
			nextSquare = &b.GameBoard[rank-1][file]
			rank--
		case "-":
			nextSquare = &b.GameBoard[rank+1][file]
			rank++
		default:
			return fmt.Errorf("can't determine movement direction '%v'", movement.Direction)
		}

		nextSquare.Pieces = append(movingStack[len(movingStack)-(DropCount):], nextSquare.Pieces...)
		// for the next drop, trim off the elements of the slice that have already been dropped off
		movingStack = movingStack[:len(movingStack)-(DropCount)]
	}

	if b.IsBlackTurn == true {
		b.IsBlackTurn = false
	} else {
		b.IsBlackTurn = true
	}
	return nil
}

//ValidatePlacement checks to see if a Placement order is okay to run
func (b *TakGame) ValidatePlacement(p Placement) error {

	if invalidPiece := p.Piece.ValidatePiece(); invalidPiece != nil {
		return invalidPiece
	}

	if _, _, translateErr := b.TranslateCoords(p.Coords); translateErr != nil {
		return fmt.Errorf("%v: %v", p.Coords, translateErr)
	}

	squareIsEmpty, emptyErr := b.SquareIsEmpty(p.Coords)
	tooManyCapstones := b.TooManyCapstones(p)
	tooManyPieces := b.TooManyPieces(p)
	rBlack := regexp.MustCompile("^(?i)black$")
	rWhite := regexp.MustCompile("^(?i)white$")
	switch {
	case emptyErr != nil:
		return fmt.Errorf("Problem checking square %v: %v", p.Coords, emptyErr)
	case b.IsBlackTurn && rWhite.MatchString(p.Piece.Color):
		return errors.New("Cannot place white piece on black turn")
	case b.IsBlackTurn == false && rBlack.MatchString(p.Piece.Color):
		return errors.New("Cannot place black piece on white turn")
	case squareIsEmpty != true:
		return fmt.Errorf("Cannot place piece on occupied square %v", p.Coords)
	case len(b.GameBoard) < 5 && p.Piece.Orientation == Capstone:
		return errors.New("no capstones allowed in games smaller than 5x5")
	case p.Piece.Orientation == Capstone && tooManyCapstones != nil:
		return tooManyCapstones
	case tooManyPieces != nil:
		return tooManyPieces
	}
	return nil
}

// ValidateMovement checks to see if a Movement order is okay to run.
func (b *TakGame) ValidateMovement(m Movement) error {

	boardSize := len(b.GameBoard)
	squareIsEmpty, emptyErr := b.SquareIsEmpty(m.Coords)
	rank, file, translateErr := b.TranslateCoords(m.Coords)
	if translateErr != nil {
		return fmt.Errorf("%v: %v", m.Coords, translateErr)
	}
	stackHeight := len(b.GameBoard[rank][file].Pieces)
	moveTooBig := b.WouldHitBoardBoundary(m)
	unparsableDirection := b.ValidMoveDirection(m)
	var stackTop Piece
	if len(b.GameBoard[rank][file].Pieces) > 0 {
		stackTop = b.GameBoard[rank][file].Pieces[0]
	}
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
	case m.Carry > len(b.GameBoard):
		return fmt.Errorf("Requested carry of %v pieces exceeds board carry limit: %v", m.Carry, boardSize)
	case totalDrops > m.Carry:
		return fmt.Errorf("Requested drops (%v) exceed number of pieces carried (%v)", m.Drops, m.Carry)
	case minDrop < 1:
		return fmt.Errorf("Stack movements (%v) include a drop less than 1: %v", m.Drops, minDrop)
	case moveTooBig != nil:
		return moveTooBig
	case unparsableDirection != nil:
		return unparsableDirection
	case stackTop.Color == White && b.IsBlackTurn == true:
		return errors.New("cannot move white-topped stack on black's turn")
	case stackTop.Color == Black && b.IsBlackTurn == false:
		return errors.New("cannot move black-topped stack on white's turn")
	}
	return nil
}

// ValidatePiece checks to make sure a piece is described correctly
func (p *Piece) ValidatePiece() error {
	rColor := regexp.MustCompile("^((?i)black|white)$")
	rType := regexp.MustCompile("^((?i)flat|wall|capstone)")
	if goodPieceColor := rColor.FindString(p.Color); goodPieceColor == "" {
		return fmt.Errorf("Invalid piece color '%v'", p.Color)
	}
	if goodPieceType := rType.FindString(p.Orientation); goodPieceType == "" {
		return fmt.Errorf("Invalid piece orientation '%v'", p.Orientation)
	}
	p.Color = strings.ToLower(p.Color)
	p.Orientation = strings.ToLower(p.Orientation)
	return nil
}

// TranslateCoords turns human-submitted coordinates and turns them into actual slice positions on a given board's grid
func (b *TakGame) TranslateCoords(coords string) (rank int, file int, error error) {
	coords = strings.ToLower(coords)
	// look for coordinates in the form LetterNumber
	r := regexp.MustCompile("^([a-h])([1-8])$")
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
	boardSize := len(b.GameBoard)
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
func (b *TakGame) SquareContents(coords string) (Stack, error) {
	grid := b.GameBoard
	rank, file, err := b.TranslateCoords(coords)
	if err != nil {
		return Stack{}, err
	}
	foundStack := grid[rank][file]
	return foundStack, nil
}

// SquareIsEmpty returns a simple boolean to signal if ... wait for it ... a square is empty
func (b *TakGame) SquareIsEmpty(coords string) (bool, error) {
	foundStack, err := b.SquareContents(coords)
	if err != nil {
		return false, fmt.Errorf("Problem checking coordinates '%v': %v", coords, err)
	}
	// far easier to compare length of a slice than to get fancy about comparing empty structs.
	if len(foundStack.Pieces) == 0 {
		return true, nil
	}
	return false, nil
}

// TooManyPieces checks for hitting a player's piece limit. This will need to be thought out a little more thoroughly,
// since running out of pieces is a game-end condition.
func (b *TakGame) TooManyPieces(p Placement) error {
	placedPieces, err := b.CountPieces()
	if err != nil {
		return err
	}
	pieceLimits := map[int]int{
		3: 10,
		4: 15,
		5: 21,
		6: 30,
		8: 50,
	}

	boardSize := len(b.GameBoard)
	if (*placedPieces)[Black] >= pieceLimits[boardSize] {
		return errors.New("Black player is out of pieces")
	} else if (*placedPieces)[White] >= pieceLimits[boardSize] {
		return errors.New("White player is out of pieces")
	}
	return nil
}

// TooManyCapstones checks for the presence of too many capstones on the board and prevents placing another
func (b *TakGame) TooManyCapstones(p Placement) error {
	capstones := map[string]int{
		Black: 0,
		White: 0,
	}

	rBlack := regexp.MustCompile("^(?i)black$")
	rWhite := regexp.MustCompile("^(?i)white$")
	for i := 0; i < len(b.GameBoard); i++ {
		for j := 0; j < len(b.GameBoard); j++ {
			if len(b.GameBoard[i][j].Pieces) > 0 && b.GameBoard[i][j].Pieces[0].Orientation == "capstone" {
				if rBlack.MatchString(b.GameBoard[i][j].Pieces[0].Color) {
					capstones[Black]++
				} else if rWhite.MatchString(b.GameBoard[i][j].Pieces[0].Color) {
					capstones[White]++
				}
			}
		}
	}

	capstoneLimit := 0
	if len(b.GameBoard) == 8 {
		capstoneLimit = 2
	} else if len(b.GameBoard) >= 5 {
		capstoneLimit = 1
	}

	if p.Piece.Orientation == "capstone" {
		if p.Piece.Color == White && capstones[White] >= capstoneLimit {
			return fmt.Errorf("Board has already reached white capstone limit: %v", capstoneLimit)
		} else if p.Piece.Color == Black && capstones[Black] >= capstoneLimit {
			return fmt.Errorf("Board has already reached black capstone limit: %v", capstoneLimit)
		}
	}
	return nil
}

// WouldHitBoardBoundary checks whether a given move exceeds the board size
func (b *TakGame) WouldHitBoardBoundary(m Movement) error {
	boardSize := len(b.GameBoard)
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
func (b *TakGame) ValidMoveDirection(m Movement) error {
	r := regexp.MustCompile("^[+-<>]$")
	goodDirection := r.MatchString(m.Direction)
	if goodDirection == false {
		return fmt.Errorf("Invalid movement direction '%v'", m.Direction)
	}
	return nil
}

// IsGameOver detects whether the given game is over
func (b *TakGame) IsGameOver() (bool, error) {
	placedPieces, err := b.CountPieces()
	if err != nil {
		return false, err
	}
	pieceLimits := map[int]int{
		3: 10,
		4: 15,
		5: 21,
		6: 30,
		8: 50,
	}

	boardSize := len(b.GameBoard)
	if (*placedPieces)[Black] >= pieceLimits[boardSize] {
		return true, nil
	} else if (*placedPieces)[White] >= pieceLimits[boardSize] {
		return true, nil
	}

	flatWin := true
	// look for a Flat Win
	for i := 0; i < len(b.GameBoard); i++ {
		for j := 0; j < len(b.GameBoard); j++ {
			if len(b.GameBoard[i][j].Pieces) == 0 {
				// there's at least one empty square; no flat Win
				flatWin = false
			}
		}
	}
	if flatWin == true {
		return true, nil
	}

	// TK: implement @lyda's road detection.

	return false, nil
}

// CountPieces counts how many black/white pieces are on the board, total
func (b *TakGame) CountPieces() (*map[string]int, error) {
	placedPieces := map[string]int{
		Black: 0,
		White: 0,
	}

	rBlack := regexp.MustCompile("^(?i)black$")
	rWhite := regexp.MustCompile("^(?i)white$")
	for i := 0; i < len(b.GameBoard); i++ {
		for j := 0; j < len(b.GameBoard); j++ {
			if len(b.GameBoard[i][j].Pieces) > 0 {
				if rBlack.MatchString(b.GameBoard[i][j].Pieces[0].Color) {
					placedPieces[Black]++
				} else if rWhite.MatchString(b.GameBoard[i][j].Pieces[0].Color) {
					placedPieces[White]++
				}
			}
		}
	}
	return &placedPieces, nil
}

// WhoWins determines who has won the game
func (b *TakGame) WhoWins() (string, error) {
	placedPieces, err := b.CountPieces()
	if err != nil {
		return "", err
	}
	if (*placedPieces)[Black] > (*placedPieces)[White] {
		b.IsBlackWinner = true
		return "Black makes a Flat Win!", nil
	} else if (*placedPieces)[White] > (*placedPieces)[Black] {
		b.IsBlackWinner = false
		return "White makes a Flat Win!", nil
	} else if (*placedPieces)[White] == (*placedPieces)[Black] {
		return "Game ends in a draw!", nil
	}

	// put in @lyda's road detection code here
	return "", nil
}

func main() {
	testGame := MakeGameBoard(7)
	testGame.GameID, _ = uuid.FromString("3fc74809-93eb-465d-a942-ef12427f83c5")
	gameIndex[testGame.GameID] = testGame

	whiteFlat := Piece{White, "flat"}
	blackFlat := Piece{Black, "flat"}
	// whiteWall := Piece{White, "wall"}
	blackWall := Piece{Black, "wall"}
	whiteCap := Piece{White, "capstone"}
	// blackCap := Piece{Black, "capstone"}

	// Board looks like this.
	// .o.....
	// ooo....
	// ..o....
	// ..oooo.
	// ooooooo
	// .....o.
	// .....o.
	testGame.GameBoard[0][1] = Stack{[]Piece{whiteCap, whiteFlat, blackFlat}}
	testGame.GameBoard[1][0] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}
	testGame.GameBoard[1][1] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	testGame.GameBoard[1][2] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}
	testGame.GameBoard[2][2] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}
	testGame.GameBoard[3][2] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}
	testGame.GameBoard[3][3] = Stack{[]Piece{whiteFlat}}
	testGame.GameBoard[3][4] = Stack{[]Piece{whiteFlat}}
	testGame.GameBoard[3][5] = Stack{[]Piece{whiteFlat}}
	testGame.GameBoard[4][5] = Stack{[]Piece{whiteFlat}}
	testGame.GameBoard[4][6] = Stack{[]Piece{whiteFlat}}
	testGame.GameBoard[4][4] = Stack{[]Piece{whiteFlat}}
	testGame.GameBoard[4][3] = Stack{[]Piece{whiteFlat}}
	testGame.GameBoard[4][2] = Stack{[]Piece{whiteFlat}}
	testGame.GameBoard[4][1] = Stack{[]Piece{whiteFlat}}
	testGame.GameBoard[4][0] = Stack{[]Piece{whiteFlat}}

	testGame.GameBoard[5][5] = Stack{[]Piece{whiteFlat}}
	testGame.GameBoard[6][5] = Stack{[]Piece{whiteFlat}}

	fmt.Printf("NS check: %v\n", testGame.NorthSouthCheck())

	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", SlashHandler)
	r.HandleFunc("/newgame/{boardSize}", NewGameHandler)
	r.HandleFunc("/showgame/{gameID}", ShowGameHandler)
	r.Handle("/action/{action}/{gameID}", webHandler(ActionHandler)).Methods("PUT")

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}
