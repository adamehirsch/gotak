package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	log "github.com/Sirupsen/logrus"

	jwt "github.com/dgrijalva/jwt-go"

	uuid "github.com/satori/go.uuid"
)

func init() {
	// funny that this seems to take precedence over the SetOutput call in gotak.go
	// log.SetOutput(ioutil.Discard)
	log.SetLevel(log.WarnLevel)
}

// this mockDB satisies the "Datastore" interface by having the methods below
type mockDB struct {
	takgame    TakGame
	playerid   uuid.UUID
	takplayer  TakPlayer
	playername string
}

func (mdb *mockDB) StoreTakGame(tg *TakGame) error {
	mdb.takgame = *tg
	return nil
}
func (mdb *mockDB) RetrieveTakGame(id uuid.UUID) (*TakGame, error) {
	log.Debug(fmt.Sprintf("retrieving game %v", mdb.takgame.GameID))
	return &mdb.takgame, nil
}
func (mdb *mockDB) StorePlayer(p *TakPlayer) error {
	mdb.takplayer = *p
	return nil
}
func (mdb *mockDB) RetrievePlayer(name string) (*TakPlayer, error) {
	return &mdb.takplayer, nil
}

func (mdb *mockDB) PlayerExists(n string) bool {
	return mdb.takplayer.Username == n
}

func TestBoardTooBig(t *testing.T) {
	testBoard, err := MakeGame(23)
	if testBoard != nil || err.Error() != "board size must be in the range 3 to 8 squares" {
		t.Errorf("Incorrect response from goofily large MakeGame request: %v, %v", testBoard, err.Error())
	}
}

func TestBoardSizeLimits(t *testing.T) {
	testBoard, _ := MakeGame(5)

	// a5
	testBoard.GameBoard[0][4] = Stack{[]Piece{whiteWall, blackFlat}}
	// c1
	testBoard.GameBoard[2][0] = Stack{[]Piece{whiteFlat, whiteFlat}}
	// d2
	testBoard.GameBoard[3][1] = Stack{[]Piece{blackWall, whiteFlat}}

	cases := []struct {
		coords  string
		stack   Stack
		problem error
	}{
		{"a5", Stack{[]Piece{whiteWall, blackFlat}}, nil},
		{"d2", Stack{[]Piece{blackWall, whiteFlat}}, nil},
		{"b2", Stack{}, nil},
		{"f1", Stack{}, errors.New("coordinates 'f1' larger than board size: 5")},
	}

	for _, c := range cases {
		testStack, err := testBoard.SquareContents(c.coords)
		// testBoard.DrawStackTops()

		if reflect.DeepEqual(testStack, c.stack) == false {
			t.Errorf("Returned stack from coords %v was %v: wanted %v\n", c.coords, testStack, c.stack)
		}

		if reflect.DeepEqual(err, c.problem) == false {
			t.Errorf("Returned error from coords %v was '%v': wanted '%v'\n", c.coords, err, c.problem)
		}
	}
}

func TestBoardSquareEmpty(t *testing.T) {
	testBoard, _ := MakeGame(5)
	// b5
	testBoard.GameBoard[1][4] = Stack{[]Piece{whiteWall, blackFlat}}
	// c1
	testBoard.GameBoard[2][0] = Stack{[]Piece{whiteFlat, whiteFlat}}
	// d2
	testBoard.GameBoard[3][1] = Stack{[]Piece{blackWall, whiteFlat}}

	// case-driven testing: The Bomb
	cases := []struct {
		Coords  string
		Empty   bool
		Problem error
	}{
		{"b5", false, nil},
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
	testBoard, _ := MakeGame(5)
	// b5
	testBoard.GameBoard[1][4] = Stack{[]Piece{whiteFlat, blackFlat}}
	// a1
	testBoard.GameBoard[0][0] = Stack{[]Piece{whiteFlat, whiteFlat}}
	// d2
	testBoard.GameBoard[3][1] = Stack{[]Piece{whiteCap, blackFlat}}
	testBoard.IsBlackTurn = false
	// let's just pretend we've got a longer turn history already in place
	testBoard.TurnHistory = append(testBoard.TurnHistory, Movement{}, Placement{})
	testBoard.WhitePlayer = "testWhite"
	testBoard.BlackPlayer = "testBlack"
	log.Debug(fmt.Sprintf("turnhistory length: %v", len(testBoard.TurnHistory)))

	cases := []struct {
		placement Placement
		Problem   error
	}{
		{Placement{Coords: "b5", Piece: whiteFlat}, errors.New("bad placement request: Cannot place piece on occupied square b5")},
		{Placement{Coords: "a1", Piece: blackFlat}, errors.New("bad placement request: Cannot place piece on occupied square a1")},
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
	testBoard, _ := MakeGame(5)
	bogusFlat := Piece{"bogus", "flatworm"}

	// b5
	testBoard.GameBoard[1][4] = Stack{[]Piece{whiteFlat, blackFlat}}
	// a1
	testBoard.GameBoard[0][0] = Stack{[]Piece{whiteFlat, whiteCap}}
	// d2
	testBoard.GameBoard[3][1] = Stack{[]Piece{whiteCap, blackFlat}}
	testBoard.IsBlackTurn = true
	testBoard.TurnHistory = append(testBoard.TurnHistory, Movement{}, Placement{})
	testBoard.WhitePlayer = "testWhite"
	testBoard.BlackPlayer = "testBlack"

	// case-driven testing: The Bomb
	cases := []struct {
		placement Placement
		Problem   error
	}{
		{Placement{Coords: "b5", Piece: blackFlat}, errors.New("bad placement request: Cannot place piece on occupied square b5")},
		{Placement{Coords: "a1", Piece: whiteCap}, errors.New("bad placement request: Cannot place piece on occupied square a1")},
		{Placement{Coords: "b2", Piece: blackFlat}, nil},
		{Placement{Coords: "a4", Piece: blackFlat}, errors.New("bad placement request: Cannot place black piece on white turn")},
		{Placement{Coords: "b3", Piece: bogusFlat}, errors.New("bad placement request: Invalid piece color 'bogus'")},
		{Placement{Coords: "h1", Piece: whiteFlat}, errors.New("bad placement request: h1: coordinates 'h1' larger than board size: 5")},
	}

	for _, c := range cases {
		err := testBoard.PlacePiece(c.placement)
		// log.WithFields(log.Fields{"isBlackTurn": testBoard.IsBlackTurn, "coords": c.placement.Coords}).Debug("turntaking, after")

		if reflect.DeepEqual(err, c.Problem) == false {
			t.Errorf("Returned error from coords %v was '%v': wanted '%v'\n", c.placement.Coords, err, c.Problem)
		}

	}
}

func TestEmptySquareDetection(t *testing.T) {
	testGame, _ := MakeGame(5)

	// b2
	testGame.GameBoard[1][3] = Stack{[]Piece{whiteCap, whiteFlat, blackFlat}}
	// c2
	testGame.GameBoard[2][1] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	// a5
	testGame.GameBoard[0][4] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}
	// d4
	testGame.GameBoard[3][1] = Stack{[]Piece{blackCap, whiteFlat, blackFlat, whiteFlat, blackFlat}}
	// c3
	testGame.GameBoard[2][2] = Stack{[]Piece{whiteWall}}
	testGame.IsBlackTurn = false
	testGame.TurnHistory = append(testGame.TurnHistory, Movement{}, Placement{})
	testGame.WhitePlayer = "testWhite"
	testGame.BlackPlayer = "testBlack"

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
	if err := testGame.MoveStack(c4Move); err != nil {
		t.Errorf("problem moving: %v th is %v", err, len(testGame.TurnHistory))
	}

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
	testBoard, _ := MakeGame(5)

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
	testBoard, _ := MakeGame(5)
	// e2
	testBoard.GameBoard[4][1] = Stack{[]Piece{whiteFlat, blackFlat}}
	// a5
	testBoard.GameBoard[0][4] = Stack{[]Piece{whiteFlat, blackFlat}}
	// e1
	testBoard.GameBoard[4][0] = Stack{[]Piece{whiteFlat, blackFlat}}
	// e3
	testBoard.GameBoard[4][2] = Stack{[]Piece{blackFlat, whiteFlat}}
	//d1
	testBoard.GameBoard[3][0] = Stack{[]Piece{whiteFlat, blackFlat, whiteFlat, blackFlat, whiteFlat, blackFlat, whiteFlat, blackFlat}}
	testBoard.IsBlackTurn = false
	testBoard.TurnHistory = append(testBoard.TurnHistory, Placement{}, Placement{})

	cases := []struct {
		move    Movement
		Problem error
	}{
		{Movement{Coords: "a5", Direction: "+", Carry: 1, Drops: []int{1}}, errors.New("Stack movement ([1]) would exceed top board edge")},
		{Movement{Coords: "e3", Direction: "+", Carry: 1, Drops: []int{1}}, errors.New("cannot move black-topped stack on white's turn")},
		{Movement{Coords: "e1", Direction: "+", Carry: 3, Drops: []int{1}}, errors.New("Stack at e1 is 2 high - cannot carry 3 pieces")},
		{Movement{Coords: "d1", Direction: "+", Carry: 6, Drops: []int{2, 2, 2}}, errors.New("Requested carry of 6 pieces exceeds board carry limit: 5")},
		{Movement{Coords: "d1", Direction: "+", Carry: 5, Drops: []int{2, 2, 2}}, errors.New("Requested drops ([2 2 2]) exceed number of pieces carried (5)")},
		{Movement{Coords: "d1", Direction: "+", Carry: 5, Drops: []int{2, 0, 2}}, errors.New("Stack movements ([2 0 2]) include a drop less than 1: 0")},
		{Movement{Coords: "a5", Direction: "<", Carry: 1, Drops: []int{1}}, errors.New("Stack movement ([1]) would exceed left board edge")},
		{Movement{Coords: "a5", Direction: "h", Carry: 1, Drops: []int{1}}, errors.New("can't parse move direction 'h'")},
		{Movement{Coords: "e1", Direction: "-", Carry: 1, Drops: []int{1}}, errors.New("Stack movement ([1]) would exceed bottom board edge")},
		{Movement{Coords: "e1", Direction: ">", Carry: 1, Drops: []int{1}}, errors.New("Stack movement ([1]) would exceed right board edge")},
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
	testGame, _ := MakeGame(5)

	// b2
	testGame.GameBoard[1][1] = Stack{[]Piece{whiteCap, whiteFlat, blackFlat}}
	// c2
	testGame.GameBoard[2][1] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	// a1
	testGame.GameBoard[0][0] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}
	// d4
	testGame.GameBoard[3][3] = Stack{[]Piece{blackCap, whiteFlat, blackFlat, whiteFlat, blackFlat}}
	// c3
	testGame.GameBoard[2][2] = Stack{[]Piece{whiteWall}}
	// testGame.DrawStackTops()

	cases := []struct {
		coords       string
		coordsAround []Coords
	}{
		{"b2", []Coords{Coords{2, 1}}},
		{"b5", nil},
		{"c2", []Coords{Coords{1, 1}, Coords{2, 2}}},
		{"a1", nil},
		{"b1", []Coords{Coords{1, 1}}},
	}

	for _, c := range cases {
		x, y, _ := testGame.TranslateCoords(c.coords)
		coordsAround := testGame.NearbyOccupiedCoords(x, y, NorthSouth)
		if reflect.DeepEqual(coordsAround, c.coordsAround) == false {
			t.Errorf("%v Wanted coords %v got CoordsAround %v\n", c.coords, c.coordsAround, coordsAround)
		}
	}
}

func TestUnCoords(t *testing.T) {
	whiteWin, _ := MakeGame(6)

	testCoords := []struct {
		x, y       int
		coords     string
		desiredErr error
	}{
		{0, 0, "a1", nil},
		{2, 2, "c3", nil},
		{3, 5, "d6", nil},
		{8, 0, "", errors.New("x '8' is out of bounds")},
		{0, 9, "", errors.New("y '9' is out of bounds")},
	}

	for _, c := range testCoords {
		coords, err := whiteWin.UnTranslateCoords(c.x, c.y)
		if coords != c.coords {
			t.Errorf("%v, %v: wanted '%v', got '%v'", c.y, c.x, c.coords, coords)
		}
		if reflect.DeepEqual(err, c.desiredErr) != true {
			t.Errorf("%v, %v: wanted '%v', got '%v'", c.y, c.x, c.desiredErr, err)
		}
	}
}

func TestTranslateCoords(t *testing.T) {
	whiteWin, _ := MakeGame(6)

	testCoords := []struct {
		x, y       int
		coords     string
		desiredErr error
	}{
		{0, 2, "a3", nil},
		{2, 5, "c6", nil},
		{3, 5, "d6", nil},
		{-1, -1, "i6", errors.New("Could not interpret coordinates 'i6'")},
		{-1, -1, "m-1", errors.New("Could not interpret coordinates 'm-1'")},
	}

	for _, c := range testCoords {
		x, y, err := whiteWin.TranslateCoords(c.coords)

		if reflect.DeepEqual(err, c.desiredErr) != true {
			t.Errorf("%v, %v: wanted '%v', got '%v'", c.y, c.x, c.desiredErr, err)
		}
		if x != c.x || y != c.y {
			t.Errorf("%v: wanted %v, %v, got %v, %v", c.coords, c.x, c.y, x, y)
		}
	}
}

func TestPathSearch(t *testing.T) {
	testGame, _ := MakeGame(3)

	// b1
	testGame.GameBoard[1][0] = Stack{[]Piece{blackCap, whiteFlat, blackFlat}}
	// b2
	testGame.GameBoard[1][1] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	// c2
	testGame.GameBoard[2][1] = Stack{[]Piece{blackFlat, blackFlat, whiteFlat, whiteFlat}}
	// c3
	testGame.GameBoard[2][2] = Stack{[]Piece{blackFlat, blackFlat, whiteFlat, whiteFlat}}

	blackVictory := testGame.IsRoadWin(Black)
	whiteVictory := testGame.IsRoadWin(White)

	// testGame.DrawStackTops()
	switch {
	case blackVictory == false:
		t.Errorf("Failed to verify Black RoadWin: %v\n", blackVictory)
	case whiteVictory == true:
		t.Errorf("Got erroneous White RoadWin: %v\n", whiteVictory)
	}

}

func TestRoadWin(t *testing.T) {
	whiteWin, _ := MakeGame(8)

	// whiteWin.GameBoard[2][0] = Stack{[]Piece{whiteCap, whiteFlat, blackFlat}}
	whiteWin.GameBoard[3][0] = Stack{[]Piece{whiteCap, whiteFlat, blackFlat}}

	whiteWin.GameBoard[0][1] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}
	whiteWin.GameBoard[1][1] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	whiteWin.GameBoard[2][1] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}
	whiteWin.GameBoard[3][1] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}

	whiteWin.GameBoard[0][2] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}
	whiteWin.GameBoard[2][2] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}

	whiteWin.GameBoard[0][3] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}
	whiteWin.GameBoard[2][3] = Stack{[]Piece{whiteFlat, blackFlat, blackFlat, whiteFlat, whiteFlat}}
	whiteWin.GameBoard[4][3] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[3][4] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[4][4] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[5][3] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[6][3] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[7][3] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[3][4] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[2][4] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[1][4] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[0][4] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[0][5] = Stack{[]Piece{whiteFlat}}

	whiteWin.GameBoard[5][4] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[5][5] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[5][6] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[5][7] = Stack{[]Piece{whiteFlat}}

	blackWin, _ := MakeGame(3)
	blackWin.GameBoard[0][0] = Stack{[]Piece{blackFlat}}
	blackWin.GameBoard[0][1] = Stack{[]Piece{blackFlat}}
	blackWin.GameBoard[1][1] = Stack{[]Piece{blackFlat}}
	blackWin.GameBoard[2][1] = Stack{[]Piece{blackFlat}}
	blackWin.GameBoard[1][2] = Stack{[]Piece{blackFlat}}

	notARoadWin, _ := MakeGame(3)
	notARoadWin.GameBoard[0][0] = Stack{[]Piece{blackFlat}}
	notARoadWin.GameBoard[1][0] = Stack{[]Piece{whiteWall}}
	notARoadWin.GameBoard[2][0] = Stack{[]Piece{whiteFlat}}
	notARoadWin.GameBoard[0][1] = Stack{[]Piece{whiteFlat}}
	notARoadWin.GameBoard[0][2] = Stack{[]Piece{blackCap, whiteFlat, blackFlat}}
	notARoadWin.GameBoard[1][1] = Stack{[]Piece{blackFlat}}
	notARoadWin.GameBoard[1][2] = Stack{[]Piece{whiteWall, whiteFlat, blackFlat}}
	notARoadWin.GameBoard[2][1] = Stack{[]Piece{blackCap}}
	notARoadWin.GameBoard[2][2] = Stack{[]Piece{whiteWall, whiteFlat, blackFlat}}

	noWin, _ := MakeGame(4)

	revWin, _ := MakeGame(5)
	revWin.GameBoard[3][0] = Stack{[]Piece{blackFlat}}
	revWin.GameBoard[3][1] = Stack{[]Piece{blackFlat}}
	revWin.GameBoard[3][2] = Stack{[]Piece{blackFlat}}
	revWin.GameBoard[2][2] = Stack{[]Piece{blackFlat}}
	revWin.GameBoard[1][2] = Stack{[]Piece{blackFlat}}
	revWin.GameBoard[3][3] = Stack{[]Piece{blackFlat}}
	revWin.GameBoard[1][3] = Stack{[]Piece{blackFlat}}
	revWin.GameBoard[1][4] = Stack{[]Piece{blackFlat}}
	revWin.GameBoard[3][4] = Stack{[]Piece{blackFlat}}

	revWin.GameBoard[4][2] = Stack{[]Piece{whiteFlat}}

	testCases := []struct {
		game     *TakGame
		isOver   bool
		whoWon   string
		checkErr error
	}{
		{whiteWin, true, "White makes a road win!", nil},
		{blackWin, true, "Black makes a road win!", nil},
		{notARoadWin, true, "White makes a Flat Win!", nil},
		{noWin, false, "", errors.New("game is not over, yet")},
		{revWin, true, "Black makes a road win!", nil},
	}
	for _, c := range testCases {
		isOver := c.game.IsGameOver()
		if isOver != c.isOver {
			t.Errorf("Expected gameOver: %+v, got %+v", c.isOver, isOver)
		}
		checkedWinner, checkErr := c.game.WhoWins()
		// c.game.DrawStackTops()
		if checkedWinner != c.whoWon {
			t.Errorf("Problem: wanted winner '%v', got winner '%v'.\n", c.whoWon, checkedWinner)
		}
		if reflect.DeepEqual(checkErr, c.checkErr) != true {
			t.Errorf("Problem: wanted error '%v', got '%v'\n", c.checkErr, checkErr)
		}
	}

}

func TestGameEnd(t *testing.T) {
	testOne, _ := MakeGame(4)
	testOne.GameBoard[3][0] = Stack{[]Piece{whiteFlat}}
	testOne.GameBoard[3][1] = Stack{[]Piece{whiteFlat}}
	testOne.GameBoard[2][1] = Stack{[]Piece{whiteFlat}}
	testOne.GameBoard[2][2] = Stack{[]Piece{whiteFlat}}

	testTwo, _ := MakeGame(4)
	testTwo.GameBoard[0][0] = Stack{[]Piece{whiteFlat}}
	testTwo.GameBoard[1][0] = Stack{[]Piece{blackWall}}
	testTwo.GameBoard[2][0] = Stack{[]Piece{whiteFlat}}
	testTwo.GameBoard[3][0] = Stack{[]Piece{blackWall}}

	testTwo.GameBoard[0][1] = Stack{[]Piece{blackWall}}
	testTwo.GameBoard[1][1] = Stack{[]Piece{whiteFlat}}
	testTwo.GameBoard[2][1] = Stack{[]Piece{blackWall}}
	testTwo.GameBoard[3][1] = Stack{[]Piece{whiteFlat}}

	testTwo.GameBoard[0][2] = Stack{[]Piece{whiteFlat}}
	testTwo.GameBoard[1][2] = Stack{[]Piece{blackWall}}
	testTwo.GameBoard[2][2] = Stack{[]Piece{blackWall}}
	testTwo.GameBoard[3][2] = Stack{[]Piece{whiteFlat}}

	testTwo.GameBoard[0][3] = Stack{[]Piece{blackWall}}
	testTwo.GameBoard[3][3] = Stack{[]Piece{whiteFlat}}

	testThree, _ := MakeGame(4)
	testThree.GameBoard[0][0] = Stack{[]Piece{blackFlat}}
	testThree.GameBoard[1][0] = Stack{[]Piece{blackWall}}
	testThree.GameBoard[2][0] = Stack{[]Piece{whiteFlat}}
	testThree.GameBoard[3][0] = Stack{[]Piece{blackWall}}

	testThree.GameBoard[0][1] = Stack{[]Piece{blackWall}}
	testThree.GameBoard[1][1] = Stack{[]Piece{whiteFlat}}
	testThree.GameBoard[2][1] = Stack{[]Piece{blackWall}}
	testThree.GameBoard[3][1] = Stack{[]Piece{whiteWall}}

	testThree.GameBoard[0][2] = Stack{[]Piece{whiteFlat}}
	testThree.GameBoard[1][2] = Stack{[]Piece{blackWall}}
	testThree.GameBoard[2][2] = Stack{[]Piece{blackWall}}
	testThree.GameBoard[3][2] = Stack{[]Piece{whiteFlat}}

	testThree.GameBoard[0][3] = Stack{[]Piece{blackWall}}
	testThree.GameBoard[3][3] = Stack{[]Piece{whiteFlat}}

	testCases := []struct {
		game           *TakGame
		isOverPreMove  bool
		whoWon         string
		isOverPostMove bool
		checkErr       error
	}{
		{testOne, false, "White makes a road win!", true, nil},
		{testTwo, false, "Game ends in a draw!", true, nil},
		{testThree, false, "Black makes a Flat Win!", true, nil},
	}
	for _, c := range testCases {
		isOverPreMove := c.game.IsGameOver()
		if isOverPreMove != c.isOverPreMove {
			t.Errorf("Premove: Expected gameOver: %+v, got %+v", c.isOverPreMove, isOverPreMove)
		}

		// c4
		c.game.GameBoard[2][3] = Stack{[]Piece{whiteFlat}}
		// b4
		c.game.GameBoard[1][3] = Stack{[]Piece{blackFlat}}
		isOverPostMove := c.game.IsGameOver()
		if isOverPostMove != c.isOverPostMove {
			t.Errorf("Postmove: Expected gameOver: %+v, got %+v", c.isOverPostMove, isOverPostMove)
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

func TestTooManyPieces(t *testing.T) {
	testOne, _ := MakeGame(3)
	testOne.GameBoard[0][1] = Stack{[]Piece{whiteWall, blackFlat, whiteFlat}}
	testOne.GameBoard[0][2] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	testOne.GameBoard[1][2] = Stack{[]Piece{whiteWall, blackFlat, whiteFlat}}
	testOne.GameBoard[1][1] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	testOne.GameBoard[2][1] = Stack{[]Piece{whiteWall, blackFlat, whiteFlat}}
	testOne.GameBoard[2][2] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	testOne.IsBlackTurn = false
	testOne.TurnHistory = append(testOne.TurnHistory, Placement{}, Placement{})
	testOne.WhitePlayer = "testWhite"
	testOne.BlackPlayer = "testBlack"

	testTwo, _ := MakeGame(3)
	testTwo.GameBoard[0][1] = Stack{[]Piece{whiteWall, blackFlat, whiteFlat}}
	testTwo.GameBoard[0][2] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	testTwo.GameBoard[1][2] = Stack{[]Piece{whiteWall, blackFlat, whiteFlat}}
	testTwo.GameBoard[1][1] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	testTwo.GameBoard[2][1] = Stack{[]Piece{whiteWall, blackFlat, whiteFlat}}
	testTwo.GameBoard[2][2] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	testTwo.IsBlackTurn = true
	testTwo.TurnHistory = append(testTwo.TurnHistory, Placement{}, Placement{})
	testTwo.WhitePlayer = "testWhite"
	testTwo.BlackPlayer = "testBlack"

	testThree, _ := MakeGame(3)
	testThree.GameBoard[0][1] = Stack{[]Piece{whiteWall, blackFlat, whiteFlat}}
	testThree.GameBoard[0][2] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	testThree.GameBoard[1][2] = Stack{[]Piece{whiteWall, whiteFlat}}
	testThree.GameBoard[1][1] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	testThree.GameBoard[2][1] = Stack{[]Piece{whiteWall, whiteFlat}}
	testThree.GameBoard[2][2] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	testThree.IsBlackTurn = true
	testThree.WhitePlayer = "testWhite"
	testThree.BlackPlayer = "testBlack"

	testCases := []struct {
		game      *TakGame
		placement Placement
		winner    string
		gameOver  bool
	}{
		{testOne, Placement{Coords: "b1", Piece: whiteFlat}, "White makes a Flat win: piece limit reached!", true},
		{testTwo, Placement{Coords: "b1", Piece: blackWall}, "Black makes a Flat win: piece limit reached!", true},
		{testThree, Placement{Coords: "b1", Piece: blackWall}, "", false},
	}

	for _, c := range testCases {
		c.game.PlacePiece(c.placement)
		if c.gameOver != c.game.IsGameOver() {
			t.Errorf("Game should be over (%v) but is instead (%v)", c.gameOver, c.game.IsGameOver())
		}
		winner, _ := c.game.WhoWins()
		if winner != c.winner {
			t.Errorf("wanted winner %v, got %v", c.winner, winner)
		}
	}

}

func TestDrawStackTops(t *testing.T) {

	testGame, _ := MakeGame(3)

	// b1
	testGame.GameBoard[1][0] = Stack{[]Piece{blackCap, whiteFlat, blackFlat}}
	// b2
	testGame.GameBoard[1][1] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	// c2
	testGame.GameBoard[2][1] = Stack{[]Piece{blackFlat, blackFlat, whiteFlat, whiteFlat}}
	// c3
	testGame.GameBoard[2][2] = Stack{[]Piece{blackFlat, blackFlat, whiteFlat, whiteFlat}}
	testGame.IsGameOver() // highlight the winning path

	testThree, _ := MakeGame(3)
	testThree.GameBoard[0][1] = Stack{[]Piece{whiteWall, blackFlat, whiteFlat}}
	testThree.GameBoard[0][2] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	testThree.GameBoard[1][2] = Stack{[]Piece{whiteWall, whiteFlat}}
	testThree.GameBoard[1][1] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	testThree.GameBoard[2][1] = Stack{[]Piece{whiteWall, whiteFlat}}
	testThree.GameBoard[2][2] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}

	testCases := []struct {
		game        *TakGame
		desiredTops []string
	}{
		{testGame, []string{" ---------", "| .  . (B)|", "| . (B)(B)|", "| . (B) . |", " ---------"}},
		{testThree, []string{" ---------", "| B  W  B |", "| W  B  W |", "| .  .  . |", " ---------"}},
	}

	for _, c := range testCases {
		drawnTops := c.game.DrawStackTops()
		if !reflect.DeepEqual(drawnTops, c.desiredTops) {
			t.Errorf("Wanted to see\nstacktops %v\n saw\nstacktops %v\n", c.desiredTops, drawnTops)
		}
	}
}

func TestCapstoneStomping(t *testing.T) {
	testOne, _ := MakeGame(4)
	testOne.GameBoard[0][0] = Stack{[]Piece{whiteFlat, whiteFlat, blackFlat}}
	testOne.GameBoard[0][1] = Stack{[]Piece{blackWall}}
	testOne.TurnHistory = append(testOne.TurnHistory, Placement{}, Placement{})
	testOneMove := Movement{Direction: "+", Carry: 2, Drops: []int{1, 1}, Coords: "a1"}
	testOne.IsBlackTurn = false
	testOne.WhitePlayer = "testWhite"
	testOne.BlackPlayer = "testBlack"

	testTwo, _ := MakeGame(4)
	testTwo.GameBoard[0][0] = Stack{[]Piece{blackWall, whiteFlat, blackFlat}}
	testTwo.GameBoard[1][0] = Stack{[]Piece{whiteCap}}
	testTwo.TurnHistory = append(testTwo.TurnHistory, Placement{}, Placement{})
	testTwoMove := Movement{Direction: "<", Carry: 1, Drops: []int{1}, Coords: "b1"}
	testTwo.IsBlackTurn = false
	testTwo.WhitePlayer = "testWhite"
	testTwo.BlackPlayer = "testBlack"

	testThree, _ := MakeGame(5)
	//c4
	testThree.GameBoard[2][3] = Stack{[]Piece{whiteCap, blackFlat, whiteFlat}}
	//d4
	testThree.GameBoard[3][3] = Stack{[]Piece{blackWall}}
	testThree.TurnHistory = append(testThree.TurnHistory, Placement{}, Placement{})
	testThree.WhitePlayer = "testWhite"
	testThree.BlackPlayer = "testBlack"

	testThreeMove := Movement{Direction: ">", Carry: 2, Drops: []int{1, 1}, Coords: "c4"}
	testThree.IsBlackTurn = false

	testFour, _ := MakeGame(5)
	//c4
	testFour.GameBoard[2][3] = Stack{[]Piece{whiteCap, blackFlat, whiteFlat}}
	//d4
	testFour.GameBoard[3][3] = Stack{[]Piece{blackWall}}
	testFour.TurnHistory = append(testFour.TurnHistory, Placement{}, Placement{})

	testFourMove := Movement{Direction: ">", Carry: 2, Drops: []int{2}, Coords: "c4"}
	testFour.IsBlackTurn = false
	testFour.WhitePlayer = "testWhite"
	testFour.BlackPlayer = "testBlack"

	testFive, _ := MakeGame(5)
	//c4
	testFive.GameBoard[2][3] = Stack{[]Piece{whiteCap, blackFlat, whiteFlat}}
	//d4
	testFive.GameBoard[3][3] = Stack{[]Piece{blackCap}}
	testFive.TurnHistory = append(testFive.TurnHistory, Placement{}, Placement{})
	testFiveMove := Movement{Direction: ">", Carry: 1, Drops: []int{1}, Coords: "c4"}
	testFive.IsBlackTurn = false
	testFive.WhitePlayer = "testWhite"
	testFive.BlackPlayer = "testBlack"

	testCases := []struct {
		Game         *TakGame
		MoveErr      error
		Move         Movement
		StompedStack *Stack
		DesiredStack Stack
	}{
		{testOne, errors.New("invalid move: Can't flatten wall at a2: no capstone on moving stack"), testOneMove, &testOne.GameBoard[0][1], Stack{Pieces: []Piece{Piece{Black, Wall}}}},
		{testTwo, nil, testTwoMove, &testTwo.GameBoard[0][0], Stack{Pieces: []Piece{Piece{White, Capstone}, Piece{Black, Flat}, Piece{White, Flat}, Piece{Black, Flat}}}},
		{testThree, errors.New("invalid move: Can't flatten wall at d4: not on last drop of move sequence"), testThreeMove, &testThree.GameBoard[3][3], Stack{Pieces: []Piece{Piece{Black, Wall}}}},
		{testFour, errors.New("invalid move: Only allowed to flatten wall at d4 with 1 capstone, not 2 pieces"), testFourMove, &testFour.GameBoard[3][3], Stack{Pieces: []Piece{Piece{Black, Wall}}}},
		{testFive, errors.New("invalid move: Movement can't flatten a capstone at d4"), testFiveMove, &testFive.GameBoard[3][3], Stack{Pieces: []Piece{Piece{Black, Capstone}}}},
	}

	for _, c := range testCases {
		err := c.Game.MoveStack(c.Move)
		if reflect.DeepEqual(c.MoveErr, err) != true {
			t.Errorf("wanted error '%v', got '%v'", c.MoveErr, err)
		}
		if reflect.DeepEqual(*c.StompedStack, c.DesiredStack) != true {
			t.Errorf("Wanted stack %v, got stack %v\n", c.DesiredStack, *c.StompedStack)
		}
	}

}

func TestLoginHandler(t *testing.T) {

	testPlayer := TakPlayer{
		Username: "testPlayer",
		// password is "foobar"
		passwordHash: []byte("$2a$10$egXKY.SPgXWMkOUIFPC2JOPnWbaTLl3W2Vp5f9xZW9W1pktAPxCE2"),
	}

	env := DBenv{db: &mockDB{
		playername: "testPlayer",
		takplayer:  testPlayer,
	}}

	testCases := []struct {
		credentials []byte
		code        int
		message     string
	}{
		{
			credentials: []byte(`{"username": "testPlayer","password": "foobar"}`),
			code:        200,
			message:     "successfully logged in",
		},
		{
			credentials: []byte(`{"username": "testPlayer","password": "wrongbar"}`),
			code:        400,
			message:     "incorrect password\n",
		},
		{
			credentials: []byte(`{"username": "wrongPlayer","password": "wrongbar"}`),
			code:        422,
			message:     "No player named 'wrongPlayer' found\n",
		},
		{
			credentials: []byte(`{"username": "","password": "wrongbar"}`),
			code:        422,
			message:     "Missing player username or password\n",
		},
	}

	for _, c := range testCases {
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(c.credentials))
		errorHandler(env.Login).ServeHTTP(rec, req)
		resp := rec.Result()
		body, _ := ioutil.ReadAll(resp.Body)

		if resp.StatusCode != c.code {
			t.Errorf("wanted return code %v, got %v", c.code, resp.StatusCode)
		}
		if isJSON(string(body)) {
			loginResp := TakJWT{}
			json.Unmarshal(body, &loginResp)
			if loginResp.Message != c.message {
				t.Errorf("wanted return message '%v', got '%v'", c.message, loginResp.Message)
			}
			tokenString := loginResp.JWT
			token, _ := jwt.Parse(tokenString, jwtKeyFn)
			if !token.Valid {
				t.Error("invalid JWT token")

			}
		} else {
			if string(body) != c.message {
				t.Errorf("wanted return body '%v', got '%v'", c.message, string(body))
			}
		}

	}
}

func TestMoveHandler(t *testing.T) {
	testGame, _ := MakeGame(5)
	testGame.GameBoard[2][3] = Stack{[]Piece{whiteCap, blackFlat, whiteFlat}}
	testGame.BlackPlayer = "testBlack"
	testGame.WhitePlayer = "testWhite"
	testGame.TurnHistory = append(testGame.TurnHistory, Placement{}, Placement{})
	testGame.IsBlackTurn = false

	desiredBoard := makeGameBoard(5)
	desiredBoard[2][3] = Stack{[]Piece{blackFlat, whiteFlat}}
	desiredBoard[3][3] = Stack{[]Piece{whiteCap}}

	testWhite := TakPlayer{Username: "testWhite"}

	mockEnv := DBenv{db: &mockDB{
		takgame:    *testGame,
		takplayer:  testWhite,
		playername: "testWhite",
	}}

	testCases := []struct {
		player TakPlayer
		move   Movement
		game   *TakGame
		board  [][]Stack
	}{
		{testWhite, Movement{Coords: "c4", Direction: "+", Carry: 1, Drops: []int{1}}, testGame, desiredBoard},
	}
	for _, c := range testCases {
		playerToken := generateJWT(&(c.player), "test")
		loginResp := TakJWT{}
		json.Unmarshal(playerToken, &loginResp)
		movement, _ := json.Marshal(c.move)
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/v1/game/%v/move", testGame.GameID.String()), bytes.NewBuffer(movement))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", loginResp.JWT))

		genRouter(&mockEnv).ServeHTTP(rec, req)

		resp := rec.Result()
		body, _ := ioutil.ReadAll(resp.Body)
		var postMoveGame TakGame
		json.Unmarshal(body, &postMoveGame)
		if !reflect.DeepEqual(postMoveGame.GameBoard, c.game.GameBoard) {
			t.Errorf("board problem: \nwanted %v\n   got %v\n", c.game.GameBoard, postMoveGame.GameBoard)
		}
	}
}

func TestFirstTwoMoves(t *testing.T) {

	testGame, _ := MakeGame(5)
	testGame.BlackPlayer = "testBlack"
	testGame.WhitePlayer = "testWhite"
	testGame.IsBlackTurn = true

	// testWhite := TakPlayer{Username: "testWhite"}
	testBlack := TakPlayer{Username: "testBlack"}

	mockEnv := DBenv{db: &mockDB{
		takgame:    *testGame,
		takplayer:  testBlack,
		playername: "testBlack",
	}}

	testCases := []struct {
		place Placement
		resp  string
		code  int
	}{
		{Placement{Piece: Piece{Color: Black, Orientation: Wall}, Coords: "a2"}, "problem placing piece at a2: bad placement request: first two placements must be of the opponent's color\n", 409},
		{Placement{Piece: Piece{Color: White, Orientation: Wall}, Coords: "a2"}, "", 200},
	}

	for _, c := range testCases {
		playerToken := generateJWT(&(testBlack), "test")
		loginResp := TakJWT{}
		json.Unmarshal(playerToken, &loginResp)
		placement, _ := json.Marshal(c.place)
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/v1/game/%v/place", testGame.GameID.String()), bytes.NewBuffer(placement))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", loginResp.JWT))

		genRouter(&mockEnv).ServeHTTP(rec, req)

		resp := rec.Result()
		body, _ := ioutil.ReadAll(resp.Body)
		//
		if !isJSON(string(body)) {
			if string(body) != c.resp {
				t.Errorf("Wanted response '%v', but got '%v'", c.resp, string(body))
			}
		} else {

		}
		if resp.StatusCode != c.code {
			t.Errorf("Wanted return code %v, got %v", c.code, resp.StatusCode)
		}
	}
}

func TestNewGameHandler(t *testing.T) {

	testBlack := TakPlayer{Username: "testBlack"}

	mockEnv := DBenv{db: &mockDB{
		takplayer:  testBlack,
		playername: "testBlack",
	}}

	testCases := []struct {
		size int
		code int
	}{
		{4, 200},
		{9, 500},
	}

	for _, c := range testCases {
		playerToken := generateJWT(&(testBlack), "test")
		loginResp := TakJWT{}
		json.Unmarshal(playerToken, &loginResp)
		rec := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", fmt.Sprintf("/v1/game/new/%v", c.size), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", loginResp.JWT))

		genRouter(&mockEnv).ServeHTTP(rec, req)

		resp := rec.Result()
		if resp.StatusCode != c.code {
			t.Errorf("Wanted return code %v, got %v", c.code, resp.StatusCode)
		}
	}
}

func TestCanShow(t *testing.T) {
	tg, _ := MakeGame(4)
	tg.BlackPlayer = "testBlack"
	tg.WhitePlayer = "testWhite"
	tg.GameOwner = "testOwner"

	testCases := []struct {
		requester string
		canShow   bool
	}{
		{"testBlack", true},
		{"testWhite", true},
		{"testOwner", true},
		{"shouldFail", false},
	}

	for _, c := range testCases {
		testPlayer := TakPlayer{
			Username: c.requester,
		}
		if tg.CanShow(&testPlayer) != c.canShow {
			t.Errorf("Wanted %v for player %v, got %v", c.canShow, c.requester, tg.CanShow(&testPlayer))
		}
	}
}

// isJSON just checks to see whether a string is valid JSON
func isJSON(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}
