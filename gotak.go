package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"

	log "github.com/Sirupsen/logrus"

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

	// Output to stdout instead of the default stderr
	// Can be any io.Writer
	log.SetOutput(os.Stdout)

	var debug = flag.Bool("debug", false, "debug mode")
	flag.Parse()

	if *debug {
		log.SetLevel(log.DebugLevel)
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	} else {
		log.SetLevel(log.WarnLevel)
	}

}

func main() {

	// ensure the database is setup
	db, err := InitSQLiteDB(viper.GetString("production.dbname"))
	if err != nil {
		log.Panicf("problem initializing db connection: %v", err)
	}
	defer db.Close()

	// set up the live database behind a Datastore interface for our methods to run against
	env := &DBenv{db}
	// Bind to a port and pass our router in, logging every request to Stdout
	log.Println(http.ListenAndServeTLS(":8000", sslCert, sslKey, handlers.LoggingHandler(os.Stdout, genRouter(env))))

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
