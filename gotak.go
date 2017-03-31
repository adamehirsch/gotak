package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {

	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", SlashHandler)
	r.HandleFunc("/newgame/{boardSize}", NewGameHandler)
	r.HandleFunc("/showgame/{gameID}", ShowGameHandler)
	r.Handle("/action/{action}/{gameID}", webHandler(ActionHandler)).Methods("PUT")

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}
