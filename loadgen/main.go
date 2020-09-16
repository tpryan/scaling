package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/tpryan/scaling/apitools"
)

var (
	port = ""
)

func main() {

	port = fmt.Sprintf(":%s", os.Getenv("PORT"))
	if port == ":" {
		port = ":8080"
	}

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/healthz", handleHealth)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	apitools.Success(w, "ok")
	return
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

	results, err := ab(n, c, urltohit)
	if err != nil {
		if err.Error() == "exit status 22" {
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
	msg := fmt.Sprintf("success - handled ab for token:%s on ip:%s", token, r.RemoteAddr)
	apitools.Respond(w, http.StatusOK, msg)
	return

}

func writeLog(data []byte, token string) error {
	name := fmt.Sprintf("/go/src/abrunner/logs/log_%s.log", token)
	fmt.Printf("Log Printed: %s\n", name)
	return ioutil.WriteFile(name, data, 0644)
}
