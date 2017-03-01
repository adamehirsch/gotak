package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	// "github.com/satori/go.uuid"
)

// Piece is the most basic element of a Tak game. One of two colors; one of three types.
type Piece struct {
	// one of "black" or "white"
	Color string
	// Type can be one of "flat", "standing", or "capstone"
	Type string
}

// Stack is probably a needless piece of structure; it's just a slice of Pieces
type Stack struct {
	// the top of the stack is at [0]
	Pieces []Piece
}

func SlashHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Gorilla!\n"))
}

// MakeGameBoard takes an integer size and returns a slice of slices of Stacks
func MakeGameBoard(size int) [][]Stack {

	board := make([][]Stack, size, size)

	for i := 0; i < size; i++ {
		row := make([]Stack, size, size)
		board[i] = row
	}
	return board
}

func main() {

	firstBoard := MakeGameBoard(3)
	firstPiece := Piece{"white", "flat"}
	secondPiece := Piece{"black", "flat"}

	firstBoard[0][0] = Stack{[]Piece{firstPiece, secondPiece}}
	fmt.Println(firstBoard)

	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", SlashHandler)
	r.HandleFunc("/newgam", SlashHandler)

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}
