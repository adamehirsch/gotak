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
		testStack, err := testBoard.CheckSquare(c.coords)
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
		{"f1", false, errors.New("Problem at coordinates 'f1': coordinates 'f1' larger than board size: 5")},
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
	firstPiece := Piece{"white", "flat"}
	secondPiece := Piece{"black", "flat"}
	thirdPiece := Piece{"white", "capstone"}
	testBoard.Grid[4][1] = Stack{[]Piece{firstPiece, secondPiece}}
	testBoard.Grid[0][0] = Stack{[]Piece{firstPiece, thirdPiece}}
	testBoard.Grid[1][3] = Stack{[]Piece{thirdPiece, secondPiece}}

	// case-driven testing: The Bomb
	cases := []struct {
		Coords  string
		Problem error
	}{
		{"b1", errors.New("Could not place piece at occupied square b1")},
		{"a5", errors.New("Could not place piece at occupied square a5")},
		{"b2", nil},
		{"h1", errors.New("Could not place piece at h1: Problem at coordinates 'h1': coordinates 'h1' larger than board size: 5")},
	}

	for _, c := range cases {
		err := testBoard.PlacePiece(c.Coords, secondPiece)

		if reflect.DeepEqual(err, c.Problem) == false {
			t.Errorf("Returned error from coords %v was '%v': wanted '%v'\n", c.Coords, err, c.Problem)
		}

	}
}
