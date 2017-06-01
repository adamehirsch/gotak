package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/gorilla/mux"

	"github.com/satori/go.uuid"
)

// WebError is a custom error type for reporting bad events when making an HTTP request
// swagger:model
type WebError struct {
	Error   error
	Message string
	Code    int
}

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
func (env *DBenv) NewGame(w http.ResponseWriter, r *http.Request) *WebError {
	player, err := env.authUser(r)
	if err != nil {
		return &WebError{err, fmt.Sprintf("problem authenticating user: %v", err), http.StatusUnprocessableEntity}
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

	newGame.GameOwner = player.Username
	newGame.IsPublic = isPublic
	// stash the new game in the db
	if err := env.db.StoreTakGame(newGame); err != nil {
		return &WebError{errors.New("problem storing new game"), "problem storing new game", http.StatusInternalServerError}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	gamePayload, _ := json.Marshal(newGame)
	w.Write([]byte(gamePayload))

	return nil
}

// ShowGame takes a given UUID, looks up the game (if it exists) and returns the current grid
func (env *DBenv) ShowGame(w http.ResponseWriter, r *http.Request) *WebError {
	player, err := env.authUser(r)
	if err != nil {
		return &WebError{err, fmt.Sprintf("problem authenticating user: %v", err), http.StatusUnprocessableEntity}
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
	if requestedGame, err = env.db.RetrieveTakGame(gameID); err != nil {
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
			fmt.Printf("before: %+v\n\n", requestedGame)
			gamePayload, _ = json.Marshal(requestedGame)
			fmt.Printf("after: %+v\n\n", requestedGame)
		}
		w.Write([]byte(gamePayload))
	} else {
		return &WebError{errors.New("Not allowed to display game"), "Not allowed to display game", http.StatusForbidden}
	}

	return nil
}

// Action will accept a JSON action for a particular game, determine whether it's a placement or movement, execute it if rules allow, and then return the updated grid.
func (env *DBenv) Action(w http.ResponseWriter, r *http.Request) *WebError {
	player, err := env.authUser(r)
	if err != nil {
		return &WebError{err, fmt.Sprintf("problem authenticating user: %v", err), http.StatusUnprocessableEntity}
	}

	// get the gameID from the URL path
	vars := mux.Vars(r)
	gameID, err := uuid.FromString(vars["gameID"])
	if err != nil {
		return &WebError{err, fmt.Sprintf("Problem with game ID: %v", err), http.StatusNotAcceptable}
	}

	// fetch out and validate that we've got a game by that ID
	requestedGame, err := env.db.RetrieveTakGame(gameID)
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

		// json.Unmarshal will parse valid but inapplicable JSON into an empty struct. Catch that.
		if ((placement.Piece) == Piece{} || (placement.Coords) == "") {
			return &WebError{errors.New("Missing piece and/or coordinates"), "Placement is missing piece and/or coordinates", http.StatusUnprocessableEntity}
		}

		// Place that Piece!
		if placementErr := requestedGame.PlacePiece(placement); placementErr != nil {
			return &WebError{err, fmt.Sprintf("problem placing piece at %v: %v", placement.Coords, placementErr), 409}
		}

		if requestedGame.StartTime.IsZero() {
			requestedGame.StartTime = time.Now()
		}

	} else if vars["action"] == "move" {

		if unmarshalError := json.Unmarshal(body, &movement); unmarshalError != nil {
			return &WebError{unmarshalError, "Problem decoding JSON", http.StatusUnprocessableEntity}
		}

		// json.Unmarshal will parse valid but inapplicable JSON into an empty struct. Catch that.
		if (movement.Coords) == "" || (movement.Direction == "") {
			return &WebError{errors.New("Missing coordinates and/or direction"), "Missing coordinates and/or direction", http.StatusUnprocessableEntity}
		}

		// Move that Stack!
		if movementErr := requestedGame.MoveStack(movement); movementErr != nil {
			return &WebError{err, fmt.Sprintf("%v: %v", movementErr, movement.Coords), 409}
		}
		// set StartTime, if it hasn't been set
		if requestedGame.StartTime.IsZero() {
			requestedGame.StartTime = time.Now()
		}
	}

	// store the updated game back in the DB
	if err = env.db.StoreTakGame(requestedGame); err != nil {
		return &WebError{err, fmt.Sprintf("storage problem: %v", err), http.StatusInternalServerError}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	gamePayload, _ := json.Marshal(requestedGame)
	w.Write([]byte(gamePayload))
	return nil
}

// Login checks credentials before issuing a JWT auth token
func (env *DBenv) Login(w http.ResponseWriter, r *http.Request) *WebError {
	var (
		player TakPlayer
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
	if player.Username == "" || player.Password == "" {
		log.WithFields(log.Fields{"username": player.Username, "password": player.Password}).Debug("login problem")
		return &WebError{errors.New("Missing username or password"), "Missing player username or password", http.StatusUnprocessableEntity}
	}

	if !env.db.PlayerExists(player.Username) {
		return &WebError{fmt.Errorf("No player named '%v' found", player.Username), fmt.Sprintf("No player named '%v' found", player.Username), http.StatusUnprocessableEntity}
	}

	dbPlayer, _ := env.db.RetrievePlayer(player.Username)

	if !VerifyPassword(player.Password, string(dbPlayer.passwordHash)) {
		return &WebError{errors.New("Incorrect password"), "incorrect password", http.StatusBadRequest}
	}

	token := generateJWT(&player, "successfully logged in")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(token)
	return nil
}

// Register handles new players
func (env *DBenv) Register(w http.ResponseWriter, r *http.Request) *WebError {
	// swagger:route POST /register Register
	//
	// registers a new user
	//
	//     Consumes:
	//     - application/json
	//     Produces:
	//     - application/json
	//     Responses:
	//       200: TakJWT
	//       422: "WebError"
	//       500: WebError
	var newPlayer TakPlayer

	// read in only up to 1MB of data from the client. Come on, now.
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		log.Println(err)
	}

	if unmarshalError := json.Unmarshal(body, &newPlayer); unmarshalError != nil {
		return &WebError{unmarshalError, "Problem decoding JSON", http.StatusUnprocessableEntity}
	}

	// json.Unmarshal will parse valid but inapplicable JSON into an empty struct. Catch that.
	if newPlayer.Username == "" || newPlayer.Password == "" {
		log.WithFields(log.Fields{"username": newPlayer.Username, "password": newPlayer.Password}).Debug("register problem")
		return &WebError{errors.New("Missing new player username or password"), "Missing new player username or password", http.StatusUnprocessableEntity}
	}

	if env.db.PlayerExists(newPlayer.Username) {
		return &WebError{fmt.Errorf("new player username %v conflicts with existing username", newPlayer.Username), fmt.Sprintf("new player username '%v' conflicts with existing username", newPlayer.Username), http.StatusUnprocessableEntity}
	}

	// every player gets a unique uuid
	newPlayer.PlayerID = uuid.NewV4()
	newPlayer.passwordHash = HashPassword(newPlayer.Password)

	if err := env.db.StorePlayer(&newPlayer); err != nil {
		return &WebError{err, fmt.Sprintf("storage problem: %v", err), http.StatusInternalServerError}
	}

	tokenBytes := generateJWT(&newPlayer, fmt.Sprintf("new player %v successfully created", newPlayer.Username))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(tokenBytes)
	return nil

}

// TakeSeat gives an open seat in a game to a requesting player
func (env *DBenv) TakeSeat(w http.ResponseWriter, r *http.Request) *WebError {
	// swagger:route GET /takeseat/{gameID} TakeSeat
	// for an authenticated user, allocates them an open seat on a specified game
	//
	//     Consumes:
	//
	//     Produces:
	//     - application/json
	//     Responses:
	//       200: TakGame
	//       404: WebError
	// 			 406: WebError
	//       422: WebError
	//       500: WebError

	player, err := env.authUser(r)
	if err != nil {
		return &WebError{err, fmt.Sprintf("problem authenticating user: %v", err), http.StatusUnprocessableEntity}
	}

	// get the gameID from the URL path
	vars := mux.Vars(r)
	gameID, err := uuid.FromString(vars["gameID"])
	if err != nil {
		return &WebError{err, "Problem with game ID", http.StatusNotAcceptable}
	}

	// fetch out and validate that we've got a game by that ID
	requestedGame, err := env.db.RetrieveTakGame(gameID)
	if err != nil {
		return &WebError{err, "No such game found", http.StatusNotFound}
	}

	switch {
	case requestedGame.WhitePlayer == player.Username || requestedGame.BlackPlayer == player.Username:
		return &WebError{errors.New("already seated at this game"), "already seated at this game", http.StatusConflict} // both seats are open
	case requestedGame.WhitePlayer == "" && requestedGame.BlackPlayer == "":
		// flip a coin to see which of the open seats you get.
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		if r.Intn(2) == 0 {
			requestedGame.BlackPlayer = player.Username
		} else {
			requestedGame.WhitePlayer = player.Username
		}
	case requestedGame.WhitePlayer != "" && requestedGame.BlackPlayer != "":
		// both seats are occupied
		return &WebError{errors.New("both seats already taken"), "both seats already taken", http.StatusConflict}
	case requestedGame.BlackPlayer == "":
		requestedGame.BlackPlayer = player.Username
	case requestedGame.WhitePlayer == "":
		requestedGame.WhitePlayer = player.Username
	}
	// store the updated game back in the DB
	if err = env.db.StoreTakGame(requestedGame); err != nil {
		return &WebError{err, fmt.Sprintf("storage problem: %v", err), http.StatusInternalServerError}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	gamePayload, _ := json.Marshal(requestedGame)
	w.Write([]byte(gamePayload))
	return nil
}

// GameIDParam is a uuid key specifying one TakGame
// swagger:parameters TakeSeat ShowGame Action
type GameIDParam struct {
	// gameID is useful
	// in: path
	GameID string `json:"gameID"`
}

// MarshalJSON is here to allow saner presentation via JSON methods
// func (tg *TakGame) MarshalJSON() ([]byte, error) {
// 	// copy the game so as to avoid recursively calling MarshalJSON
// 	// thanks https://ashleyd.ws/custom-json-marshalling-in-golang/
// 	type GameAlias TakGame
//
// 	return json.Marshal(&struct {
// 		GameBoard GameBoard `json:"gameBoard"`
// 		*GameAlias
// 	}{
// 		GameBoard: MakeGameBoardCartesian(&tg.GameBoard),
// 		GameAlias: (*GameAlias)(tg),
// 	})
// }

// MakeGameBoardCartesian should produce a gameboard rotated 90 degrees for more intuitive json marshaling
func MakeGameBoardCartesian(gb *GameBoard) GameBoard {
	boardSize := len(*gb)
	rotatedBoard := make(GameBoard, boardSize)

	for i := range rotatedBoard {
		rotatedBoard[i] = make([]Stack, boardSize)
	}
	for y := 0; y < boardSize; y++ {
		for x := 0; x < boardSize; x++ {
			rotatedBoard[y][(boardSize-x)-1] = (*gb)[x][y]
		}
	}
	return rotatedBoard
}
