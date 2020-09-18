package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"cloud.google.com/go/compute/metadata"
	"github.com/tpryan/scaling/apitools"
	"github.com/tpryan/scaling/caching"
)

var (
	cache        *caching.Cache
	debug        = true
	port         = ""
	selfHostName = ""
)

func main() {
	var err error

	redisHost := os.Getenv("REDISHOST")
	redisPort := os.Getenv("REDISPORT")

	port = fmt.Sprintf(":%s", os.Getenv("PORT"))
	if port == ":" {
		port = ":8080"
	}

	cache, err = caching.NewCache(redisHost, redisPort, debug)
	if err != nil {
		log.Fatal(fmt.Errorf("could not start cache: %w", err))
	}

	selfHostName, err = getHostIP()
	if err != nil {
		selfHostName = "docker.for.mac.localhost:8082"
	}

	if err := cache.RegisterNode(selfHostName); err != nil {
		fmt.Printf("caching issue host: %s port: %s\n", redisHost, redisPort)
		log.Fatal(fmt.Errorf("could not register the node: %w", err))
	}

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/healthz", handleHealth)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(fmt.Errorf("could not start webserver: %w", err))
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	apitools.Success(w, "ok")
	return
}

func ab(n, c, u string) ([]byte, error) {
	args := []string{"-l", "-n", n, "-c", c, "-v", "2", "-q", u}
	cmd := "ab"
	fmt.Printf("%s %s\n", cmd, args)
	return exec.Command(cmd, args...).Output()
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	n := r.URL.Query().Get("n")
	c := r.URL.Query().Get("c")
	urltohit := r.URL.Query().Get("url")

	if len(n) == 0 {
		apitools.Error(w, errors.New("n request variable not set"))
		return
	}

	if len(c) == 0 {
		apitools.Error(w, errors.New("c request variable not set"))
		return
	}

	if len(urltohit) == 0 {
		apitools.Error(w, errors.New("url request variable not set"))
		return
	}

	urltohit += "?token=" + token

	results, err := ab(n, c, urltohit)
	if err != nil {
		if err.Error() == "exit status 22" {
			fmt.Printf("urltohit: %s\n", urltohit)
			fmt.Printf("results: %s\n", results)
			fmt.Printf("error: %s\n", err)
			apitools.Error(w, fmt.Errorf("might be an issue with env variable `urltohit=%s` ", urltohit))
			return
		}
		apitools.Error(w, err)
		return
	}
	if err := writeLog(results, token); err != nil {
		apitools.Error(w, err)
		return
	}

	msg := caching.ABResponse{Token: token, IP: r.RemoteAddr, Status: "success"}
	apitools.JSON(w, msg)
	return

}

func writeLog(data []byte, token string) error {
	name := fmt.Sprintf("/go/src/abrunner/logs/log_%s.log", token)
	fmt.Printf("Log Printed: %s\n", name)
	return ioutil.WriteFile(name, data, 0644)
}

func getHostIP() (string, error) {
	client := metadata.NewClient(&http.Client{
		Transport: userAgentTransport{
			userAgent: "gcprelay-query",
			base:      http.DefaultTransport,
		},
		Timeout: 1 * time.Second})

	return client.ExternalIP()
}

// userAgentTransport sets the User-Agent header before calling base.
type userAgentTransport struct {
	userAgent string
	base      http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface.
func (t userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.userAgent)
	return t.base.RoundTrip(req)
}
