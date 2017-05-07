package main

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// PlacePiece should put a Piece at a valid board position and return the updated board
func (tg *TakGame) PlacePiece(p Placement) error {

	if tg.IsGameOver() {
		winner, err := tg.WhoWins()
		if err != nil {
			return err
		}
		return fmt.Errorf("game over: %v", winner)
	}

	if err := tg.ValidatePlacement(p); err != nil {
		return fmt.Errorf("bad placement request: %v", err)
	}
	p.Piece.Color = strings.ToLower(p.Piece.Color)
	x, y, _ := tg.TranslateCoords(p.Coords)
	square := &tg.GameBoard[x][y]
	// Place That Piece! (top of the piece stacks is at position 0)
	square.Pieces = append([]Piece{p.Piece}, square.Pieces...)

	// Record the placemnt!
	tg.TurnHistory = append(tg.TurnHistory, p)

	if tg.IsBlackTurn == true {
		tg.IsBlackTurn = false
	} else {
		tg.IsBlackTurn = true
	}
	if tg.IsGameOver() {
		tg.WhoWins()
	}
	return nil
}

// MoveStack moves a stack from a valid board position and return the updated board
func (tg *TakGame) MoveStack(m Movement) error {

	if tg.IsGameOver() {
		winner, err := tg.WhoWins()
		if err != nil {
			return err
		}
		return fmt.Errorf("game over: %v", winner)
	}

	if err := tg.ValidateMovement(m); err != nil {
		return fmt.Errorf("invalid move: %v", err)
	}

	// I've already validated the move above explicitly; assume no error
	x, y, _ := tg.TranslateCoords(m.Coords)

	// pointer to the square where the movement originates
	square := &tg.GameBoard[x][y]

	carriedPieces := tg.FindMovingPieces(m)

	// remove the carried pieces off the top of the origin stack
	square.Pieces = square.Pieces[m.Carry:]

	// set up for the sequence of next squares the move will cover
	var nextSquare *Stack

	// Move That Stack! Each Drop can be a different length, remember
	for k, DropCount := range m.Drops {

		switch m.Direction {
		case ">":
			nextSquare = &tg.GameBoard[x+1][y]
			x++
		case "<":
			nextSquare = &tg.GameBoard[x-1][y]
			x--
		case "+":
			nextSquare = &tg.GameBoard[x][y+1]
			y--
		case "-":
			nextSquare = &tg.GameBoard[x][y-1]
			y++
		default:
			return fmt.Errorf("can't determine movement direction '%v'", m.Direction)
		}

		// if we're on the very last drop, there's a stack on the nextSquare, and we're down to one carriedPiece
		if k == len(m.Drops)-1 && len(nextSquare.Pieces) > 0 && len(carriedPieces) == 1 {
			if carriedPieces[0].Orientation == Capstone && nextSquare.Pieces[0].Orientation == Wall {
				nextSquare.Pieces[0].Orientation = Flat
			}
		}
		nextSquare.Pieces = append(carriedPieces[len(carriedPieces)-(DropCount):], nextSquare.Pieces...)
		// for the next drop, trim off the elements of the slice that have already been dropped off
		carriedPieces = carriedPieces[:len(carriedPieces)-(DropCount)]

	}

	// record the move in the game's turn history
	tg.TurnHistory = append(tg.TurnHistory, m)
	if tg.IsBlackTurn == true {
		tg.IsBlackTurn = false
	} else {
		tg.IsBlackTurn = true
	}
	return nil
}

// FindMovingPieces determines which pieces will move with a given Movement
func (tg *TakGame) FindMovingPieces(m Movement) []Piece {
	// I've already validated the move above explicitly; assume no error
	x, y, _ := tg.TranslateCoords(m.Coords)

	// pointer to the square where the movement originates
	square := &tg.GameBoard[x][y]

	// create a new slice for the pieces in motion, and copy the top pieces from the origin square
	movingStack := make([]Piece, m.Carry)
	copy(movingStack, square.Pieces[0:m.Carry])
	return movingStack
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
	hitPieceLimit, pieceErr := tg.HitPieceLimit()
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
	case hitPieceLimit:
		return pieceErr
	}
	return nil
}

// ValidateMovement checks to see if a Movement order is okay to run.
func (tg *TakGame) ValidateMovement(m Movement) error {

	boardSize := tg.Size
	var stackTop Piece

	squareIsEmpty, emptyErr := tg.SquareIsEmpty(m.Coords)
	x, y, translateErr := tg.TranslateCoords(m.Coords)
	if translateErr != nil {
		return fmt.Errorf("%v: %v", m.Coords, translateErr)
	}
	stackHeight := len(tg.GameBoard[x][y].Pieces)
	if len(tg.GameBoard[x][y].Pieces) > 0 {
		stackTop = tg.GameBoard[x][y].Pieces[0]
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
	case tg.WouldHitBoardBoundary(m) != nil:
		return tg.WouldHitBoardBoundary(m)
	case tg.WallInWay(m) != nil:
		return tg.WallInWay(m)
	case tg.ValidMoveDirection(m) != nil:
		return tg.ValidMoveDirection(m)
	}
	return nil
}

// WallInWay will check to see whether there's a wall in the way of a move that won't be correctly flattened by a capstone. Basically has to play out a whole move, however
func (tg *TakGame) WallInWay(m Movement) error {
	if badMove := tg.ValidMoveDirection(m); badMove != nil {
		return fmt.Errorf("can't parse move direction '%v'", m.Direction)
	}

	if badMove := tg.WouldHitBoardBoundary(m); badMove != nil {
		return fmt.Errorf("move would hit board boundary: %v", m.Drops)
	}

	x, y, translateError := tg.TranslateCoords(m.Coords)
	if translateError != nil {
		return fmt.Errorf("can't parse coordinates '%v'", m.Coords)
	}

	// is the stack we're carrying topped by a capstone?
	carriedPieces := tg.FindMovingPieces(m)

	hasCapstone := false
	if carriedPieces[0].Orientation == Capstone {
		hasCapstone = true
	}

	// at each square along the carry path, check for a standing stone
	for move := 1; move <= len(m.Drops); move++ {
		var xplace, yplace int
		switch m.Direction {
		case ">":
			xplace = x + move
			yplace = y
		case "<":
			xplace = x - move
			yplace = y
		case "+":
			xplace = x
			yplace = y + move
		case "-":
			xplace = x
			yplace = y - move
		default:
			return fmt.Errorf("can't determine movement direction '%v'", m.Direction)
		}

		nextSquare := tg.GameBoard[xplace][yplace]
		humanCoords, _ := tg.UnTranslateCoords(xplace, yplace)

		// if there's a piece on the square we're looking at ...
		if len(nextSquare.Pieces) > 0 {
			// fmt.Printf("Pieces encountered: %v\n", nextSquare.Pieces)
			switch nextSquare.Pieces[0].Orientation {
			// if it's a capstone, you're just out of luck.
			case Capstone:
				return fmt.Errorf("Movement can't flatten a capstone at %v", humanCoords)
			case Wall:
				if hasCapstone == false {
					return fmt.Errorf("Can't flatten standing stone at %v: no capstone on moving stack", humanCoords)
				}
				if m.Drops[move-1] != 1 {
					return fmt.Errorf("Only allowed to flatten standing stone at %v with 1 capstone, not %v pieces", humanCoords, m.Drops[move-1])
				}
				if move != len(m.Drops) {
					return fmt.Errorf("Can't flatten standing stone at %v: not on last drop of move sequence", humanCoords)
				}
				// if it's the last stop on the move, the last drop order is one piece, and the last dropped piece is a capstone, we're good
				return nil
			}
		}
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
func (tg *TakGame) TranslateCoords(coords string) (x int, y int, error error) {
	coords = strings.ToLower(coords)
	// look for coordinates in the form LetterNumber
	r := regexp.MustCompile("^([a-h])([1-8])$")
	validcoords := r.FindAllStringSubmatch(coords, -1)

	if len(validcoords) <= 0 {
		return -1, -1, fmt.Errorf("Could not interpret coordinates '%v'", coords)
	}
	letter := validcoords[0][1]
	number := validcoords[0][2]
	// Assuming we've got a valid looking set of coordinates, look them up on the provided board
	// Also of note is that Tak coordinates start with "a" as the first y at the *bottom*
	// of the board, so to get the right slice position for the ys, I've got to do the math below.
	x = LetterMap[letter]
	y, err := strconv.Atoi(number)
	boardSize := len(tg.GameBoard)
	y = (y - 1)

	switch {
	case err != nil:
		return -1, -1, fmt.Errorf("problem interpreting coordinates %v", validcoords[0][0])
	case y < 0 || x >= tg.Size || y >= tg.Size:
		return -1, -1, fmt.Errorf("coordinates '%v' larger than board size: %v", validcoords[0][0], boardSize)
	}
	return x, y, nil
}

// UnTranslateCoords converts x, y coords back into human-readable Tak coords
func (tg *TakGame) UnTranslateCoords(x int, y int) (string, error) {
	boardSize := len(tg.GameBoard)
	if 0 > y || y > boardSize-1 {
		return "", fmt.Errorf("y '%v' is out of bounds", y)
	}
	number := y + 1

	letter, ok := NumberToLetter[x]
	if ok == false {
		return "", fmt.Errorf("x '%v' is out of bounds", x)
	}
	return fmt.Sprintf("%v%d", letter, number), nil
}

// SquareContents looks at a given spot on a given board and returns what's there
func (tg *TakGame) SquareContents(coords string) (Stack, error) {
	grid := tg.GameBoard
	x, y, err := tg.TranslateCoords(coords)
	if err != nil {
		return Stack{}, err
	}
	foundStack := grid[x][y]
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

// HitPieceLimit checks for hitting a player's piece limit. This will need to be thought out a little more thoroughly,
// since running out of pieces is a game-end condition.
func (tg *TakGame) HitPieceLimit() (bool, error) {
	_, totalPlacedPieces := tg.CountAllPlacedPieces()

	boardSize := len(tg.GameBoard)
	if totalPlacedPieces[Black] >= PieceLimits[boardSize] {
		return true, errors.New("Black player is out of pieces")
	} else if totalPlacedPieces[White] >= PieceLimits[boardSize] {
		return true, errors.New("White player is out of pieces")
	}
	return false, nil
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
	if badMove != nil {
		return fmt.Errorf("can't parse move direction '%v'", m.Direction)
	}
	x, y, translateError := tg.TranslateCoords(m.Coords)
	if translateError != nil {
		return fmt.Errorf("can't parse coordinates '%v'", m.Coords)
	}
	switch {
	case (m.Direction == "<") && (x-len(m.Drops)) < 0:
		return fmt.Errorf("Stack movement (%v) would exceed left board edge", m.Drops)
	case (m.Direction == ">") && (x+len(m.Drops)) >= boardSize:
		return fmt.Errorf("Stack movement (%v) would exceed right board edge", m.Drops)
	case (m.Direction == "-") && (y-len(m.Drops)) < 0:
		return fmt.Errorf("Stack movement (%v) would exceed bottom board edge", m.Drops)
	case (m.Direction == "+") && (y+len(m.Drops)) >= boardSize:
		return fmt.Errorf("Stack movement (%v) would exceed top board edge", m.Drops)
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
	pieceLimitReached, _ := tg.HitPieceLimit()
	gameOver := false

	if pieceLimitReached || tg.IsFlatWin() || tg.IsRoadWin(Black) || tg.IsRoadWin(White) {
		gameOver = true
	}

	tg.GameOver = gameOver

	if gameOver && tg.WinTime.IsZero() {
		tg.WinTime = time.Now()
	}
	return gameOver
}

// IsFlatWin determines if there's been a flat win, i.e. board has no empty spaces
func (tg *TakGame) IsFlatWin() bool {
	boardSize := len(tg.GameBoard)

	flatWin := true
	// look for a Flat Win
	for i := 0; i < boardSize; i++ {
		for j := 0; j < boardSize; j++ {
			if len(tg.GameBoard[i][j].Pieces) == 0 {
				// there's at least one empty square; no flat Win
				flatWin = false
			}
		}
	}
	return flatWin
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
		return "", errors.New("game is not over, yet")
	}
	stackTops, _ := tg.CountAllPlacedPieces()
	pieceLimitReached, _ := tg.HitPieceLimit()

	switch {
	case tg.IsBlackTurn && tg.IsRoadWin(Black):
		tg.BlackWinner = true
		return "Black makes a road win!", nil
	case tg.IsBlackTurn == false && tg.IsRoadWin(White): // this is where to start
		tg.WhiteWinner = true
		return "White makes a road win!", nil
	case tg.IsRoadWin(Black):
		tg.BlackWinner = true
		return "Black makes a road win!", nil
	case tg.IsRoadWin(White):
		tg.WhiteWinner = true
		return "White makes a road win!", nil
	case tg.IsFlatWin() && stackTops[Black] > stackTops[White]:
		tg.BlackWinner = true
		return "Black makes a Flat Win!", nil
	case tg.IsFlatWin() && stackTops[White] > stackTops[Black]:
		tg.WhiteWinner = true
		return "White makes a Flat Win!", nil
	case tg.IsFlatWin() && stackTops[White] == stackTops[Black]:
		tg.DrawGame = true
		return "Game ends in a draw!", nil
	case pieceLimitReached && stackTops[Black] > stackTops[White]:
		tg.BlackWinner = true
		return "Black makes a Flat win: piece limit reached!", nil
	case pieceLimitReached && stackTops[White] > stackTops[Black]:
		tg.WhiteWinner = true
		return "White makes a Flat win: piece limit reached!", nil
	case pieceLimitReached && stackTops[White] == stackTops[Black]:
		tg.DrawGame = true
		return "Draw game: piece limit reached!", nil
	}
	return "", nil
}
