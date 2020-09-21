package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"
	"github.com/tpryan/apitools"
	"github.com/tpryan/scaling/caching"
	"google.golang.org/api/run/v1"
)

var (
	cache       *caching.Cache
	debug       = true
	port        = ""
	instance    = caching.Instance{}
	environment = ""
)

func main() {
	var err error

	port = fmt.Sprintf(":%s", os.Getenv("PORT"))
	if port == ":" {
		port = ":8080"
	}

	redisHost := os.Getenv("REDISHOST")
	redisPort := os.Getenv("REDISPORT")
	environment = os.Getenv("SCALE_ENV")

	instanceID, err := getID()
	if err != nil {
		log.Fatal(err)
	}

	instance.Env = environment
	instance.ID = instanceID

	cache, err = caching.NewCache(redisHost, redisPort, debug)
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

func getCloudRunURL() (string, error) {
	ctx := context.Background()
	runService, err := run.NewService(ctx)
	if err != nil {
		return "", err
	}

	svc, err := runService.Namespaces.Services.Get("loadreceiver").Do()
	if err != nil {
		return "", err
	}

	return svc.Status.Url, nil
}
