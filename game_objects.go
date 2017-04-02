package main

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

// Various constants for use throughout the game
const (
	Flat       string = "flat"
	Standing   string = "standing"
	Capstone   string = "capstone"
	Black      string = "black"
	White      string = "white"
	NorthSouth string = "NS"
	WestEast   string = "WE"
)

// Piece is the most basic element of a Tak game. One of two colors; one of three types.
type Piece struct {
	// one of "black" or "white"
	Color string `json:"color"`
	// Type can be one of "flat", "standing", or "capstone"
	Orientation string `json:"orientation"`
}

var whiteFlat = Piece{"white", "flat"}
var blackFlat = Piece{"black", "flat"}
var whiteCap = Piece{"white", "capstone"}
var blackCap = Piece{"black", "capstone"}
var whiteWall = Piece{"white", "wall"}
var blackWall = Piece{"black", "wall"}

// Stack is just a slice of Pieces.
type Stack struct {
	// Note: the "top" of the stack is at [0]
	Pieces []Piece
}

// TakPlayer describes a human player
type TakPlayer struct {
	Name        string     `json:"name"`
	PlayerID    uuid.UUID  `json:"playerID"`
	PlayedGames []*TakGame `json:"playedGames"`
}

// TakGame is the general object representing an entire game, including a board, an id, and some metadata.
type TakGame struct {
	GameID      uuid.UUID  `json:"gameID"`
	GameBoard   [][]Stack  `json:"gameBoard"`
	IsBlackTurn bool       `json:"isBlackTurn"`
	BlackWinner bool       `json:"BlackWinner"`
	WhiteWinner bool       `json:"WhiteWinner"`
	RoadWin     bool       `json:"roadWin"`
	FlatWin     bool       `json:"flatWin"`
	DrawGame    bool       `json:"drawGame"`
	GameOver    bool       `json:"gameOver"`
	GameWinner  uuid.UUID  `json:"gameWinner"`
	WinningPath []Coords   `json:"winningPath"`
	StartTime   time.Time  `json:"startTime"`
	WinTime     time.Time  `json:"winTime"`
	BlackPlayer *TakPlayer `json:"blackPlayerID"`
	WhitePlayer *TakPlayer `json:"whitePlayerID"`
	Size        int        `json:"size"`
}

// PieceLimits is a map of gridsize to piece limits per player
// note there's no limit listed for 7x7 games, which are "rarely played"
var PieceLimits = map[int]int{
	3: 10,
	4: 15,
	5: 21,
	6: 30,
	8: 50,
}

// I'll need some way to keep multiple boards stored and accessible; a map between UUID and Board might be just the ticket.
var gameIndex = make(map[uuid.UUID]*TakGame)

// MakeGameBoard takes an integer size and returns a &GameBoard
func MakeGameBoard(size int) *TakGame {

	// each board gets a unique, random UUIDv4
	newUUID := uuid.NewV4()

	// first make the rows...
	newGameBoard := make([][]Stack, size, size)

	// ... then populate with the columns of spaces
	for i := 0; i < size; i++ {
		row := make([]Stack, size, size)
		newGameBoard[i] = row
	}

	newTakGame := TakGame{GameID: newUUID, GameBoard: newGameBoard}
	newTakGame.Size = size
	gameIndex[newUUID] = &newTakGame
	return &newTakGame
}

// LetterMap converts Tak x-values (letters) to their start-at-zero grid index value. 8x8 games are the max size.
var LetterMap = map[string]int{
	"a": 0,
	"b": 1,
	"c": 2,
	"d": 3,
	"e": 4,
	"f": 5,
	"g": 6,
	"h": 7,
}

// NumberToLetter converts grid index values back to Tak x-values (letters)
var NumberToLetter = map[int]string{
	0: "a",
	1: "b",
	2: "c",
	3: "d",
	4: "e",
	5: "f",
	6: "g",
	7: "h",
}

// Placement descripts the necessary aspects to describe an action that places a new piece on the board
type Placement struct {
	Piece  Piece  `json:"piece"`
	Coords string `json:"coords"`
}

// Movement contains the necessary aspects to describe an action that moves a stack.
type Movement struct {
	Coords    string `json:"coords"`
	Direction string `json:"direction"`
	Carry     int    `json:"carry"`
	Drops     []int  `json:"drops"`
}

// WebError is a custom error type for reporting bad events when making an HTTP request
type WebError struct {
	Error   error
	Message string
	Code    int
}
