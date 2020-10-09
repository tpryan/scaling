package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/logging"
	"github.com/tpryan/scaling/apitools"
	"github.com/tpryan/scaling/caching"
)

var (
	cache        *caching.Cache
	debug        = true
	port         = ""
	selfHostName = ""
	active       = false
	nodeID       = ""
	logger       *logging.Logger
)

func main() {
	var err error

	redisHost := os.Getenv("REDISHOST")
	redisPort := os.Getenv("REDISPORT")
	projectID := os.Getenv("PROJECTID")

	client, err := logging.NewClient(context.Background(), projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	logger = client.Logger("generator")

	port = fmt.Sprintf(":%s", os.Getenv("PORT"))
	if port == ":" {
		port = ":8080"
	}

	cache, err = caching.NewCache(redisHost, redisPort, debug)
	if err != nil {
		sdlog("could not start cache", err)
		log.Fatal(fmt.Errorf("could not start cache: %w", err))
	}

	nodeID, err = caching.CreateID()
	if err != nil {
		sdlog("could not create cache id", err)
		log.Fatal(fmt.Errorf("could not create cache id: %w", err))
	}

	selfHostName, err = getHostIP()
	if err != nil {
		selfHostName = "docker.for.mac.localhost:8082"
	}

	if err := cache.RegisterNode(nodeID, selfHostName, false); err != nil {
		msg := fmt.Sprintf("caching issue host: %s port: %s\n", redisHost, redisPort)
		sdlog(msg, err)
		log.Fatal(fmt.Errorf("could not register the node: %w", err))
	}

	go startPolling()

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/healthz", handleHealth)

	fmt.Printf("starting webserver\n")
	if err := http.ListenAndServe(port, nil); err != nil {
		sdlog("could not start webserver", err)
		log.Fatal(fmt.Errorf("could not start webserver: %w", err))
	}
}

func registerNode() {

	fmt.Printf("registering node\n")
	if err := cache.RegisterNode(nodeID, selfHostName, active); err != nil {
		sdlog("could not register node", err)
		fmt.Printf("could not register node\n")
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	apitools.Success(w, "ok")
	return
}

func startPolling() {
	fmt.Printf("starting register polling\n")
	for {
		time.Sleep(1 * time.Second)
		go registerNode()
	}
}

func ab(n, c, u string) ([]byte, error) {
	args := []string{"-l", "-n", n, "-c", c, "-v", "2", "-q", u}
	cmd := "ab"
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

	active = true
	if err := cache.RegisterNode(nodeID, selfHostName, active); err != nil {
		apitools.Error(w, err)
		return
	}

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

	active = false
	if err := cache.RegisterNode(nodeID, selfHostName, active); err != nil {
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
	return ioutil.WriteFile(name, data, 0644)
}

func getHostIP() (string, error) {
	client := metadata.NewClient(&http.Client{
		Transport: userAgentTransport{
			userAgent: "loadgenerator",
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

func sdlog(msg string, err error) {
	if err != nil {

		txt := fmt.Sprintf(msg+": %s", err)

		logger.Log(logging.Entry{Payload: txt, Severity: logging.Error})
		return
	}

	logger.Log(logging.Entry{Payload: msg, Severity: logging.Info})
	return
}
