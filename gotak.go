package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

func main() {

	whiteWin := MakeGameBoard(7)
	whiteWin.GameID, _ = uuid.FromString("3fc74809-93eb-465d-a942-ef12427f83c5")
	gameIndex[whiteWin.GameID] = whiteWin

	whiteFlat := Piece{White, "flat"}
	blackFlat := Piece{Black, "flat"}
	// whiteWall := Piece{White, "wall"}
	blackWall := Piece{Black, "wall"}
	whiteCap := Piece{White, "capstone"}
	// blackCap := Piece{Black, "capstone"}

	// Board looks like this.
	// .o.o...
	// oooo...
	// o.o....
	// o.o....
	// ooooooo
	// o....o.
	// .....o.
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
	whiteWin.GameBoard[4][5] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[4][6] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[4][4] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[4][3] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[4][2] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[4][1] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[4][0] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[5][0] = Stack{[]Piece{whiteFlat}}

	whiteWin.GameBoard[5][5] = Stack{[]Piece{whiteFlat}}
	whiteWin.GameBoard[6][5] = Stack{[]Piece{whiteFlat}}
	if whiteWin.IsGameOver() {
		fmt.Println(whiteWin.WhoWins())
	}

	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", SlashHandler)
	r.HandleFunc("/newgame/{boardSize}", NewGameHandler)
	r.HandleFunc("/showgame/{gameID}", ShowGameHandler)
	r.Handle("/action/{action}/{gameID}", webHandler(ActionHandler)).Methods("PUT")

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}
