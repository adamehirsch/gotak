package main

import (
	"errors"
	"reflect"
	"testing"
)

func TestBoardLimits(t *testing.T) {
	testBoard := MakeGameBoard(5)
	firstPiece := Piece{"white", "flat"}
	secondPiece := Piece{"black", "flat"}
	thirdPiece := Piece{"white", "capstone"}
	testBoard.Grid[0][0] = Stack{[]Piece{firstPiece, secondPiece}}
	testBoard.Grid[0][2] = Stack{[]Piece{firstPiece, thirdPiece}}
	testBoard.Grid[3][3] = Stack{[]Piece{thirdPiece, secondPiece}}

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

	// if err != nil {
	// 	t.Errorf("Problem with coordinates %v: %v", testCoords, err)
	// }
}
