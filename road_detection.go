package main

import "fmt"

// Coords are just an rank, file pair
type Coords struct {
	rank, file int
}

// Square tracks a connected line of pieces through a board.
type Square struct {
	rank, file int
	parent     *Square
}

// newSearchedSquare creates a new potential path step
func newSearchedSquare(rank, file int) *Square {
	square := &Square{
		rank:   rank,
		file:   file,
		parent: nil,
	}
	return square
}

// squareSearched starts from the current Square and traces its parent and
// parent's parents back to the beginning of the path, returning true if the
// requested coordinates show up anywhere in the ancestry, and false if it's a
// new location in this search run.
func (s *Square) squareSearched(rank, file int) bool {
	// the square doing the searching is looking to see whether the rank/file coords show up in its _own_ ancestry. Loop prevention.
	parentSquare := s.parent
	for parentSquare != nil {
		if parentSquare.rank == rank && parentSquare.file == file {
			return true
		}
		parentSquare = parentSquare.parent
	}
	return false
}

// OccupiedCoords returns a simple boolean if ... wait for it ... a square is empty
func (t *TakGame) OccupiedCoords(rank, file int) bool {
	grid := t.GameBoard
	foundStack := grid[rank][file]
	// far easier to compare length of a slice than to get fancy about comparing empty structs.
	if len(foundStack.Pieces) != 0 {
		return true
	}
	return false
}

// CoordsAround returns a series of rank/file coordinates for orthogonal positions around a given start point that don't exceed the board size.
func (t *TakGame) CoordsAround(rank, file int) []Coords {
	boardSize := len(t.GameBoard)
	var coordsToCheck []Coords

	if (file - 1) >= 0 {
		coordsToCheck = append(coordsToCheck, Coords{rank, file - 1})
	}

	if (file + 1) <= (boardSize - 1) {
		coordsToCheck = append(coordsToCheck, Coords{rank, file + 1})
	}

	if (rank - 1) >= 0 {
		coordsToCheck = append(coordsToCheck, Coords{rank - 1, file})
	}

	if (rank + 1) <= (boardSize - 1) {
		coordsToCheck = append(coordsToCheck, Coords{rank + 1, file})
	}
	return coordsToCheck
}

// NorthSouthCheck looks for a vertical path across the board
func (t *TakGame) NorthSouthCheck() bool {
	for j := 0; j < len(t.GameBoard); j++ {
		// only proceed if there's a piece on the top row
		if t.OccupiedCoords(0, j) {
			// Establish a path and send nsCheck looking.
			if foundAPath := t.nsCheck(newSearchedSquare(0, j)); foundAPath == true {
				return true
			}
		}
	}
	return false
}

var nsWinningPath []Coords

func (t *TakGame) nsCheck(s *Square) bool {
	boardsize := len(t.GameBoard)
	nsWinningPath = append(nsWinningPath, Coords{rank: s.rank, file: s.file})
	if s.rank == (boardsize - 1) {
		// the square being checked is on the very bottom row: success!
		fmt.Printf("NS Winning Path: %v\n\n", nsWinningPath)
		return true
	}
	// get a list of adjacent orthogonal spaces (on the board)
	coordsNearby := t.CoordsAround(s.rank, s.file)
	for _, c := range coordsNearby {

		// if there's a piece on the board in an adjacent square that hasn't been seen...
		if t.OccupiedCoords(c.rank, c.file) && !s.squareSearched(c.rank, c.file) {
			nextSquare := newSearchedSquare(c.rank, c.file)
			nextSquare.parent = s
			// let's get recursive all up in here. Keep drilling down until we get to the bottom of the board ...
			if found := t.nsCheck(nextSquare); found {
				return true
			}
		}
	}
	// ... or, you know, fail
	// trim last entry off
	nsWinningPath = nsWinningPath[:len(nsWinningPath)-1]
	return false
}

// WestEastCheck looks for a horizontal path across the board
func (t *TakGame) WestEastCheck() bool {
	for j := 0; j < len(t.GameBoard); j++ {
		// only proceed if there's a piece on the leftmost row
		if t.OccupiedCoords(j, 0) {
			// Establish a path and send weCheck looking.
			if foundAPath := t.weCheck(newSearchedSquare(j, 0)); foundAPath == true {
				return true
			}
		}
	}
	return false
}

var weWinningPath []Coords

func (t *TakGame) weCheck(s *Square) bool {
	boardsize := len(t.GameBoard)
	weWinningPath = append(weWinningPath, Coords{rank: s.rank, file: s.file})
	if s.file == (boardsize - 1) {
		// the square being checked is on the very rightmost row: success!
		fmt.Printf("Winning Path: %v\n\n", weWinningPath)
		return true
	}
	// get a list of adjacent orthogonal spaces (on the board)
	coordsNearby := t.CoordsAround(s.rank, s.file)
	for _, c := range coordsNearby {

		// if there's a piece on the board in an adjacent square that hasn't been seen...
		if t.OccupiedCoords(c.rank, c.file) && !s.squareSearched(c.rank, c.file) {
			nextSquare := newSearchedSquare(c.rank, c.file)
			nextSquare.parent = s
			// let's get recursive all up in here. Keep drilling down until we get to the bottom of the board ...
			if found := t.weCheck(nextSquare); found {
				return true
			}
		}
	}
	// ... or, you know, fail, and trim the last entry, which will not be part of the winning path
	weWinningPath = weWinningPath[:len(weWinningPath)-1]
	return false
}
