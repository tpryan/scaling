package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/tpryan/apitools"
	"github.com/tpryan/scaling/caching"
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

	redisHost := os.Getenv("REDISHOST")
	redisPort := os.Getenv("REDISPORT")

	port = fmt.Sprintf(":%s", os.Getenv("PORT"))
	if port == ":" {
		port = ":8080"
	}

	cache, err = caching.NewCache(redisHost, redisPort, debug)
	if err != nil {
		log.Fatal(err)
	}

	// r := mux.NewRouter()
	// r.HandleFunc("/healthz", handleHealth)
	// r.HandleFunc("/api/index", handleIndex)
	// r.HandleFunc("/api/nodelist", handleNodeList)
	// r.HandleFunc("/api/clear", handleClear)
	// r.HandleFunc("/api/distribute", handleDistribute)

	// fs := wrapHandler(http.FileServer(http.Dir("static")))
	// r.PathPrefix("/").Handler(fs)

	// http.Handle("/", r)

	http.HandleFunc("/healthz", handleHealth)
	http.HandleFunc("/api/index", handleIndex)
	http.HandleFunc("/api/nodelist", handleNodeList)
	http.HandleFunc("/api/clear", handleClear)
	http.HandleFunc("/api/distribute", handleDistribute)
	// http.Handle("/", http.StripPrefix("/static", http.FileServer(http.Dir("./static"))))

	http.Handle("/", http.FileServer(http.Dir("/static")))

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

func handleNodeList(w http.ResponseWriter, r *http.Request) {

	list, err := cache.ListNodes()
	if err != nil {
		fmt.Printf("%s\n", err)
	}

	apitools.JSON(w, list)

	return
}

func handleClear(w http.ResponseWriter, r *http.Request) {

	if err := cache.Clear(); err != nil {
		fmt.Printf("%s\n", err)
	}

	apitools.Success(w, "cleared")

	return
}

func handleDistribute(w http.ResponseWriter, r *http.Request) {

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

	ab, err := cache.Distribute(n, c, urltohit, token)
	if err != nil {
		fmt.Printf("%s\n", err)
	}

	apitools.JSON(w, ab)

	return
}

type notFoundRedirectRespWr struct {
	http.ResponseWriter // We embed http.ResponseWriter
	status              int
}

func (w *notFoundRedirectRespWr) WriteHeader(status int) {
	w.status = status // Store the status for our own use
	if status != http.StatusNotFound {
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *notFoundRedirectRespWr) Write(p []byte) (int, error) {
	if w.status != http.StatusNotFound {
		return w.ResponseWriter.Write(p)
	}
	return len(p), nil // Lie that we successfully written it
}

func wrapHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nfrw := &notFoundRedirectRespWr{ResponseWriter: w}
		h.ServeHTTP(nfrw, r)
		if nfrw.status == 404 {
			http.Redirect(w, r, "/index.html", http.StatusFound)
		}
	}
}
