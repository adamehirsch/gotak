package main

import "fmt"

// Coords are just an y, x pair
type Coords struct {
	y, x int
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

// IsRoadWin looks for a path across the board
func (tg *TakGame) IsRoadWin(color string) bool {
	tg.WinningPath = nil
	for j := 0; j < len(tg.GameBoard); j++ {
		// check WestEast roads
		if tg.OccupiedCoords(j, 0) {
			// Check for WestEast roads.
			if foundAPath := tg.roadCheck(newSearchedSquare(j, 0), WestEast, color); foundAPath == true {
				return true
			}
		}
		// now check for a NorthSouth road
		if tg.OccupiedCoords(0, j) {
			if foundAPath := tg.roadCheck(newSearchedSquare(j, 0), NorthSouth, color); foundAPath == true {
				return true
			}
		}
	}
	return false
}

var tempWinningPath []string

func (tg *TakGame) roadCheck(s *Square, dir string, color string) bool {

	boardsize := len(tg.GameBoard)
	humanCoords, _ := tg.UnTranslateCoords(s.y, s.x)
	tempWinningPath = append(tempWinningPath, humanCoords)
	fmt.Printf("before: %v\n", tg.WinningPath)

	// if tg.GameOver == false {
	// 	// only do this once; we sometimes re-run the RoadCheck
	// 	// let's optimistically believe that the square we're working on is part of the winning path, unless and until proved otherwise
	// 	if humanCoords, err := tg.UnTranslateCoords(s.y, s.x); err == nil {
	// 		tg.WinningPath = append(tg.WinningPath, humanCoords)
	// 		fmt.Printf("-> checking: %+v %+v %v\n", s.x, s.y, tg.WinningPath)
	// 	}
	// }

	var thisPiecePosition int
	if dir == NorthSouth {
		thisPiecePosition = s.y
	} else if dir == WestEast {
		thisPiecePosition = s.x
	}

	if thisPiecePosition == (boardsize - 1) {
		// the square being checked is on the rightmost or bottom row:
		// declare success and shortcut the rest of the search.
		return true
	}

	// get a list of adjacent orthogonal spaces (on the board)
	coordsNearby := tg.NearbyOccupiedCoords(s.y, s.x)
	for _, c := range coordsNearby {

		// if there's a correctly colored piece on the board in an adjacent square that hasn't been seen...
		if tg.OccupiedCoords(c.y, c.x) && tg.GameBoard[c.y][c.x].Pieces[0].Color == color && !s.squareSearched(c.y, c.x) {
			nextSquare := newSearchedSquare(c.y, c.x)
			nextSquare.parent = s
			// let's get recursive all up in here. Keep drilling down until we get to the bottom of the board ...
			if found := tg.roadCheck(nextSquare, dir, color); found {
				fmt.Printf(" bingo: %v\n", tempWinningPath)
				tg.WinningPath = tempWinningPath
				return true
			}
		}
	}
	// ... or, you know, fail to find a path through this square. If so, trim the
	// last entry, which will not be part of the winning path
	// fmt.Printf("pruning %v\n", tg.WinningPath[len(tg.WinningPath)-1:])
	tempWinningPath = tempWinningPath[:len(tempWinningPath)-1]
	fmt.Printf(" after: %v\n", tempWinningPath)
	// if tg.GameOver == false {
	// }
	return false
}
