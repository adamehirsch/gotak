// Package classification GoTak server
//
// meant to provide a running service to adjudicate games of Tak.
//     Schemes: https
//     Host: localhost
//     BasePath: /
//     Version: 0.0.1
//     License: MIT http://opensource.org/licenses/MIT
//     Contact: Adam Hirsch <gotak@quakerporn.com>
// swagger:meta
package main

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jessevdk/go-flags"
	"github.com/justinas/alice"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

var (
	sslKey        string
	sslCert       string
	jwtSigningKey string
	loginDays     int
	dbFile        string
)

// commandline options
var opts struct {
	Debug     bool   `short:"d" long:"debug" description:"Show verbose debug information"`
	SSLkey    string `long:"sslkey" description:"SSL key file"`
	SSLcert   string `long:"sslcert" description:"SSL cert file"`
	DBfile    string `long:"dbfile" description:"sqlite database storage file"`
	JWTkey    string `long:"jwtkey" description:"encryption key for JWT authentication tokens"`
	LoginDays int    `long:"logindays" description:"duration of time a JWT token is valid"`
}

//go:generate swagger generate spec
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
	jwtSigningKey = viper.GetString("production.jwtSigningKey")
	dbFile = viper.GetString("production.dbname")

	// ... flags, however, overrule the config file. Replace any unset flag values with values from the config file.
	flags.Parse(&opts)

	if opts.SSLkey == "" {
		opts.SSLkey = sslKey
	}

	if opts.SSLcert == "" {
		opts.SSLcert = sslCert
	}

	if opts.LoginDays == 0 {
		opts.LoginDays = loginDays
	}

	if opts.JWTkey == "" {
		opts.JWTkey = jwtSigningKey
	}

	if opts.DBfile == "" {
		opts.DBfile = dbFile
	}

	if _, err := os.Stat(opts.SSLkey); os.IsNotExist(err) {
		panic(fmt.Sprintf("can't read SSL key %v: %v", opts.SSLkey, err))
	}

	if _, err := os.Stat(opts.SSLcert); os.IsNotExist(err) {
		panic(fmt.Sprintf("can't read SSL cert %v: %v", opts.SSLcert, err))
	}

	// Output to stdout instead of the default stderr
	// Can be any io.Writer
	log.SetOutput(os.Stdout)

	if opts.Debug {
		log.SetLevel(log.DebugLevel)
		fmt.Printf("Provided options: %+v", opts)
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	} else {
		log.SetLevel(log.WarnLevel)
	}

}

func main() {

	// ensure the database is setup
	db, err := InitSQLiteDB(opts.DBfile)
	if err != nil {
		log.Panicf("problem initializing db connection: %v", err)
	}
	defer db.Close()

	// set up the live database behind a Datastore interface for our methods to run against
	env := &DBenv{db}
	// Bind to a port and pass our router in, logging every request to Stdout
	http.ListenAndServeTLS(":8000", sslCert, sslKey, handlers.LoggingHandler(os.Stdout, genRouter(env)))

}

func genRouter(env *DBenv) *mux.Router {
	r := mux.NewRouter()
	checkedChain := alice.New(checkJWTsignature.Handler)
	r.HandleFunc("/", SlashHandler)
	r.Handle("/login", errorHandler(env.Login)).Methods("POST")
	r.Handle("/register", errorHandler(env.Register)).Methods("POST")
	r.Handle("/newgame/{boardSize}", checkedChain.Then(errorHandler(env.NewGame)))
	r.Handle("/showgame/{gameID}", checkedChain.Then(errorHandler(env.ShowGame)))
	r.Handle("/takeseat/{gameID}", checkedChain.Then(errorHandler(env.TakeSeat)))
	r.Handle("/action/{action}/{gameID}", checkedChain.Then(errorHandler(env.Action))).Methods("PUT")
	return r
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
