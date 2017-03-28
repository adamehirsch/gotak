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
)

// PlacePiece should put a Piece at a valid board position and return the updated board
func (tg *TakGame) PlacePiece(p Placement) error {
	if err := tg.ValidatePlacement(p); err != nil {
		return fmt.Errorf("bad placement request: %v", err)
	}
	p.Piece.Color = strings.ToLower(p.Piece.Color)
	rank, file, _ := tg.TranslateCoords(p.Coords)
	square := &tg.GameBoard[rank][file]
	// Place That Piece!
	square.Pieces = append([]Piece{p.Piece}, square.Pieces...)
	if tg.IsBlackTurn == true {
		tg.IsBlackTurn = false
	} else {
		tg.IsBlackTurn = true
	}
	return nil
}

// MoveStack moves a stack from a valid board position and return the updated board
func (tg *TakGame) MoveStack(movement Movement) error {

	if err := tg.ValidateMovement(movement); err != nil {
		return fmt.Errorf("invalid move: %v", err)
	}

	// I've already validated the move above explicitly; assume no error
	rank, file, _ := tg.TranslateCoords(movement.Coords)
	// pointer to the square where the movement originates
	square := &tg.GameBoard[rank][file]
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
			nextSquare = &tg.GameBoard[rank][file+1]
			file++
		case "<":
			nextSquare = &tg.GameBoard[rank][file-1]
			file--
		case "+":
			nextSquare = &tg.GameBoard[rank-1][file]
			rank--
		case "-":
			nextSquare = &tg.GameBoard[rank+1][file]
			rank++
		default:
			return fmt.Errorf("can't determine movement direction '%v'", movement.Direction)
		}

		nextSquare.Pieces = append(movingStack[len(movingStack)-(DropCount):], nextSquare.Pieces...)
		// for the next drop, trim off the elements of the slice that have already been dropped off
		movingStack = movingStack[:len(movingStack)-(DropCount)]
	}

	if tg.IsBlackTurn == true {
		tg.IsBlackTurn = false
	} else {
		tg.IsBlackTurn = true
	}
	return nil
}

//ValidatePlacement checks to see if a Placement order is okay to run
func (tg *TakGame) ValidatePlacement(p Placement) error {

	if invalidPiece := p.Piece.ValidatePiece(); invalidPiece != nil {
		return invalidPiece
	}

	if _, _, translateErr := tg.TranslateCoords(p.Coords); translateErr != nil {
		return fmt.Errorf("%v: %v", p.Coords, translateErr)
	}

	squareIsEmpty, emptyErr := tg.SquareIsEmpty(p.Coords)
	tooManyCapstones := tg.TooManyCapstones(p)
	tooManyPieces := tg.TooManyPieces(p)
	rBlack := regexp.MustCompile("^(?i)black$")
	rWhite := regexp.MustCompile("^(?i)white$")
	switch {
	case emptyErr != nil:
		return fmt.Errorf("Problem checking square %v: %v", p.Coords, emptyErr)
	case tg.IsBlackTurn && rWhite.MatchString(p.Piece.Color):
		return errors.New("Cannot place white piece on black turn")
	case tg.IsBlackTurn == false && rBlack.MatchString(p.Piece.Color):
		return errors.New("Cannot place black piece on white turn")
	case squareIsEmpty != true:
		return fmt.Errorf("Cannot place piece on occupied square %v", p.Coords)
	case len(tg.GameBoard) < 5 && p.Piece.Orientation == Capstone:
		return errors.New("no capstones allowed in games smaller than 5x5")
	case p.Piece.Orientation == Capstone && tooManyCapstones != nil:
		return tooManyCapstones
	case tooManyPieces != nil:
		return tooManyPieces
	}
	return nil
}

// ValidateMovement checks to see if a Movement order is okay to run.
func (tg *TakGame) ValidateMovement(m Movement) error {

	boardSize := len(tg.GameBoard)
	squareIsEmpty, emptyErr := tg.SquareIsEmpty(m.Coords)
	rank, file, translateErr := tg.TranslateCoords(m.Coords)
	if translateErr != nil {
		return fmt.Errorf("%v: %v", m.Coords, translateErr)
	}
	stackHeight := len(tg.GameBoard[rank][file].Pieces)
	moveTooBig := tg.WouldHitBoardBoundary(m)
	unparsableDirection := tg.ValidMoveDirection(m)
	var stackTop Piece
	if len(tg.GameBoard[rank][file].Pieces) > 0 {
		stackTop = tg.GameBoard[rank][file].Pieces[0]
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
	case stackTop.Color == White && tg.IsBlackTurn == true:
		return errors.New("cannot move white-topped stack on black's turn")
	case stackTop.Color == Black && tg.IsBlackTurn == false:
		return errors.New("cannot move black-topped stack on white's turn")
	case emptyErr != nil:
		return fmt.Errorf("Problem checking square %v: %v", m.Coords, emptyErr)
	case squareIsEmpty == true:
		return fmt.Errorf("Cannot move non-existent stack: unoccupied square %v", m.Coords)
	case m.Carry > stackHeight:
		return fmt.Errorf("Stack at %v is %v high - cannot carry %v pieces", m.Coords, stackHeight, m.Carry)
	case m.Carry > len(tg.GameBoard):
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
func (tg *TakGame) TranslateCoords(coords string) (rank int, file int, error error) {
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
	boardSize := len(tg.GameBoard)
	rank = (boardSize - 1) - (rank - 1)

	switch {
	case err != nil:
		return -1, -1, fmt.Errorf("problem interpreting coordinates %v", validcoords[0][0])
	case rank < 0 || file >= boardSize:
		return -1, -1, fmt.Errorf("coordinates '%v' larger than board size: %v", validcoords[0][0], boardSize)
	}
	return rank, file, nil
}

// UnTranslateCoords converts x, y coords back into human-readable Tak coords
func (tg *TakGame) UnTranslateCoords(rank int, file int) (string, error) {
	boardSize := len(tg.GameBoard)
	if 0 > rank || rank > boardSize {
		return "", fmt.Errorf("rank '%v' is out of bounds", rank)
	}
	number := boardSize - rank

	letter, ok := NumberToLetter[file]
	if ok == false {
		return "", fmt.Errorf("file '%v' is out of bounds", file)
	}
	return fmt.Sprintf("%v%d", letter, number), nil
}

// SquareContents looks at a given spot on a given board and returns what's there
func (tg *TakGame) SquareContents(coords string) (Stack, error) {
	grid := tg.GameBoard
	rank, file, err := tg.TranslateCoords(coords)
	if err != nil {
		return Stack{}, err
	}
	foundStack := grid[rank][file]
	return foundStack, nil
}

// SquareIsEmpty returns a simple boolean to signal if ... wait for it ... a square is empty
func (tg *TakGame) SquareIsEmpty(coords string) (bool, error) {
	foundStack, err := tg.SquareContents(coords)
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
func (tg *TakGame) TooManyPieces(p Placement) error {
	_, totalPlacedPieces := tg.CountAllPlacedPieces()

	boardSize := len(tg.GameBoard)
	if totalPlacedPieces[Black] >= PieceLimits[boardSize] {
		return errors.New("Black player is out of pieces")
	} else if totalPlacedPieces[White] >= PieceLimits[boardSize] {
		return errors.New("White player is out of pieces")
	}
	return nil
}

// TooManyCapstones checks for the presence of too many capstones on the board and prevents placing another
func (tg *TakGame) TooManyCapstones(p Placement) error {
	capstones := map[string]int{
		Black: 0,
		White: 0,
	}

	rBlack := regexp.MustCompile("^(?i)black$")
	rWhite := regexp.MustCompile("^(?i)white$")
	for i := 0; i < len(tg.GameBoard); i++ {
		for j := 0; j < len(tg.GameBoard); j++ {
			if len(tg.GameBoard[i][j].Pieces) > 0 && tg.GameBoard[i][j].Pieces[0].Orientation == Capstone {
				if rBlack.MatchString(tg.GameBoard[i][j].Pieces[0].Color) {
					capstones[Black]++
				} else if rWhite.MatchString(tg.GameBoard[i][j].Pieces[0].Color) {
					capstones[White]++
				}
			}
		}
	}

	capstoneLimit := 0

	switch {
	case len(tg.GameBoard) == 8:
		capstoneLimit = 2
	case len(tg.GameBoard) >= 5:
		capstoneLimit = 1
	}

	if p.Piece.Orientation == Capstone {
		if p.Piece.Color == White && capstones[White] >= capstoneLimit {
			return fmt.Errorf("Board has already reached white capstone limit: %v", capstoneLimit)
		} else if p.Piece.Color == Black && capstones[Black] >= capstoneLimit {
			return fmt.Errorf("Board has already reached black capstone limit: %v", capstoneLimit)
		}
	}
	return nil
}

// WouldHitBoardBoundary checks whether a given move exceeds the board size
func (tg *TakGame) WouldHitBoardBoundary(m Movement) error {
	boardSize := len(tg.GameBoard)
	badMove := tg.ValidMoveDirection(m)
	rank, file, translateError := tg.TranslateCoords(m.Coords)
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
func (tg *TakGame) ValidMoveDirection(m Movement) error {
	r := regexp.MustCompile("^[+-<>]$")
	goodDirection := r.MatchString(m.Direction)
	if goodDirection == false {
		return fmt.Errorf("Invalid movement direction '%v'", m.Direction)
	}
	return nil
}

// IsGameOver detects whether the given game is over
func (tg *TakGame) IsGameOver() bool {
	_, totalPlacedPieces := tg.CountAllPlacedPieces()

	boardSize := len(tg.GameBoard)
	if totalPlacedPieces[Black] >= PieceLimits[boardSize] {
		return true
	} else if totalPlacedPieces[White] >= PieceLimits[boardSize] {
		return true
	}

	flatWin := true
	// look for a Flat Win
	for i := 0; i < len(tg.GameBoard); i++ {
		for j := 0; j < len(tg.GameBoard); j++ {
			if len(tg.GameBoard[i][j].Pieces) == 0 {
				// there's at least one empty square; no flat Win
				flatWin = false
			}
		}
	}
	if flatWin == true {
		return true
	}

	if tg.RoadWinCheck(Black) || tg.RoadWinCheck(White) {
		return true
	}

	return false
}

// CountAllPlacedPieces counts how many black/white pieces top stacks on the board, as well as total placed pieces
func (tg *TakGame) CountAllPlacedPieces() (stackTops map[string]int, totalPlacedPieces map[string]int) {

	stackTops = map[string]int{
		Black: 0,
		White: 0,
	}

	totalPlacedPieces = map[string]int{
		Black: 0,
		White: 0,
	}
	rBlack := regexp.MustCompile("^(?i)black$")
	rWhite := regexp.MustCompile("^(?i)white$")
	for i := 0; i < len(tg.GameBoard); i++ {
		for j := 0; j < len(tg.GameBoard); j++ {
			if len(tg.GameBoard[i][j].Pieces) > 0 {
				if rBlack.MatchString(tg.GameBoard[i][j].Pieces[0].Color) {
					stackTops[Black]++
				} else if rWhite.MatchString(tg.GameBoard[i][j].Pieces[0].Color) {
					stackTops[White]++
				}
				for p := 0; p < len(tg.GameBoard[i][j].Pieces); p++ {
					if rBlack.MatchString(tg.GameBoard[i][j].Pieces[p].Color) {
						totalPlacedPieces[Black]++
					} else if rWhite.MatchString(tg.GameBoard[i][j].Pieces[p].Color) {
						totalPlacedPieces[White]++
					}
				}
			}
		}
	}
	return stackTops, totalPlacedPieces
}

// WhoWins determines who has won the game
func (tg *TakGame) WhoWins() (string, error) {
	if tg.IsGameOver() == false {
		return "", errors.New("Game is not over, yet")
	}
	stackTops, _ := tg.CountAllPlacedPieces()

	switch {
	case tg.IsBlackTurn && tg.RoadWinCheck(Black):
		tg.IsBlackWinner = true
		return "Black makes a road win!", nil
	case tg.IsBlackTurn == false && tg.RoadWinCheck(White):
		tg.IsWhiteWinner = true
		return "White makes a road win!", nil
	case tg.RoadWinCheck(Black):
		tg.IsBlackWinner = true
		return "Black makes a road win!", nil
	case tg.RoadWinCheck(White):
		tg.IsWhiteWinner = true
		return "White makes a road win!", nil
	case stackTops[Black] > stackTops[White]:
		tg.IsBlackWinner = true
		return "Black makes a Flat Win!", nil
	case stackTops[White] > stackTops[Black]:
		tg.IsWhiteWinner = true
		return "White makes a Flat Win!", nil
	case stackTops[White] == stackTops[Black]:
		tg.DrawGame = true
		return "Game ends in a draw!", nil
	}
	return "", nil
}

func main() {

	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", SlashHandler)
	r.HandleFunc("/newgame/{boardSize}", NewGameHandler)
	r.HandleFunc("/showgame/{gameID}", ShowGameHandler)
	r.Handle("/action/{action}/{gameID}", webHandler(ActionHandler)).Methods("PUT")

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}
