package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/crypto/bcrypt"

	"database/sql"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
	"gopkg.in/authboss.v1"
)

var (
	ab            = authboss.New()
	sslKey        string
	sslCert       string
	jwtSigningKey []byte
)

func init() {
	// read in the configuration file, set up the database
	viper.SetConfigName("conf")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("can't read configuration file: %v", err))
	}

	sslKey = viper.GetString("production.sslKey")
	sslCert = viper.GetString("production.sslCert")
	jwtSigningKey = []byte(viper.GetString("production.jwtSigningKey"))

	if _, err := os.Stat(sslKey); os.IsNotExist(err) {
		panic(fmt.Sprintf("can't read SSL key %v: %v", sslKey, err))
	}

	if _, err := os.Stat(sslCert); os.IsNotExist(err) {
		panic(fmt.Sprintf("can't read SSL cert %v: %v", sslCert, err))
	}

	database, _ := sql.Open("sqlite3", "./gotak.db")
	statement, _ := database.Prepare("CREATE TABLE IF NOT EXISTS users (guid CHARACTER(37) PRIMARY KEY, username VARCHAR UNIQUE NOT NULL, hash VARCHAR)")
	statement.Exec()

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

// VerifyPassword will verify ... wait for it ... a password matches a hash
func VerifyPassword(pw []byte, hpw []byte) bool {
	if err := bcrypt.CompareHashAndPassword(hpw, pw); err != nil {
		return false
	}
	return true

}

func main() {

	r := mux.NewRouter()

	r.HandleFunc("/", SlashHandler)
	r.Handle("/login", LoginHandler).Methods("GET")
	r.Handle("/register", webHandler(RegisterHandler)).Methods("POST")

	r.Handle("/newgame/{boardSize}", jwtMiddleware.Handler(NewGameHandler))
	r.Handle("/showgame/{gameID}", jwtMiddleware.Handler(ShowGameHandler))
	r.Handle("/action/{action}/{gameID}", jwtMiddleware.Handler(webHandler(ActionHandler))).Methods("PUT")

	// Setup to serve static assest like images, css from the /static/{file} route
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	// Bind to a port and pass our router in, logging every request to Stdout
	log.Fatal(http.ListenAndServeTLS(":8000", sslCert, sslKey, handlers.LoggingHandler(os.Stdout, r)))
}
