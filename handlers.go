package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	jwt "github.com/dgrijalva/jwt-go"

	"github.com/gorilla/mux"
	"github.com/satori/go.uuid"
)

// Let's try simplifying error reporting back to the user by making our own Handler that produces WebError
type webHandler func(http.ResponseWriter, *http.Request) *WebError

func (fn webHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil { // e is *webError, not os.Error.
		http.Error(w, e.Message, e.Code)
	}
}

// SlashHandler is a slim handler to present some canned HTML for humans to read
func SlashHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("GOTAK!\n"))
}

// NewGameHandler will generate a new board with a specified size and return the UUID by which it will be known throughout its short, happy life.
var NewGameHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	if boardsize, err := strconv.Atoi(vars["boardSize"]); err == nil {
		newGame, err := MakeGame(boardsize)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Could not create requested board: %v\n", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(newGame); err != nil {
			log.Println(err)
		}
	} else {
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprintf(w, "Could not understand requested board size: %v\n", vars["boardSize"])
	}

})

// ShowGameHandler takes a given UUID, looks up the game (if it exists) and returns the current grid
var ShowGameHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if gameID, err := uuid.FromString(vars["gameID"]); err == nil {

		if requestedGame, ok := gameIndex[gameID]; ok == true {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(requestedGame); err != nil {
				log.Println(err)
			}

		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "requested game '%v' not found.", gameID)
		}

	} else {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "requested game ID '%v' not understood.", gameID)

	}
})

/*
ActionHandler will accept a JSON action for a particular game, determine whether it's a placement or movement, execute it if rules allow, and then return the updated grid.
*/
func ActionHandler(w http.ResponseWriter, r *http.Request) *WebError {
	// get the gameID from the URL path
	vars := mux.Vars(r)
	gameID, err := uuid.FromString(vars["gameID"])
	if err != nil {
		return &WebError{err, "Problem with game ID", http.StatusNotAcceptable}
	}

	// fetch out and validate that we've got a game by that ID
	requestedGame, ok := gameIndex[gameID]
	if ok != true {
		return &WebError{err, "No such game found", http.StatusNotFound}
	}

	// read in only up to 1MB of data from the client. Come on, now.
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		log.Println(err)
	}

	// I am assuming there is a cleaner way to do this.
	var placement Placement
	var movement Movement

	if vars["action"] == "place" {
		if unmarshalError := json.Unmarshal(body, &placement); unmarshalError != nil {
			return &WebError{unmarshalError, "Problem decoding JSON", http.StatusUnprocessableEntity}
		}

		// json.Unmarshal will sometimes parse valid but inapplicable JSON into an empty struct. Catch that.
		if ((placement.Piece) == Piece{} || (placement.Coords) == "") {
			return &WebError{errors.New("Missing piece and/or coordinates"), "Placement is missing piece and/or coordinates", http.StatusUnprocessableEntity}
		}

		if err := requestedGame.PlacePiece(placement); err != nil {
			return &WebError{err, fmt.Sprintf("problem placing piece at %v: %v", placement.Coords, err), 409}
		}

	} else if vars["action"] == "move" {

		if unmarshalError := json.Unmarshal(body, &movement); unmarshalError != nil {
			return &WebError{unmarshalError, "Problem decoding JSON", http.StatusUnprocessableEntity}
		}

		// json.Unmarshal will sometimes parse valid but inapplicable JSON into an empty struct. Catch that.
		if (movement.Coords) == "" || (movement.Direction == "") {
			return &WebError{errors.New("Missing coordinates and/or direction"), "Missing coordinates and/or direction", http.StatusUnprocessableEntity}
		}

		if err := requestedGame.MoveStack(movement); err != nil {
			return &WebError{err, fmt.Sprintf("%v: %v", err, movement.Coords), 409}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	gamePayload, _ := json.Marshal(requestedGame)
	// json.NewEncoder(w).Encode(requestedGame)
	w.Write([]byte(gamePayload))
	return nil
}

// LoginHandler will eventually check credentials before issuing a JWT auth token
var LoginHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	/* Create the token */
	token := jwt.New(jwt.SigningMethodHS256)

	// /* Create a map to store our claims */
	claims := token.Claims.(jwt.MapClaims)

	/* Set token claims  - hardcoded for right now*/
	claims["name"] = "Ado Kukic"
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	/* Sign the token with our secret */
	tokenString, _ := token.SignedString([]byte(jwtSigningKey))

	/* Finally, write the token to the browser window */
	w.Write([]byte(tokenString))
})

var jwtMiddleware = jwtmiddleware.New(jwtmiddleware.Options{
	ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSigningKey), nil
	},
	SigningMethod: jwt.SigningMethodHS256,
})
