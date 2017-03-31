package main

import (
	"fmt"
	"strings"
)

// Coords are just an y, x pair
type Coords struct {
	Y int `json:"y"`
	X int `json:"x"`
}

// Square tracks a connected line of pieces through a board.
type Square struct {
	y, x   int
	parent *Square
}

// newSearchedSquare creates a new potential path step
func newSearchedSquare(y, x int) *Square {
	square := &Square{
		y:      y,
		x:      x,
		parent: nil,
	}
	return square
}

// squareSearched starts from the current Square and traces its parent and
// parent's parents back to the beginning of the path, returning true if the
// requested coordinates show up anywhere in the ancestry, and false if it's a
// new location in this search run.
func (s *Square) squareSearched(y, x int) bool {
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

// OccupiedCoords returns a simple boolean if ... wait for it ... a square is empty
func (tg *TakGame) OccupiedCoords(y, x int) bool {
	grid := tg.GameBoard
	foundStack := grid[y][x]
	// far easier to compare length of a slice than to get fancy about comparing empty structs.
	if len(foundStack.Pieces) != 0 {
		return true
	}
	return false
}

// NearbyOccupiedCoords returns a series of occpupied y/x coordinates for
// orthogonal positions around a given start point that don't exceed the board size.
func (tg *TakGame) NearbyOccupiedCoords(y, x int) []Coords {
	boardSize := len(tg.GameBoard)
	var coordsToCheck []Coords

	if (x-1) >= 0 && tg.OccupiedCoords(y, x-1) {
		coordsToCheck = append(coordsToCheck, Coords{y, x - 1})
	}

	if (x+1) <= (boardSize-1) && tg.OccupiedCoords(y, x+1) {
		coordsToCheck = append(coordsToCheck, Coords{y, x + 1})
	}

	if (y-1) >= 0 && tg.OccupiedCoords(y-1, x) {
		coordsToCheck = append(coordsToCheck, Coords{y - 1, x})
	}

	if (y+1) <= (boardSize-1) && tg.OccupiedCoords(y+1, x) {
		coordsToCheck = append(coordsToCheck, Coords{y + 1, x})
	}
	return coordsToCheck
}

// IsRoadWin looks for a path across the board by the player of a given color.
func (tg *TakGame) IsRoadWin(color string) bool {

	for j := 0; j < len(tg.GameBoard); j++ {
		// now check for a NorthSouth road
		if tg.OccupiedCoords(0, j) && tg.GameBoard[0][j].Pieces[0].Color == color {
			if foundAPath := tg.roadCheck(newSearchedSquare(0, j), NorthSouth, color, []Coords{}); foundAPath == true {
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
		// check WestEast roads
		if tg.OccupiedCoords(j, 0) && tg.GameBoard[j][0].Pieces[0].Color == color {
			// Check for WestEast roads.
			if foundAPath := tg.roadCheck(newSearchedSquare(j, 0), WestEast, color, []Coords{}); foundAPath == true {
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
		// the square being checked is on the rightmost and/or bottom row:
		// declare success and shortcut the rest of the search.
		tg.WinningPath = pp
		return true
	}

	// get a list of adjacent orthogonal spaces (on the board)
	coordsNearby := tg.NearbyOccupiedCoords(s.y, s.x)
	for _, c := range coordsNearby {

		// if there's a correctly colored piece on the board in an adjacent square that hasn't been seen...
		if tg.OccupiedCoords(c.Y, c.X) && tg.GameBoard[c.Y][c.X].Pieces[0].Color == color && !s.squareSearched(c.Y, c.X) {
			nextSquare := newSearchedSquare(c.Y, c.X)
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
func (tg *TakGame) DrawStackTops() {

	boardSize := len(tg.GameBoard)
	printablePath := make([][]string, boardSize)
	for i := range printablePath {
		printablePath[i] = make([]string, boardSize)
	}

	for _, v := range tg.WinningPath {
		printablePath[v.Y][v.X] = "o"
	}

	fmt.Println(" " + strings.Repeat("---", boardSize))
	for y := 0; y < boardSize; y++ {
		fmt.Print("|")
		for x := 0; x < boardSize; x++ {
			if len(tg.GameBoard[y][x].Pieces) == 0 {
				fmt.Print(" . ")
			} else {
				highlightOpen := " "
				highlightClose := " "
				if printablePath[y][x] == "o" {
					highlightOpen = "("
					highlightClose = ")"
				}
				stackTop := "B"
				if tg.GameBoard[y][x].Pieces[0].Color == White {
					stackTop = "W"
				}
				fmt.Printf("%s%s%s", highlightOpen, stackTop, highlightClose)
			}
		}
		fmt.Print("|\n")
	}
	fmt.Println(" " + strings.Repeat("---", boardSize))

}
