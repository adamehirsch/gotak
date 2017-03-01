package main

import (
	"fmt"
	"net/http"

	"github.com/satori/go.uuid"
)

// Piece is the most basic element of a Tak game. One of two colors; one of three types.
type Piece struct {
	// one of "black" or "white"
	color string
	// Type can be one of "flat", "standing", or "capstone"
	orientation string
}

// Stack is potentially a needless piece of structure; it's just a slice of Pieces
type Stack struct {
	// the top of the stack is at [0]
	pieces []Piece
}

// Board is an NxN grid of spaces, optionally occupied by Stacks of Pieces. A given Board has a guaranteed unique uuid
type Board struct {
	uuid uuid.UUID
	grid [][]Stack
}

// func MakeRandomBoard() Board {
// 	firstBoard := MakeGameBoard(3)
//
// 	firstPiece := Piece{"white", "flat"}
// 	secondPiece := Piece{"black", "flat"}
// 	firstBoard.grid[0][0] = Stack{[]Piece{firstPiece, secondPiece}}
// 	return firstBoard
// }

// MakeGameBoard takes an integer size and returns a slice of slices of Stacks
func MakeGameBoard(size int) Board {
	// each board gets a unique, random UUIDv4
	newUUID := uuid.NewV4()

	// first make the rows...
	newGrid := make([][]Stack, size, size)

	// ... then populate with the columns of spaces
	for i := 0; i < size; i++ {
		row := make([]Stack, size, size)
		newGrid[i] = row
	}

	newBoard := Board{newUUID, newGrid}
	// newBoard.grid = newGrid
	// fmt.Println("2", newBoard)
	// newBoard.uuid = newUUID
	// fmt.Println("3", newBoard)

	return newBoard
}

func NewGameHandler(w http.ResponseWriter, r *http.Request) {
	return
}

func main() {

	testBoard := MakeGameBoard(2)
	fmt.Println(testBoard)

	// r := mux.NewRouter()
	// // Routes consist of a path and a handler function.
	// r.HandleFunc("/", SlashHandler)
	// r.HandleFunc("/newgame", NewGameHandler)
	//
	// // Bind to a port and pass our router in
	// log.Fatal(http.ListenAndServe(":8000", r))
}

func SlashHandler(w http.ResponseWriter, r *http.Request) {
	// fooBoard := MakeRandomBoard()
	w.Write([]byte("Gorilla!\n"))
}
