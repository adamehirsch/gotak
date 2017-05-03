package main

import (
	"fmt"
	"strings"
)

// Coords are just an x, y pair
type Coords struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Square tracks a connected line of pieces through a board.
type Square struct {
	x, y   int
	parent *Square
}

// newSearchedSquare creates a new potential path step
func newSearchedSquare(x, y int) *Square {
	square := &Square{
		x:      x,
		y:      y,
		parent: nil,
	}
	return square
}

// squareSearched starts from the current Square and traces its parent and
// parent's parents back to the beginning of the path, returning true if the
// requested coordinates show up anywhere in the ancestry, and false if it's a
// new location in this search run.
func (s *Square) squareSearched(x, y int) bool {
	// the square doing the searching is looking to see whether the y/x coords show up in its _own_ ancestry. Loop prevention.
	parentSquare := s.parent
	for parentSquare != nil {
		if parentSquare.y == y && parentSquare.x == x {
			return true
		}
		parentSquare = parentSquare.parent
	}
	return false
}

// CoordsAreOccupied returns a simple boolean if ... wait for it ... a square is empty
func (tg *TakGame) CoordsAreOccupied(x, y int) bool {
	grid := tg.GameBoard
	foundStack := grid[x][y]
	// far easier to compare length of a slice than to get fancy about comparing empty structs.
	if len(foundStack.Pieces) != 0 {
		return true
	}
	return false
}

// NearbyOccupiedCoords returns a series of occpupied y/x coordinates for
// orthogonal positions around a given start point that don't exceed the board size.
func (tg *TakGame) NearbyOccupiedCoords(x, y int, direction string) []Coords {
	var coordsToCheck []Coords

	// for the first part of a NS road, make the first move vertically, not horizontally
	// i.e. look at adjacent horizontal squares only on a WE seek or a NS seek that's left the first row
	if direction == WestEast || y != 0 {

		if (x-1) >= 0 && tg.CoordsAreOccupied(x-1, y) {
			coordsToCheck = append(coordsToCheck, Coords{x - 1, y})
		}

		if (x+1) <= (tg.Size-1) && tg.CoordsAreOccupied(x+1, y) {
			coordsToCheck = append(coordsToCheck, Coords{x + 1, y})
		}
	}

	// for the first part of a WE road, make the first move horizontally, not vertically
	if direction == NorthSouth || x != 0 {
		if (y-1) >= 0 && tg.CoordsAreOccupied(x, y-1) {
			coordsToCheck = append(coordsToCheck, Coords{x, y - 1})
		}

		if (y+1) <= (tg.Size-1) && tg.CoordsAreOccupied(x, y+1) {
			coordsToCheck = append(coordsToCheck, Coords{x, y + 1})
		}
	}

	return coordsToCheck
}

// IsRoadWin looks for a path across the board by the player of a given color.
func (tg *TakGame) IsRoadWin(color string) bool {

	for j := 0; j < tg.Size; j++ {
		// fmt.Printf("WE %v j%v\n", color, j)

		// check for WestEast roads, starting on the leftmost side of the board
		if tg.CoordsAreOccupied(0, j) && tg.GameBoard[0][j].Pieces[0].Color == color {
			// Check for WestEast roads.
			if foundAPath := tg.roadCheck(newSearchedSquare(0, j), WestEast, color, []Coords{}); foundAPath == true {
				tg.RoadWin = true
				if color == Black {
					tg.BlackWinner = true
				}
				if color == White {
					tg.WhiteWinner = true
				}
				return true
			}
		}
		// fmt.Printf("NS %v j%v\n", color, j)

		// check NorthSouth roads
		if tg.CoordsAreOccupied(j, 0) && tg.GameBoard[j][0].Pieces[0].Color == color {
			if foundAPath := tg.roadCheck(newSearchedSquare(j, 0), NorthSouth, color, []Coords{}); foundAPath == true {
				tg.RoadWin = true
				if color == Black {
					tg.BlackWinner = true
				}
				if color == White {
					tg.WhiteWinner = true
				}
				return true
			}
		}

	}
	return false
}

func (tg *TakGame) roadCheck(s *Square, dir string, color string, pp []Coords) bool {

	boardsize := len(tg.GameBoard)
	// let's optimistically believe that this square we're checking will be part of the winningPath
	pp = append(pp, Coords{X: s.x, Y: s.y})
	var thisPiecePosition int
	if dir == NorthSouth {
		thisPiecePosition = s.y
	} else if dir == WestEast {
		thisPiecePosition = s.x
	}

	if thisPiecePosition == (boardsize - 1) {
		// the square being checked is on the rightmost and/or top row:
		// declare success and shortcut the rest of the search.
		tg.WinningPath = pp
		return true
	}

	// get a list of adjacent orthogonal spaces (on the board)
	coordsNearby := tg.NearbyOccupiedCoords(s.x, s.y, dir)

	for _, c := range coordsNearby {
		// if there's a correctly colored piece on the board in an adjacent square that hasn't been seen...
		if tg.CoordsAreOccupied(c.X, c.Y) && tg.GameBoard[c.X][c.Y].Pieces[0].Color == color && !s.squareSearched(c.X, c.Y) {
			nextSquare := newSearchedSquare(c.X, c.Y)
			nextSquare.parent = s
			// let's get recursive all up in here. Keep drilling down until we get to the bottom of the board ...
			if found := tg.roadCheck(nextSquare, dir, color, pp); found {
				return true
			}
		}
	}
	return false
}

// DrawStackTops draws a board with the winning path, if any, highlighted
func (tg *TakGame) DrawStackTops() []string {

	boardSize := tg.Size
	printablePath := make([][]string, boardSize)
	for i := range printablePath {
		printablePath[i] = make([]string, boardSize)
	}

	for _, v := range tg.WinningPath {
		printablePath[v.X][v.Y] = "o"
	}
	printableBoard := make([]string, boardSize+2)
	printableBoard[0] = " " + strings.Repeat("---", boardSize)
	// print the board from the top down
	for y := boardSize - 1; y >= 0; y-- {
		line := "|"
		for x := 0; x < boardSize; x++ {
			if len(tg.GameBoard[x][y].Pieces) == 0 {
				line = line + " . "
			} else {
				highlightOpen := " "
				highlightClose := " "
				if printablePath[x][y] == "o" {
					highlightOpen = "("
					highlightClose = ")"
				}
				stackTop := "B"
				if tg.GameBoard[x][y].Pieces[0].Color == White {
					stackTop = "W"
				}
				line = line + fmt.Sprintf("%s%s%s", highlightOpen, stackTop, highlightClose)
			}
		}
		line = line + "|"
		printableBoard[boardSize-y] = line
	}
	printableBoard[boardSize+1] = " " + strings.Repeat("---", boardSize)
	return printableBoard
}
