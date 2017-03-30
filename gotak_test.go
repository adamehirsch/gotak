package main

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	uuid "github.com/satori/go.uuid"
)

var whiteFlat = Piece{"white", "flat"}
var blackFlat = Piece{"black", "flat"}
var whiteCap = Piece{"white", "capstone"}
var blackCap = Piece{"black", "capstone"}
var whiteWall = Piece{"white", "wall"}
var blackWall = Piece{"black", "wall"}

func TestBoardSizeLimits(t *testing.T) {
	testBoard := MakeGameBoard(5)
	testBoard.GameBoard[4][0] = Stack{[]Piece{whiteWall, blackFlat}}
	testBoard.GameBoard[0][2] = Stack{[]Piece{whiteFlat, whiteFlat}}
	testBoard.GameBoard[1][3] = Stack{[]Piece{blackWall, whiteFlat}}

	// case-driven testing: The Bomb
	cases := []struct {
		coords  string
		stack   Stack
		problem error
	}{
		{"a1", Stack{[]Piece{whiteWall, blackFlat}}, nil},
		{"d4", Stack{[]Piece{blackWall, whiteFlat}}, nil},
		{"b2", Stack{}, nil},
		{"f1", Stack{}, errors.New("coordinates 'f1' larger than board size: 5")},
	}

	for _, c := range cases {
		testStack, err := testBoard.SquareContents(c.coords)
		if reflect.DeepEqual(testStack, c.stack) == false {
			t.Errorf("Returned stack from coords %v was %v: wanted %v\n", c.coords, testStack, c.stack)
		}

		if reflect.DeepEqual(err, c.problem) == false {
			t.Errorf("Returned error from coords %v was '%v': wanted '%v'\n", c.coords, err, c.problem)
		}
	}
}

// verify that
func TestBoardSquareEmpty(t *testing.T) {
	testBoard := MakeGameBoard(5)

	testBoard.GameBoard[4][1] = Stack{[]Piece{whiteWall, blackFlat}}
	testBoard.GameBoard[0][2] = Stack{[]Piece{whiteFlat, whiteFlat}}
	testBoard.GameBoard[1][3] = Stack{[]Piece{blackWall, whiteFlat}}

	// case-driven testing: The Bomb
	cases := []struct {
		Coords  string
		Empty   bool
		Problem error
	}{
		{"b1", false, nil},
		{"a5", true, nil},
		{"b2", true, nil},
		{"f1", false, errors.New("Problem checking coordinates 'f1': coordinates 'f1' larger than board size: 5")},
	}

	for _, c := range cases {
		isEmpty, err := testBoard.SquareIsEmpty(c.Coords)

		if reflect.DeepEqual(err, c.Problem) == false {
			t.Errorf("Returned error from coords %v was '%v': wanted '%v'\n", c.Coords, err, c.Problem)
		}

		if reflect.DeepEqual(isEmpty, c.Empty) == false {
			t.Errorf("Returned stack from coords %v was %v: wanted %v\n", c.Coords, isEmpty, c.Empty)
		}
	}
}

func TestNoPlacementOnOccupiedSquare(t *testing.T) {
	testBoard := MakeGameBoard(5)
	testBoard.GameBoard[4][1] = Stack{[]Piece{whiteFlat, blackFlat}}
	testBoard.GameBoard[0][0] = Stack{[]Piece{whiteFlat, whiteFlat}}
	testBoard.GameBoard[1][3] = Stack{[]Piece{whiteCap, blackFlat}}

	// case-driven testing: The Bomb
	cases := []struct {
		placement Placement
		Problem   error
	}{
		{Placement{Coords: "b1", Piece: whiteFlat}, errors.New("bad placement request: Cannot place piece on occupied square b1")},
		{Placement{Coords: "a5", Piece: blackFlat}, errors.New("bad placement request: Cannot place piece on occupied square a5")},
		{Placement{Coords: "b3", Piece: whiteWall}, nil},
		{Placement{Coords: "h1", Piece: blackFlat}, errors.New("bad placement request: h1: coordinates 'h1' larger than board size: 5")},
	}

	for _, c := range cases {
		err := testBoard.PlacePiece(c.placement)
		if testBoard.IsBlackTurn == true {
			testBoard.IsBlackTurn = false
		} else {
			testBoard.IsBlackTurn = true
		}
		if reflect.DeepEqual(err, c.Problem) == false {
			t.Errorf("Returned error from coords %v was '%v': wanted '%v'\n", c.placement.Coords, err, c.Problem)
		}

	}
}

func TestTurnTaking(t *testing.T) {
	testBoard := MakeGameBoard(5)
	bogusFlat := Piece{"bogus", "flatworm"}

	testBoard.GameBoard[4][1] = Stack{[]Piece{whiteFlat, blackFlat}}
	testBoard.GameBoard[0][0] = Stack{[]Piece{whiteFlat, whiteCap}}
	testBoard.GameBoard[1][3] = Stack{[]Piece{whiteCap, blackFlat}}
	testBoard.IsBlackTurn = true

	// case-driven testing: The Bomb
	cases := []struct {
		placement Placement
		Problem   error
	}{
		{Placement{Coords: "b1", Piece: blackFlat}, errors.New("bad placement request: Cannot place piece on occupied square b1")},
		{Placement{Coords: "a5", Piece: whiteCap}, errors.New("bad placement request: Cannot place piece on occupied square a5")},
		{Placement{Coords: "b2", Piece: whiteFlat}, errors.New("bad placement request: Cannot place white piece on black turn")},
		{Placement{Coords: "a4", Piece: blackFlat}, errors.New("bad placement request: Cannot place black piece on white turn")},
		{Placement{Coords: "b3", Piece: bogusFlat}, errors.New("bad placement request: Invalid piece color 'bogus'")},
		{Placement{Coords: "h1", Piece: whiteFlat}, errors.New("bad placement request: h1: coordinates 'h1' larger than board size: 5")},
	}

	for _, c := range cases {
		err := testBoard.PlacePiece(c.placement)
		if testBoard.IsBlackTurn == true {
			testBoard.IsBlackTurn = false
		} else {
			testBoard.IsBlackTurn = true
		}

		if reflect.DeepEqual(err, c.Problem) == false {
			t.Errorf("Returned error from coords %v was '%v': wanted '%v'\n", c.placement.Coords, err, c.Problem)
		}

	}
}

func TestEmptySquareDetection(t *testing.T) {
	testGame := MakeGameBoard(5)

	// b2
	testGame.GameBoard[3][1] = Stack{[]Piece{whiteCap, whiteFlat, blackFlat}}
	// c2
	testGame.GameBoard[3][2] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	// a1
	testGame.GameBoard[4][0] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}
	// d4
	testGame.GameBoard[1][3] = Stack{[]Piece{blackCap, whiteFlat, blackFlat, whiteFlat, blackFlat}}
	// c3
	testGame.GameBoard[2][2] = Stack{[]Piece{whiteWall}}

	cases := []struct {
		coords string
		empty  bool
	}{
		{"c3", false},
		{"c4", true},
	}

	for _, c := range cases {
		isEmpty, _ := testGame.SquareIsEmpty(c.coords)
		if isEmpty != c.empty {
			t.Errorf("coords %v SquareIsEmpty: '%v': should be '%v'\n", c.coords, isEmpty, c.empty)
		}
	}

	c4Move := Movement{Coords: "c3", Direction: "+", Carry: 1, Drops: []int{1}}
	testGame.MoveStack(c4Move)
	cases = []struct {
		coords string
		empty  bool
	}{
		{"c3", true},
		{"c4", false},
	}

	for _, c := range cases {
		isEmpty, _ := testGame.SquareIsEmpty(c.coords)
		if isEmpty != c.empty {
			t.Errorf("Post-move: coords %v SquareIsEmpty: '%v': should be '%v'\n", c.coords, isEmpty, c.empty)
		}
	}
}

func TestValidMoveDirection(t *testing.T) {
	testBoard := MakeGameBoard(5)

	cases := []struct {
		move    Movement
		Problem error
	}{
		{Movement{Coords: "b2", Direction: "+", Carry: 1, Drops: []int{1}}, nil},
		{Movement{Coords: "b2", Direction: "a", Carry: 1, Drops: []int{1}}, errors.New("Invalid movement direction 'a'")},
	}

	for _, c := range cases {
		err := testBoard.ValidMoveDirection(c.move)
		if reflect.DeepEqual(err, c.Problem) == false {
			t.Errorf("Returned error from coords %v was '%v': wanted '%v'\n", c.move, err, c.Problem)
		}

	}

}

func TestValidMovement(t *testing.T) {
	testBoard := MakeGameBoard(5)
	testBoard.GameBoard[4][1] = Stack{[]Piece{whiteFlat, blackFlat}}
	testBoard.GameBoard[0][0] = Stack{[]Piece{whiteFlat, blackFlat}}

	cases := []struct {
		move    Movement
		Problem error
	}{
		{Movement{Coords: "a5", Direction: "+", Carry: 1, Drops: []int{1}}, errors.New("Stack movement ([1]) would exceed top board edge")},
		{Movement{Coords: "b2", Direction: "a", Carry: 1, Drops: []int{1}}, errors.New("Cannot move non-existent stack: unoccupied square b2")},
	}

	for _, c := range cases {
		err := testBoard.ValidateMovement(c.move)
		if reflect.DeepEqual(err, c.Problem) == false {
			t.Errorf("Returned error from coords %v was '%v': wanted '%v'\n", c.move.Coords, err, c.Problem)
		}

	}

}

func TestCoordsAround(t *testing.T) {
	testGame := MakeGameBoard(5)

	// b2
	testGame.GameBoard[3][1] = Stack{[]Piece{whiteCap, whiteFlat, blackFlat}}
	// c2
	testGame.GameBoard[3][2] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	// a1
	testGame.GameBoard[4][0] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}
	// d4
	testGame.GameBoard[1][3] = Stack{[]Piece{blackCap, whiteFlat, blackFlat, whiteFlat, blackFlat}}
	// c3
	testGame.GameBoard[2][2] = Stack{[]Piece{whiteWall}}

	cases := []struct {
		coords       string
		coordsAround []Coords
	}{
		{"b2", []Coords{Coords{3, 2}}},
		{"b5", nil},
		{"c2", []Coords{Coords{3, 1}, Coords{2, 2}}},
		{"a1", nil},
		{"b1", []Coords{Coords{4, 0}, Coords{3, 1}}},
	}

	for _, c := range cases {
		y, x, _ := testGame.TranslateCoords(c.coords)
		coordsAround := testGame.NearbyOccupiedCoords(y, x)
		if reflect.DeepEqual(coordsAround, c.coordsAround) == false {
			t.Errorf("%v Wanted coords %v got CoordsAround %v\n", c.coords, c.coordsAround, coordsAround)
		}
	}
}

func TestPathSearch(t *testing.T) {
	testGame := MakeGameBoard(3)

	// c2
	testGame.GameBoard[0][1] = Stack{[]Piece{blackCap, whiteFlat, blackFlat}}
	// b2
	testGame.GameBoard[1][1] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	// b3
	testGame.GameBoard[1][2] = Stack{[]Piece{blackFlat, blackFlat, whiteFlat, whiteFlat}}
	// a3
	testGame.GameBoard[2][2] = Stack{[]Piece{blackFlat, blackFlat, whiteFlat, whiteFlat}}

	blackVictory := testGame.IsRoadWin(Black)
	whiteVictory := testGame.IsRoadWin(White)

	switch {
	case blackVictory == false:
		t.Errorf("Failed to verify Black RoadWin: %v\n", blackVictory)
	case whiteVictory == true:
		t.Errorf("Got erroneous White RoadWin: %v\n", whiteVictory)
	}

}

func TestRoadWin(t *testing.T) {
	whiteWin := MakeGameBoard(8)
	whiteWin.GameID, _ = uuid.FromString("3fc74809-93eb-465d-a942-ef12427f83c5")
	gameIndex[whiteWin.GameID] = whiteWin

	// Board looks like this; two possible white roadwins
	//8 .o.o....
	//7 oooo....
	//6 o.o.....
	//5 o.o.....
	//4 oooooooo
	//3 o....o..
	//2 .....o..
	//1 .....o..
	// abcdefgh
	whiteWin.GameBoard[0][1] = Stack{[]Piece{whiteCap, whiteFlat, blackFlat}}
	whiteWin.GameBoard[0][3] = Stack{[]Piece{whiteCap, whiteFlat, blackFlat}}

	whiteWin.GameBoard[1][0] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}
	whiteWin.GameBoard[1][1] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	whiteWin.GameBoard[1][2] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}
	whiteWin.GameBoard[1][3] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}

	whiteWin.GameBoard[2][0] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}
	whiteWin.GameBoard[2][2] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}

	whiteWin.GameBoard[3][0] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}
	whiteWin.GameBoard[3][2] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}
	whiteWin.GameBoard[4][4] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[4][5] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[4][6] = Stack{[]Piece{whiteFlat}}
	// whiteWin.GameBoard[4][7] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[4][3] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[4][2] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[4][1] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[4][0] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[5][0] = Stack{[]Piece{whiteFlat}}

	whiteWin.GameBoard[5][5] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[6][5] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[7][5] = Stack{[]Piece{whiteFlat}}

	isover := whiteWin.IsGameOver()
	whowins, _ := whiteWin.WhoWins()
	fmt.Printf("8x8? %v who wins? %v winningPath: %v\n", isover, whowins, whiteWin.WinningPath)

	blackWin := MakeGameBoard(3)
	blackWin.GameBoard[0][0] = Stack{[]Piece{blackFlat}}
	blackWin.GameBoard[1][0] = Stack{[]Piece{blackFlat}}
	blackWin.GameBoard[1][1] = Stack{[]Piece{whiteCap, whiteFlat, blackFlat}}
	blackWin.GameBoard[2][0] = Stack{[]Piece{blackFlat}}
	isover = blackWin.IsGameOver()
	whowins, _ = blackWin.WhoWins()
	fmt.Printf("3x3? %v who wins? %v winningPath: %v\n", isover, whowins, blackWin.WinningPath)

	notAWin := MakeGameBoard(4)
	notAWin.GameBoard[0][0] = Stack{[]Piece{blackFlat}}
	notAWin.GameBoard[1][0] = Stack{[]Piece{blackFlat}}
	notAWin.GameBoard[1][1] = Stack{[]Piece{whiteCap, whiteFlat, blackFlat}}

	noWin := MakeGameBoard(4)

	testCases := []struct {
		game     *TakGame
		isOver   bool
		whoWon   string
		checkErr error
	}{
		{whiteWin, true, "White makes a road win!", nil},
		{blackWin, true, "Black makes a road win!", nil},
		{noWin, false, "", errors.New("game is not over, yet")},
		{notAWin, false, "", errors.New("game is not over, yet")},
	}
	for _, c := range testCases {
		isOver := c.game.IsGameOver()
		// fmt.Printf("==> Testing Roadwin %v: WP = %v\n", k, c.game.WinningPath)
		if isOver != c.isOver {
			t.Errorf("Expected gameOver: %+v, got %+v", c.isOver, isOver)
		}
		checkedWinner, checkErr := c.game.WhoWins()
		if checkedWinner != c.whoWon {
			t.Errorf("Problem: wanted winner '%v', got winner '%v'.\n", c.whoWon, checkedWinner)
		}
		if reflect.DeepEqual(checkErr, c.checkErr) != true {
			t.Errorf("Problem: wanted error '%v', got '%v'\n", c.checkErr, checkErr)
		}
	}

}

func TestUnCoords(t *testing.T) {
	whiteWin := MakeGameBoard(7)

	testCoords := []struct {
		y, x       int
		coords     string
		desiredErr error
	}{
		{0, 0, "a7", nil},
		{2, 2, "c5", nil},
		{3, 5, "f4", nil},
		{8, 0, "", errors.New("y '8' is out of bounds")},
	}

	for _, c := range testCoords {
		coords, err := whiteWin.UnTranslateCoords(c.y, c.x)
		if coords != c.coords {
			t.Errorf("%v, %v: wanted '%v', got '%v'", c.y, c.x, c.coords, coords)
		}
		if reflect.DeepEqual(err, c.desiredErr) != true {
			t.Errorf("%v, %v: wanted '%v', got '%v'", c.y, c.x, c.desiredErr, err)
		}
	}
}
