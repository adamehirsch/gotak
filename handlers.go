package main

import (
	"database/sql"
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
	"github.com/dgrijalva/jwt-go/request"
	"github.com/gorilla/mux"

	"github.com/satori/go.uuid"
)

// // Let's try simplifying error reporting back to the user by making our own middleware Handler that produces WebError
type errorHandler func(http.ResponseWriter, *http.Request) *WebError

// ... and make the errorHandler type satisy the http.Handler interface requirements
func (fn errorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil { // note that e is *webError, not os.Error.
		http.Error(w, e.Message, e.Code)
	}
}

// func errorHandler(next http.Handler) http.Handler {
//   return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//     next.ServeHTTP(w, r)
//   })
// }

// SlashHandler is a slim handler to present some canned HTML for humans to read
func SlashHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("GOTAK!\n"))
}

// NewGame will generate a new board with a specified size and return the UUID by which it will be known throughout its short, happy life.
func NewGame(w http.ResponseWriter, r *http.Request) *WebError {
	vars := mux.Vars(r)

	if boardsize, err := strconv.Atoi(vars["boardSize"]); err == nil {
		newGame, err := MakeGame(boardsize)
		if err != nil {
			return &WebError{fmt.Errorf("could not create requested board size: %v", err), fmt.Sprintf("could not create requested board: %v", err), http.StatusInternalServerError}

		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(newGame); err != nil {
			log.Println(err)
		}
	} else {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return &WebError{fmt.Errorf("could not understand requested board size: %v", vars["boardSize"]), fmt.Sprintf("could not understand requested board size: %v", vars["boardSize"]), http.StatusBadRequest}
	}
	return nil
}

// ShowGame takes a given UUID, looks up the game (if it exists) and returns the current grid
func ShowGame(w http.ResponseWriter, r *http.Request) *WebError {
	token, _ := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor, jwtKeyFn)
	claims := token.Claims.(jwt.MapClaims)
	for k, v := range claims {
		fmt.Printf("claim %s :\t%#v\n", k, v)
	}

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
	return nil
}

/*
Action will accept a JSON action for a particular game, determine whether it's a placement or movement, execute it if rules allow, and then return the updated grid.
*/
func Action(w http.ResponseWriter, r *http.Request) *WebError {
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

// Login checks credentials before issuing a JWT auth token
func Login(w http.ResponseWriter, r *http.Request) *WebError {
	var (
		player PlayerCredentials
		name   string
		hash   string
	)
	// read in only up to 1MB of data from the client. Come on, now.
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		log.Println(err)
	}

	if unmarshalError := json.Unmarshal(body, &player); unmarshalError != nil {
		return &WebError{unmarshalError, "Problem decoding JSON", http.StatusUnprocessableEntity}
	}

	// json.Unmarshal will parse valid but inapplicable JSON into an empty struct. Catch that.
	if player.UserName == "" || player.Password == "" {
		return &WebError{errors.New("Missing new player username or password"), "Missing new player username or password", http.StatusUnprocessableEntity}
	}

	queryErr := db.QueryRow("SELECT username, hash FROM users WHERE username = ?", player.UserName).Scan(&name, &hash)

	switch {
	case queryErr == sql.ErrNoRows:
		return &WebError{fmt.Errorf("No player named '%v' found", player.UserName), fmt.Sprintf("No player named '%v' found", player.UserName), http.StatusUnprocessableEntity}
	case queryErr != nil:
		// problem with running the query? Yell.
		log.Fatal(queryErr)
	case !VerifyPassword(player.Password, hash):
		return &WebError{errors.New("Incorrect password"), "incorrect password", http.StatusBadRequest}
	}

	// Okay, the person's authenticated. Give them a token.
	token := jwt.New(jwt.SigningMethodHS256)

	// /* Create a map to store our claims */
	claims := token.Claims.(jwt.MapClaims)

	// TODO set user to logged in user
	claims["user"] = player.UserName
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	/* Sign the token with our secret */
	tokenString, _ := token.SignedString(jwtSigningKey)

	/* Finally, write the token to the browser window */
	w.Write([]byte(tokenString))
	return nil
}

// Register registers new players
func Register(w http.ResponseWriter, r *http.Request) *WebError {
	var player PlayerCredentials

	// read in only up to 1MB of data from the client. Come on, now.
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		log.Println(err)
	}

	if unmarshalError := json.Unmarshal(body, &player); unmarshalError != nil {
		return &WebError{unmarshalError, "Problem decoding JSON", http.StatusUnprocessableEntity}
	}

	// json.Unmarshal will sometimes parse valid but inapplicable JSON into an empty struct. Catch that.
	if player.UserName == "" || player.Password == "" {
		return &WebError{errors.New("Missing new player username or password"), "Missing new player username or password", http.StatusUnprocessableEntity}
	}

	// check to see if the name conflicts in the DB
	var matchName string

	queryErr := db.QueryRow("SELECT username FROM users WHERE username = ?", player.UserName).Scan(&matchName)
	switch {
	case queryErr == sql.ErrNoRows:
		// that's what we want to see: no rows.
		break
	case queryErr != nil:
		// problem with running the query? Yell.
		log.Fatal(err)
	case matchName != "":
		return &WebError{fmt.Errorf("new player username %v conflicts with existing username", matchName), fmt.Sprintf("new player username '%v' conflicts with existing username", matchName), http.StatusUnprocessableEntity}
	}

	// every player gets a unique uuid
	newPlayerID := uuid.NewV4()
	newPlayerHash := HashPassword(player.Password)

	stmt, _ := db.Prepare("INSERT INTO users(guid, username, hash) VALUES(?, ?, ?)")
	_, err = stmt.Exec(newPlayerID.String(), player.UserName, newPlayerHash)
	if err != nil {
		log.Fatal(err)
	}

	newTakPlayer := TakPlayer{
		Name:         player.UserName,
		PasswordHash: newPlayerHash,
		PlayerID:     newPlayerID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	playerRegistration, _ := json.Marshal(newTakPlayer)
	w.Write([]byte(playerRegistration))

	return nil
}

// checkJWTsignature will check a given token and verify that it was signed with the key and method specified below before passing access to its referenced Handler
var checkJWTsignature = jwtmiddleware.New(jwtmiddleware.Options{
	ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
		return jwtSigningKey, nil
	},
	SigningMethod: jwt.SigningMethodHS256,
	Debug:         false,
})

func jwtKeyFn(token *jwt.Token) (interface{}, error) {
	return jwtSigningKey, nil
}

// func checkExpireTime(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		if token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor, jwtKeyFn); err == nil {
// 			claims := token.Claims.(jwt.MapClaims)
// 			expiryUnixTime := int64(claims["exp"].(float64))
// 			nowTime := time.Now().Unix()
// 			switch {
// 			case nowTime >= expiryUnixTime:
// 				fmt.Printf("(%v) Token expired at %v\n", time.Now().Format(time.UnixDate), expiryUnixTime)
// 			case nowTime < expiryUnixTime:
// 				fmt.Printf("(%v) Token expires at %v\n", time.Now().Format(time.UnixDate), expiryUnixTime)
// 			}
// 			// fmt.Printf("at %v: Token for user %v expires %v (%v)\n", nowTime, claims["user"], expiryUnixTime)
// 		}
// 		next.ServeHTTP(w, r)
//
// 	})
// }
