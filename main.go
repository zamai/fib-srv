// My understanding of the task suggest that next progresses the sequance further
package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

const stateFile = "state.data"

// Server biolerplate is coppied from gorrila mux example
func main() {
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the App gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	app := newApp()

	r := mux.NewRouter()
	r.HandleFunc("/current", app.currentHandler).Methods("GET")
	r.HandleFunc("/next", app.nextHandler).Methods("GET")
	r.HandleFunc("/previous", app.previousHandler).Methods("GET")

	srv := &http.Server{
		Addr:         "0.0.0.0:8080",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r,
	}
	// Run our App in a goroutine so that it doesn't block.
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()
	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)

	// after all requests are shut down, save the state of the app to file
	saveString := fmt.Sprintf("%d,%d", app.prev, app.cur)
	err := ioutil.WriteFile(stateFile, []byte(saveString), 0777)
	if err != nil {
		log.Printf("error saving state to file: %v \n", err)
	}

	log.Println("shutting down")
	os.Exit(0)
}

func newApp() *App {
	// get fib val from file
	data, err := ioutil.ReadFile(stateFile)
	if err != nil {
		// file not found or error reading init new app
		return &App{cur: 1, prev: 0}
	}
	// Convention of saving numbers to file: previus,current
	savedNumbers := strings.Split(string(data), ",")
	prev, err := strconv.Atoi(savedNumbers[0])
	if err != nil {
		fmt.Printf("error restoring state from file: can not convert string to int: %v \n", err)
		return &App{cur: 1, prev: 0}
	}
	cur, err := strconv.Atoi(savedNumbers[1])
	if err != nil {
		fmt.Printf("error restoring state from file: can not convert string to int: %v \n", err)
		return &App{cur: 1, prev: 0}
	}
	return &App{cur: cur, prev: prev}
}

type App struct {
	cur  int
	prev int
	m *sync.RWMutex
}

func (a *App) currentHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "current -> %v\n", a.cur)
}

func (a *App) nextHandler(w http.ResponseWriter, r *http.Request) {
	// advancing the state of the sequance
	a.m.Lock()
	defer a.m.Unlock()
	next := a.prev + a.cur
	a.prev = a.cur
	a.cur = next
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "next -> %v\n", next)
}

func (a *App) previousHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "previous -> %v\n", a.prev)
}
