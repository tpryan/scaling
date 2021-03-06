// Package receiver contains an HTTP Cloud Function.
package receiver

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/teris-io/shortid"
	"github.com/tpryan/scaling/apitools"
	"github.com/tpryan/scaling/caching"
)

var (
	cache       *caching.Cache
	debug       = true
	port        = ""
	instance    = caching.Instance{}
	environment = ""
	endpoint    = ""
)

func init() {
	port = fmt.Sprintf(":%s", os.Getenv("PORT"))
	if port == ":" {
		port = ":8080"
	}

	redisHost := os.Getenv("REDISHOST")
	redisPort := os.Getenv("REDISPORT")
	environment = os.Getenv("SCALE_ENV")
	endpoint = os.Getenv("ENDPOINT")

	instanceID, err := getID()
	if err != nil {
		log.Fatal(err)
	}

	instance.Env = environment
	instance.ID = instanceID

	cache, err = caching.NewCache(redisHost, redisPort, debug)
	if err != nil {
		log.Fatal(fmt.Errorf("cannot connect to %s:%s: %s", redisHost, redisHost, err))
	}

	if err := cache.RegisterReceiver(environment, endpoint); err != nil {
		log.Fatal(fmt.Errorf("cannot register a new instance: %s", err))
	}
}

// Record takes a hit from load and records in Redis
func Record(w http.ResponseWriter, r *http.Request) {

	if err := cache.Record(instance); err != nil {
		apitools.Error(w, err)
		return
	}

	instance.Incr()

	apitools.JSON(w, instance)
	return

}

func getID() (string, error) {

	sid, err := shortid.New(1, shortid.DefaultABC, uint64(time.Now().Unix()))
	if err != nil {
		return "", err
	}

	return sid.Generate()
}
