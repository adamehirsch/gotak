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

// Stack is just a slice of Pieces.
type Stack struct {
	// Note: the "top" of the stack is at [0]
	Pieces []Piece
}

// TakGame is the general object representing an entire game, including a board, an id, and some metadata.
type TakGame struct {
	GameID        uuid.UUID
	GameBoard     [][]Stack
	IsBlackTurn   bool
	IsBlackWinner bool
	IsWhiteWinner bool
	DrawGame      bool
	GameOver      bool
	GameWinner    uuid.UUID
	WinningPath   []Coords
	WinTime       time.Time
	BlackPlayer   uuid.UUID
	WhitePlayer   uuid.UUID
}

// PieceLimits is a map of gridsize to piece limits per player
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
