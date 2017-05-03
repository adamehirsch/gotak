package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"

	jwtmiddleware "github.com/auth0/go-jwt-middleware"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/gorilla/mux"

	"github.com/satori/go.uuid"
)

// simplify error reporting in web handlers by making our own type that handles WebError return values
type errorHandler func(http.ResponseWriter, *http.Request) *WebError

// ... make anything of errorHandler type satisy the http.Handler interface requirements
func (fn errorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil { // note that e is *webError, not os.Error.
		http.Error(w, e.Message, e.Code)
	}
}

// SlashHandler is a slim handler to present some canned text for humans to read
func SlashHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("GOTAK!\n"))
}

// NewGame will generate a new board with a specified size and return the UUID by which it will be known throughout its short, happy life.
func NewGame(w http.ResponseWriter, r *http.Request) *WebError {
	player, err := authUser(r)
	if err != nil {
		return &WebError{err, "problem authenticating user", http.StatusUnprocessableEntity}
	}

	// pull the boardSize out of the URL itself, e.g. /newgame/4, /newgame/6
	vars := mux.Vars(r)
	var boardSize int

	if boardSize, err = strconv.Atoi(vars["boardSize"]); err != nil {
		return &WebError{fmt.Errorf("could not understand requested board size: %v", vars["boardSize"]), fmt.Sprintf("could not understand requested board size: %v", vars["boardSize"]), http.StatusBadRequest}
	}

	newGame, err := MakeGame(boardSize)
	if err != nil {
		return &WebError{fmt.Errorf("could not create requested board size: %v", err), fmt.Sprintf("could not create requested board: %v", err), http.StatusInternalServerError}
	}

	// optional URL parameter to indicate the game's open to anyone. Future use, I suspect.
	isPublic, _ := regexp.MatchString("^(?i)true|yes$", r.FormValue("public"))

	newGame.GameOwner = player.Name
	newGame.IsPublic = isPublic
	// stash the new game in the db
	if err := StoreTakGame(newGame); err != nil {
		return &WebError{errors.New("problem storing new game"), "problem storing new game", http.StatusInternalServerError}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(newGame); err != nil {
		log.Println(err)
	}

	return nil
}

// ShowGame takes a given UUID, looks up the game (if it exists) and returns the current grid
func ShowGame(w http.ResponseWriter, r *http.Request) *WebError {
	player, err := authUser(r)
	if err != nil {
		return &WebError{err, "problem authenticating user", http.StatusUnprocessableEntity}
	}

	vars := mux.Vars(r)
	var (
		gameID        uuid.UUID
		requestedGame *TakGame
	)

	if gameID, err = uuid.FromString(vars["gameID"]); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "requested game ID '%v' not understood.", gameID)
	}
	if requestedGame, err = RetrieveTakGame(gameID); err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "requested game '%v' not found.", gameID)
	}

	if requestedGame.CanShow(player) {
		// optional URL parameter to just show the stack tops.
		showTops, _ := regexp.MatchString("^(?i)true|yes$", r.FormValue("showtops"))
		var gamePayload []byte

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if showTops {
			topView := requestedGame.DrawStackTops()
			gamePayload, _ = json.Marshal(topView)
		} else {
			gamePayload, _ = json.Marshal(requestedGame)
		}
		w.Write([]byte(gamePayload))
	} else {
		return &WebError{errors.New("Not allowed to display game"), "Not allowed to display game", http.StatusForbidden}
	}

	return nil
}

// Action will accept a JSON action for a particular game, determine whether it's a placement or movement, execute it if rules allow, and then return the updated grid.
func Action(w http.ResponseWriter, r *http.Request) *WebError {
	player, err := authUser(r)
	if err != nil {
		return &WebError{err, "problem authenticating user", http.StatusUnprocessableEntity}
	}

	// get the gameID from the URL path
	vars := mux.Vars(r)
	gameID, err := uuid.FromString(vars["gameID"])
	if err != nil {
		return &WebError{err, "Problem with game ID", http.StatusNotAcceptable}
	}

	// fetch out and validate that we've got a game by that ID
	requestedGame, err := RetrieveTakGame(gameID)
	if err != nil {
		return &WebError{err, "No such game found", http.StatusNotFound}
	}

	if !requestedGame.PlayersTurn(player) {
		return &WebError{errors.New("Not your turn"), "Not this players turn", http.StatusBadRequest}
	}

	// read in only up to 1MB of data from the client. Come on, now.
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		log.Println(err)
	}

	var (
		placement Placement
		movement  Movement
	)

	if vars["action"] == "place" {
		if unmarshalError := json.Unmarshal(body, &placement); unmarshalError != nil {
			return &WebError{unmarshalError, "Problem decoding JSON", http.StatusUnprocessableEntity}
		}

		// json.Unmarshal will sometimes parse valid but inapplicable JSON into an empty struct. Catch that.
		if ((placement.Piece) == Piece{} || (placement.Coords) == "") {
			return &WebError{errors.New("Missing piece and/or coordinates"), "Placement is missing piece and/or coordinates", http.StatusUnprocessableEntity}
		}

		if placementErr := requestedGame.PlacePiece(placement); placementErr != nil {
			return &WebError{err, fmt.Sprintf("problem placing piece at %v: %v", placement.Coords, placementErr), 409}
		}

	} else if vars["action"] == "move" {

		if unmarshalError := json.Unmarshal(body, &movement); unmarshalError != nil {
			return &WebError{unmarshalError, "Problem decoding JSON", http.StatusUnprocessableEntity}
		}

		// json.Unmarshal will sometimes parse valid but inapplicable JSON into an empty struct. Catch that.
		if (movement.Coords) == "" || (movement.Direction == "") {
			return &WebError{errors.New("Missing coordinates and/or direction"), "Missing coordinates and/or direction", http.StatusUnprocessableEntity}
		}

		if movementErr := requestedGame.MoveStack(movement); movementErr != nil {
			return &WebError{err, fmt.Sprintf("%v: %v", movementErr, movement.Coords), 409}
		}
	}
	// store the updated game back in the DB
	if err = StoreTakGame(requestedGame); err != nil {
		return &WebError{err, fmt.Sprintf("storage problem: %v", err), http.StatusInternalServerError}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	gamePayload, _ := json.Marshal(requestedGame)
	w.Write([]byte(gamePayload))
	return nil
}

// Login checks credentials before issuing a JWT auth token
func Login(w http.ResponseWriter, r *http.Request) *WebError {
	var (
		player PlayerCredentials
		name   string
		id     string
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
		return &WebError{errors.New("Missing username or password"), "Missing player username or password", http.StatusUnprocessableEntity}
	}

	// look up the details
	queryErr := db.QueryRow("SELECT guid, username, hash FROM players WHERE username = ?", player.UserName).Scan(&id, &name, &hash)

	switch {
	case queryErr == sql.ErrNoRows:
		return &WebError{fmt.Errorf("No player named '%v' found", player.UserName), fmt.Sprintf("No player named '%v' found", player.UserName), http.StatusUnprocessableEntity}
	case queryErr != nil:
		// problem with running the query? Yell.
		log.Fatal(queryErr)
	case !VerifyPassword(player.Password, hash):
		return &WebError{errors.New("Incorrect password"), "incorrect password", http.StatusBadRequest}
	}

	token := generateJWT(player, "successfully logged in")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(token)
	return nil
}

// Register handles new players
func Register(w http.ResponseWriter, r *http.Request) *WebError {
	var newPlayer PlayerCredentials

	// read in only up to 1MB of data from the client. Come on, now.
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		log.Println(err)
	}

	if unmarshalError := json.Unmarshal(body, &newPlayer); unmarshalError != nil {
		return &WebError{unmarshalError, "Problem decoding JSON", http.StatusUnprocessableEntity}
	}

	// json.Unmarshal will sometimes parse valid but inapplicable JSON into an empty struct. Catch that.
	if newPlayer.UserName == "" || newPlayer.Password == "" {
		return &WebError{errors.New("Missing new player username or password"), "Missing new player username or password", http.StatusUnprocessableEntity}
	}

	// check to see if the name conflicts in the DB
	var matchName string

	queryErr := db.QueryRow("SELECT username FROM players WHERE username = ?", newPlayer.UserName).Scan(&matchName)
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
	newPlayer.PlayerID = uuid.NewV4()
	newPlayerHash := HashPassword(newPlayer.Password)

	stmt, _ := db.Prepare("INSERT INTO players(guid, username, hash) VALUES(?, ?, ?)")
	_, err = stmt.Exec(newPlayer.PlayerID, newPlayer.UserName, newPlayerHash)
	if err != nil {
		log.Fatal(err)
	}

	tokenBytes := generateJWT(newPlayer, "new player successfully created")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(tokenBytes)
	return nil

}

// TakeSeat gives an open seat in a game to a requesting player
func TakeSeat(w http.ResponseWriter, r *http.Request) *WebError {
	player, err := authUser(r)
	if err != nil {
		return &WebError{err, "problem authenticating user", http.StatusUnprocessableEntity}
	}

	// get the gameID from the URL path
	vars := mux.Vars(r)
	gameID, err := uuid.FromString(vars["gameID"])
	if err != nil {
		return &WebError{err, "Problem with game ID", http.StatusNotAcceptable}
	}

	// fetch out and validate that we've got a game by that ID
	requestedGame, err := RetrieveTakGame(gameID)
	if err != nil {
		return &WebError{err, "No such game found", http.StatusNotFound}
	}

	switch {
	case requestedGame.WhitePlayer == player.Name || requestedGame.BlackPlayer == player.Name:
		return &WebError{errors.New("already seated at this game"), "already seated at this game", http.StatusConflict} // both seats are open
	case requestedGame.WhitePlayer == "" && requestedGame.BlackPlayer == "":
		// flip a coin to see which of the open seats you get.
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		if r.Intn(2) == 0 {
			requestedGame.BlackPlayer = player.Name
		} else {
			requestedGame.WhitePlayer = player.Name
		}
		// both seats are occupied
	case requestedGame.WhitePlayer != "" && requestedGame.BlackPlayer != "":
		return &WebError{errors.New("both seats already taken"), "both seats already taken", http.StatusConflict}
	case requestedGame.BlackPlayer == "":
		requestedGame.BlackPlayer = player.Name
	case requestedGame.WhitePlayer == "":
		requestedGame.WhitePlayer = player.Name
	}
	// store the updated game back in the DB
	if err = StoreTakGame(requestedGame); err != nil {
		return &WebError{err, fmt.Sprintf("storage problem: %v", err), http.StatusInternalServerError}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	gamePayload, _ := json.Marshal(requestedGame)
	w.Write([]byte(gamePayload))
	return nil
}

// checkJWTsignature will check a given token and verify that it was signed with the key and method specified below before passing access to its referenced Handler
var checkJWTsignature = jwtmiddleware.New(jwtmiddleware.Options{
	ValidationKeyGetter: jwtKeyFn,
	SigningMethod:       jwt.SigningMethodHS256,
	Debug:               false,
})

func jwtKeyFn(token *jwt.Token) (interface{}, error) {
	return jwtSigningKey, nil
}

func generateJWT(p PlayerCredentials, m string) []byte {
	// Okay, the person's authenticated. Give them a token.
	token := jwt.New(jwt.SigningMethodHS256)

	// /* Create a map to store our claims */
	claims := token.Claims.(jwt.MapClaims)

	claims["user"] = p.UserName
	claims["exp"] = time.Now().Add(time.Hour * 24 * time.Duration(loginDays)).Unix()

	// sign the token
	tokenString, _ := token.SignedString(jwtSigningKey)
	thisJWT := TakJWT{
		JWT:     tokenString,
		Message: m,
	}
	JWTjson, _ := json.Marshal(thisJWT)
	return []byte(JWTjson)
}

// HashPassword uses bcrypt to produce a password hash suitable for storage
func HashPassword(pw string) []byte {
	password := []byte(pw)
	// Hashing the password with the default cost should be ample
	hashedPassword, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	return hashedPassword
}

// VerifyPassword will verify ... wait for it ... that a password matches a hash
func VerifyPassword(pw string, hpw string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hpw), []byte(pw)); err != nil {
		return false
	}
	return true

}

// authUser parses the username out of the JWT token and returns it to whoever's asking
func authUser(r *http.Request) (player *TakPlayer, err error) {
	token, _ := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor, jwtKeyFn)
	claims := token.Claims.(jwt.MapClaims)
	username, ok := claims["user"].(string)
	if !ok {
		return nil, fmt.Errorf("no such player found: %v", username)
	}
	if player, err = RetrievePlayerByName(username); err != nil {
		return nil, err
	}
	return player, nil
}

// CanShow determines whether a given game can be shown to a given player
func (tg *TakGame) CanShow(p *TakPlayer) bool {
	switch {
	case tg.IsPublic:
		return true
	case tg.BlackPlayer == p.Name || tg.WhitePlayer == p.Name || tg.GameOwner == p.Name:
		return true
	default:
		return false
	}
}

// PlayersTurn determines whether a given player can make the next move
func (tg *TakGame) PlayersTurn(p *TakPlayer) bool {
	switch {
	case tg.BlackPlayer == p.Name && tg.IsBlackTurn == true:
		return true
	case tg.WhitePlayer == p.Name && tg.IsBlackTurn == false:
		return true
	default:
		return false
	}
}
