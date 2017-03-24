package main

import "fmt"

// Coords are just an rank, file pair
type Coords struct {
	rank, file int
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
			// Establish a path and send ns looking.
			if foundAPath := t.nsCheck(newSearchedSquare(0, j)); foundAPath == true {
				return true
			}
		}
	}
	return false
}

var winningPath []Coords

func (t *TakGame) nsCheck(s *Square) bool {
	boardsize := len(t.GameBoard)
	winningPath = append(winningPath, Coords{rank: s.rank, file: s.file})
	fmt.Printf("#### nsCheck new square at %v, %v ####\n", s.rank, s.file)

	if s.rank == (boardsize - 1) {
		// the square being checked is on the very bottom row: success!
		fmt.Printf("* square at %v, %v completes a NS path!\n", s.rank, s.file)
		fmt.Printf("Winning Path: %v\n\n", winningPath)
		return true
	}
	// get a list of orthogonal spaces (on the board)
	coordsNearby := t.CoordsAround(s.rank, s.file)
	fmt.Printf("    * square at %v, %v needs coords checked: %v\n", s.rank, s.file, coordsNearby)
	for _, c := range coordsNearby {
		fmt.Printf(" - checking square at %v, %v with parent at %v, %v\n", c.rank, c.file, s.rank, s.file)

		// if there's a piece on the board in an adjacent square that hasn't been seen...
		if t.OccupiedCoords(c.rank, c.file) && !s.squareSearched(c.rank, c.file) {
			fmt.Printf("      - creating new searchSquare at %v, %v\n", c.rank, c.file)
			nextSquare := newSearchedSquare(c.rank, c.file)
			nextSquare.parent = s
			// let's get recursive all up in here. Keep drilling down until we get to the bottom of the board ...
			if found := t.nsCheck(nextSquare); found {
				return true
			}
		}
	}
	// ... or, you know, fail
	fmt.Printf("####> No NS path found, ending check on square %v, %v\n", s.rank, s.file)
	// trim last entry off
	winningPath = winningPath[:len(winningPath)-1]
	return false
}

// Square tracks a connected line of pieces through a board.
type Square struct {
	rank, file int
	parent     *Square
}

func newSearchedSquare(rank, file int) *Square {
	square := &Square{
		rank:   rank,
		file:   file,
		parent: nil,
	}
	return square
}

// squareSearched returns true if a coord has been squareSearched...
func (s *Square) squareSearched(rank, file int) bool {
	fmt.Printf("    --> square %v, %v previously searched?\n", rank, file)
	parentSquare := s.parent
	for parentSquare != nil {
		// iterate over each square's parents, back to the start of the search
		// if a square's parent is at any point itself, it's been seen?  this loop prevention is not clear to me.
		if parentSquare.rank == rank && parentSquare.file == file {
			fmt.Printf("      --> square at %v, %v has parent %v, %v\n", s.rank, s.file, parentSquare.rank, parentSquare.file)
			fmt.Printf("      --> squareSearched at %v, %v TRUE - shortcut search\n", s.rank, s.file)
			return true
		}
		fmt.Printf("      --> following parent chain back to square %v\n", parentSquare.parent)
		parentSquare = parentSquare.parent
	}
	fmt.Printf("    --> square %v, %v NEW!\n", rank, file)
	return false
}

// OccupiedCoords returns a simple boolean to signal if ... wait for it ... a square is empty
func (t *TakGame) OccupiedCoords(rank, file int) bool {
	grid := t.GameBoard
	foundStack := grid[rank][file]
	// far easier to compare length of a slice than to get fancy about comparing empty structs.
	fmt.Printf("    square at %v, %v is occupied? -> %v\n", rank, file, len(foundStack.Pieces) != 0)
	if len(foundStack.Pieces) != 0 {
		return true
	}
	return false
}
