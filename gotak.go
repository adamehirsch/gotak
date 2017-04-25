package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"gopkg.in/authboss.v1"
)

var (
	ab            = authboss.New()
	sslKey        string
	sslCert       string
	jwtSigningKey string
)

func init() {
	viper.SetConfigName("conf")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("can't read configuration file: %v", err))
	}

	sslKey = viper.GetString("production.sslKey")
	sslCert = viper.GetString("production.sslCert")
	jwtSigningKey = viper.GetString("production.jwtSigningKey")

	if _, err := os.Stat(sslKey); os.IsNotExist(err) {
		panic(fmt.Sprintf("can't read SSL key %v: %v", sslKey, err))
	}

	if _, err := os.Stat(sslCert); os.IsNotExist(err) {
		panic(fmt.Sprintf("can't read SSL cert %v: %v", sslCert, err))
	}

}

func main() {

	r := mux.NewRouter()

	r.HandleFunc("/", SlashHandler)
	r.Handle("/login", LoginHandler).Methods("GET")

	r.Handle("/newgame/{boardSize}", jwtMiddleware.Handler(NewGameHandler))
	r.Handle("/showgame/{gameID}", jwtMiddleware.Handler(ShowGameHandler))
	r.Handle("/action/{action}/{gameID}", jwtMiddleware.Handler(webHandler(ActionHandler))).Methods("PUT")

	// Setup to serve static assest like images, css from the /static/{file} route
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	// Bind to a port and pass our router in, logging every request to Stdout
	log.Fatal(http.ListenAndServeTLS(":8000", sslCert, sslKey, handlers.LoggingHandler(os.Stdout, r)))
}
