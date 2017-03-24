package main

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

func (t *TakGame) nsCheck(s *Square) bool {
	boardsize := len(t.GameBoard)
	if s.rank == (boardsize - 1) {
		// the square being checked is on the very bottom row: success!
		return true
	}
	// get a list of orthogonal spaces (on the board)
	coordsNearby := t.CoordsAround(s.rank, s.file)
	for _, c := range coordsNearby {
		// if there's a piece on the board in an adjacent square ...
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

// squareSearched returns if a coord has been squareSearched.
func (s *Square) squareSearched(rank, file int) bool {
	parentSquare := s.parent
	for parentSquare != nil {
		// if a square's parent is itself, it's been seen?  this is not clear to me.
		if parentSquare.rank == rank && parentSquare.file == file {
			return true
		}
		parentSquare = parentSquare.parent
	}
	return false
}

// OccupiedCoords returns a simple boolean to signal if ... wait for it ... a square is empty
func (t *TakGame) OccupiedCoords(rank, file int) bool {
	grid := t.GameBoard
	foundStack := grid[rank][file]
	// far easier to compare length of a slice than to get fancy about comparing empty structs.
	if len(foundStack.Pieces) != 0 {
		return true
	}
	return false
}
