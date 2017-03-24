package main

import (
	uuid "github.com/satori/go.uuid"
)

// Various constants for use throughout the game
const (
	Flat     = "flat"
	Standing = "standing"
	Capstone = "capstone"
	Black    = "black"
	White    = "white"
)

// Piece is the most basic element of a Tak game. One of two colors; one of three types.
type Piece struct {
	// one of "black" or "white"
	Color string `json:"color"`
	// Type can be one of "flat", "standing", or "capstone"
	Orientation string `json:"orientation"`
}

// Stack is potentially a needless piece of structure; it's just a slice of Pieces... so maybe I could/should just refer to []Piece instead of having a distinct struct for it
type Stack struct {
	// the top of the stack is at [0]
	Pieces []Piece
}

// TakGame is the general object representing an entire game, including a board, an id, and some metadata. A given TakGame has a guaranteed unique uuid
type TakGame struct {
	GameID        uuid.UUID
	GameBoard     [][]Stack
	IsBlackTurn   bool
	IsBlackWinner bool
	GameOver      bool
	GameWinner    uuid.UUID
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

// LetterMap converts Tak files to their index value
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
