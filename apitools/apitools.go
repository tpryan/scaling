package apitools

import (
	"fmt"
	"log"
	"net/http"
)

// JSONProducer is an interface that spits out a JSON string version of itself
type JSONProducer interface {
	JSON() (string, error)
}

// JSON uses an http.ResponseWriter to send a JSON message responding to an
// API call.
func JSON(w http.ResponseWriter, j JSONProducer) {
	json, err := j.JSON()
	if err != nil {
		Error(w, err)
		return
	}
	Respond(w, http.StatusOK, json)
	return
}

// Success uses an http.ResponseWriter to send a JSON message responding to an
// API call.
func Success(w http.ResponseWriter, msg string) {
	s := fmt.Sprintf("{\"msg\":\"%s\"}", msg)

	if msg == "true" || msg == "false" {
		s = msg
	}

	Respond(w, http.StatusOK, s)
	return
}

// Error uses an http.ResponseWriter to send a JSON message of an error
// responding to an API call.
func Error(w http.ResponseWriter, err error) {
	s := fmt.Sprintf("{\"error\":\"%s\"}", err)
	Respond(w, http.StatusInternalServerError, s)
	return
}

// Respond does the basic response using the http.ResponseWriter
func Respond(w http.ResponseWriter, code int, msg string) {

	if code != http.StatusOK {
		log.Printf("Webserver : %s", msg)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	w.Write([]byte(msg))

	return
}
