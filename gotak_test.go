package main

import (
	"errors"
	"reflect"
	"testing"
)

func TestBoardSizeLimits(t *testing.T) {
	testBoard := MakeGameBoard(5)
	firstPiece := Piece{"white", "flat"}
	secondPiece := Piece{"black", "flat"}
	thirdPiece := Piece{"white", "capstone"}
	testBoard.Grid[4][0] = Stack{[]Piece{firstPiece, secondPiece}}
	testBoard.Grid[0][2] = Stack{[]Piece{firstPiece, thirdPiece}}
	testBoard.Grid[1][3] = Stack{[]Piece{thirdPiece, secondPiece}}

	// case-driven testing: The Bomb
	cases := []struct {
		coords  string
		stack   Stack
		problem error
	}{
		{"a1", Stack{[]Piece{firstPiece, secondPiece}}, nil},
		{"d4", Stack{[]Piece{thirdPiece, secondPiece}}, nil},
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
	firstPiece := Piece{"white", "flat"}
	secondPiece := Piece{"black", "flat"}
	thirdPiece := Piece{"white", "capstone"}
	testBoard.Grid[4][1] = Stack{[]Piece{firstPiece, secondPiece}}
	testBoard.Grid[0][0] = Stack{[]Piece{firstPiece, thirdPiece}}
	testBoard.Grid[1][3] = Stack{[]Piece{thirdPiece, secondPiece}}

	// case-driven testing: The Bomb
	cases := []struct {
		Coords  string
		Empty   bool
		Problem error
	}{
		{"b1", false, nil},
		{"a5", false, nil},
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
	whiteFlat := Piece{"white", "flat"}
	blackFlat := Piece{"black", "flat"}
	whiteCap := Piece{"white", "capstone"}
	testBoard.Grid[4][1] = Stack{[]Piece{whiteFlat, blackFlat}}
	testBoard.Grid[0][0] = Stack{[]Piece{whiteFlat, whiteCap}}
	testBoard.Grid[1][3] = Stack{[]Piece{whiteCap, blackFlat}}

	// case-driven testing: The Bomb
	cases := []struct {
		placement Placement
		Problem   error
	}{
		{Placement{Coords: "b1", Piece: whiteCap}, errors.New("bad placement request: Cannot place piece on occupied square b1")},
		{Placement{Coords: "a5", Piece: blackFlat}, errors.New("bad placement request: Cannot place piece on occupied square a5")},
		{Placement{Coords: "b2", Piece: whiteFlat}, nil},
		{Placement{Coords: "h1", Piece: blackFlat}, errors.New("bad placement request: h1: coordinates 'h1' larger than board size: 5")},
	}

	for _, c := range cases {
		err := testBoard.PlacePiece(c.placement)
		if testBoard.IsDarkTurn == true {
			testBoard.IsDarkTurn = false
		} else {
			testBoard.IsDarkTurn = true
		}
		if reflect.DeepEqual(err, c.Problem) == false {
			t.Errorf("Returned error from coords %v was '%v': wanted '%v'\n", c.placement.Coords, err, c.Problem)
		}

	}
}

func TestTurnTaking(t *testing.T) {
	testBoard := MakeGameBoard(5)
	whiteFlat := Piece{"white", "flat"}
	bogusFlat := Piece{"bogus", "flatworm"}
	blackFlat := Piece{"black", "flat"}
	whiteCap := Piece{"white", "capstone"}
	testBoard.Grid[4][1] = Stack{[]Piece{whiteFlat, blackFlat}}
	testBoard.Grid[0][0] = Stack{[]Piece{whiteFlat, whiteCap}}
	testBoard.Grid[1][3] = Stack{[]Piece{whiteCap, blackFlat}}
	testBoard.IsDarkTurn = true

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
		if testBoard.IsDarkTurn == true {
			testBoard.IsDarkTurn = false
		} else {
			testBoard.IsDarkTurn = true
		}

		if reflect.DeepEqual(err, c.Problem) == false {
			t.Errorf("Returned error from coords %v was '%v': wanted '%v'\n", c.placement.Coords, err, c.Problem)
		}

	}
}
