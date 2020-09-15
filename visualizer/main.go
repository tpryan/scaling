package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/tpryan/apitools"
	"github.com/tpryan/scaling/caching"
)

var (
	cache        *caching.Cache
	cacheEnabled = true
	debug        = true
	port         = ":8081"
	instance     = caching.Instance{}
	environment  = ""
)

func main() {
	var err error

	redisHost := os.Getenv("REDISHOST")
	redisPort := os.Getenv("REDISPORT")

	cache, err = caching.NewCache(redisHost, redisPort, cacheEnabled, debug)
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/healthz", handleHealth)
	r.HandleFunc("/api/index", handleIndex)
	r.HandleFunc("/api/clear", handleClear)
	r.HandleFunc("/", handleIndex)

	http.Handle("/", r)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}

}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	apitools.Success(w, "ok")
	return
}

func handleIndex(w http.ResponseWriter, r *http.Request) {

	index, err := cache.Index()
	if err != nil {
		fmt.Printf("%s\n", err)
	}

	apitools.JSON(w, index)

	return
}

func handleClear(w http.ResponseWriter, r *http.Request) {

	if err := cache.Clear(); err != nil {
		fmt.Printf("%s\n", err)
	}

	apitools.Success(w, "cleared")

	return
}
