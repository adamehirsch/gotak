package main

import (
	"fmt"
	"net/http"
	// "github.com/satori/go.uuid"
)

type Piece struct {
	// one of "black" or "white"
	Color string
	// Type can be one of "flat", "standing", or "capstone"
	Type string
}

type Stack struct {
	// the top of the stack is at [0]
	Pieces []Piece
}

func SlashHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Gorilla!\n"))
}

func MakeGameBoard(size int) [][]string {

	board := make([][]string, size, size)

	for i := 0; i < size; i++ {
		fmt.Println("i is ", i)
		row := make([]string, size, size)
		board[i] = row
	}
	fmt.Println(board)
	return board
}

func main() {

	foo := MakeGameBoard(8)
	foo[1][1] = "X"
	fmt.Println(foo)

	// r := mux.NewRouter()
	// // Routes consist of a path and a handler function.
	// r.HandleFunc("/", SlashHandler)
	// r.HandleFunc("/newgam", SlashHandler)
	//
	// // Bind to a port and pass our router in
	// log.Fatal(http.ListenAndServe(":8000", r))
}
