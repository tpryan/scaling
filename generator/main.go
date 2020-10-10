package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
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
	logWorking   = true
	receviers    = []*url.URL{}
)

func main() {
	var err error

	redisHost := os.Getenv("REDISHOST")
	redisPort := os.Getenv("REDISPORT")
	projectID := os.Getenv("PROJECTID")

	logger, err = getLogger(projectID)
	if err != nil {
		logWorking = false
		fmt.Printf("Failed to create logger or client: %v", err)
	}

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

	if err := cache.RegisterGenerator(nodeID, selfHostName, false); err != nil {
		msg := fmt.Sprintf("caching issue host: %s port: %s\n", redisHost, redisPort)
		sdlog(msg, err)
		log.Fatal(fmt.Errorf("could not register the generator: %w", err))
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

func verifyURL(URLString string) bool {
	fmt.Printf("URL: %s\n", URLString)
	u, err := url.Parse(URLString)
	if err != nil {
		fmt.Printf("Verified: true \n")
		return false
	}

	for _, v := range receviers {
		if v.Hostname() == u.Hostname() && v.Port() == u.Port() {
			fmt.Printf("Verified: true \n")
			return true
		}
	}

	fmt.Printf("Verified: true \n")
	return false
}

func getLogger(projectID string) (*logging.Logger, error) {
	client, err := logging.NewClient(context.Background(), projectID)
	if err != nil {
		return nil, fmt.Errorf("Failed to create client: %v", err)
	}

	return client.Logger("generator"), nil
}

func registerNode() {

	err := cache.RegisterGenerator(nodeID, selfHostName, active)
	if err != nil {
		sdlog("could not register node", err)
	}

	rs, err := cache.Receivers()
	if err != nil {
		sdlog("could not get generator list", err)
	}

	receviers, err = rs.URLList()
	if err != nil {
		sdlog("could not make generator url list", err)
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

	ok := verifyURL(urltohit)
	if !ok {
		apitools.Error(w, errors.New("only in registered generators can be used - no ddosing"))
		return
	}

	urltohit += "?token=" + token

	active = true
	if err := cache.RegisterGenerator(nodeID, selfHostName, active); err != nil {
		apitools.Error(w, err)
		return
	}

	fmt.Printf("sending load to %s \n", urltohit)
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
	fmt.Printf("load sent\n")
	fmt.Printf("%s\n", results)

	active = false
	if err := cache.RegisterGenerator(nodeID, selfHostName, active); err != nil {
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
	name := fmt.Sprintf("/go/src/generator/logs/log_%s.log", token)
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
	log.Printf("sdLog called, logworking is %t", logWorking)
	txt := fmt.Sprintf(msg+": %s", err)
	log.Printf("%s", msg)
	if !logWorking {
		log.Printf("%s", txt)
		return
	}

	if err != nil {
		logger.Log(logging.Entry{Payload: txt, Severity: logging.Error})
		return
	}

	logger.Log(logging.Entry{Payload: msg, Severity: logging.Info})
	return
}
