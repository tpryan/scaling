package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"
	"github.com/tpryan/apitools"
	"github.com/tpryan/scaling/caching"
)

var (
	cache        *caching.Cache
	cacheEnabled = true
	debug        = true
	port         = ":8080"
	instance     = caching.Instance{}
	environment  = ""
)

func main() {
	var err error

	redisHost := os.Getenv("REDISHOST")
	redisPort := os.Getenv("REDISPORT")
	environment = os.Getenv("SCALE_ENV")

	instanceID, err := getID()
	if err != nil {
		log.Fatal(err)
	}

	instance.Env = environment
	instance.ID = instanceID

	cache, err = caching.NewCache(redisHost, redisPort, cacheEnabled, debug)
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/healthz", handleHealth)
	r.HandleFunc("/", handleRecord)

	http.Handle("/", r)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}

}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	apitools.Success(w, "ok")
	return
}

func handleRecord(w http.ResponseWriter, r *http.Request) {
	if err := cache.Record(instance); err != nil {
		apitools.Error(w, err)
		return
	}

	instance.Incr()

	apitools.JSON(w, instance)
	return
}

func getID() (string, error) {
	b, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return b.String(), nil
}
