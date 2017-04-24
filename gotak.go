package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("conf")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic("can't read configuration file")
	}
	// jwtSigningKey := viper.GetString("production.jwtSigningKey")
	sslKey := viper.GetString("production.sslKey")
	sslCert := viper.GetString("production.sslCert")
	if _, err := os.Stat(sslKey); os.IsNotExist(err) {
		panic(fmt.Sprintf("can't read SSL key %v", sslKey))
	}

	if _, err := os.Stat(sslCert); os.IsNotExist(err) {
		panic(fmt.Sprintf("can't read SSL cert %v", sslCert))
	}

	r := mux.NewRouter()
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", SlashHandler)
	r.HandleFunc("/newgame/{boardSize}", NewGameHandler)
	r.HandleFunc("/showgame/{gameID}", ShowGameHandler)
	r.Handle("/action/{action}/{gameID}", webHandler(ActionHandler)).Methods("PUT")

	// Setup to serve static assest like images, css from the /static/{file} route
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	// Bind to a port and pass our router in, logging every request to Stdout
	log.Fatal(http.ListenAndServeTLS(":8000", sslCert, sslKey, handlers.LoggingHandler(os.Stdout, r)))
}
