package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"

	authboss "gopkg.in/authboss.v1"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

var (
	ab            = authboss.New()
	sslKey        string
	sslCert       string
	jwtSigningKey []byte
	loginDays     int
)

func init() {

	// read in the configuration file
	viper.SetConfigName("conf")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("can't read configuration file: %v", err))
	}

	sslKey = viper.GetString("production.sslKey")
	sslCert = viper.GetString("production.sslCert")
	loginDays = viper.GetInt("production.loginDays")

	jwtSigningKey = []byte(viper.GetString("production.jwtSigningKey"))

	if _, err := os.Stat(sslKey); os.IsNotExist(err) {
		panic(fmt.Sprintf("can't read SSL key %v: %v", sslKey, err))
	}

	if _, err := os.Stat(sslCert); os.IsNotExist(err) {
		panic(fmt.Sprintf("can't read SSL cert %v: %v", sslCert, err))
	}

	// ensure the database is setup
	InitDB(viper.GetString("production.dbname"))

}

// gorilla mux requires some explicit steps to get pprof to attach to it
func attachProfiler(router *mux.Router) {
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

	// Manually add support for paths linked to by index page at /debug/pprof/
	router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	router.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	router.Handle("/debug/pprof/block", pprof.Handler("block"))
}

func main() {
	var debug = flag.Bool("debug", false, "debug mode")
	flag.Parse()
	defer db.Close()

	r := mux.NewRouter()

	if *debug {
		attachProfiler(r)
	}
	checkedChain := alice.New(checkJWTsignature.Handler)

	r.HandleFunc("/", SlashHandler)
	r.Handle("/login", errorHandler(Login)).Methods("POST")
	r.Handle("/register", errorHandler(Register)).Methods("POST")

	r.Handle("/newgame/{boardSize}", checkedChain.Then(errorHandler(NewGame)))
	r.Handle("/showgame/{gameID}", checkedChain.Then(errorHandler(ShowGame)))
	//
	r.Handle("/action/{action}/{gameID}", checkedChain.Then(errorHandler(Action))).Methods("PUT")
	r.Handle("/showtops/{gameID}", checkedChain.Then(errorHandler(ShowStackStops))).Methods("GET")

	// Bind to a port and pass our router in, logging every request to Stdout
	log.Println(http.ListenAndServeTLS(":8000", sslCert, sslKey, handlers.LoggingHandler(os.Stdout, r)))

}
