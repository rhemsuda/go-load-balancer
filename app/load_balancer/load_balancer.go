package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"context"
	"log"
	"net/http"
	"sync"
	"time"
	"strconv"
	"bytes"
	"encoding/json"
)

type ErrorMessage struct {
	Error string `json:"error"`
}

type HttpError struct {
	Msg string
	Status int
}

// Define the error code for business server unavailable
serverUnavailableError: HttpError{ Msg: "business server not available", Status: http.StatusGatewayTimeout }

// Map of channels to hold the current server statuses
var statusChannels map[string](chan int64)

// Start the http server - create a go-routine which defers a WaitGroup until after finished processing
func startHttpServer(wg *sync.WaitGroup, port int) *http.Server {
	srv := &http.Server{Addr: fmt.Sprintf("%s%s", ":", strconv.Itoa(port))}

	http.HandleFunc("/", tryReverseString)

	go func() {
		defer wg.Done()

		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("load_balancer: ListenAndServe(): %v", err)
		}
	}()

	return srv
}

// This method will try to send a request to one of our business logic servers. It will try for 30 seconds and return a failure response if it can't.
// This method reads the status channels to determine the fastest server to send our request to
func tryReverseString(w http.ResponseWriter, req *http.Request) {
	httpClient := http.Client{}
	
	timeout := 30 * time.Second
	startTime := time.Now()
	
	// Attempt to send to a server for 30 seconds
	for time.Now().Sub(startTime) < timeout {
		
		// Get the fastest server URL by checking for the lowest response time
		var fastestServer string
		var previousResponseTime int64
		for serverUrl, v := range statusChannels {
			responseTime := <- v
			if responseTime > 0 {
				if previousResponseTime == 0 || responseTime < previousResponseTime {
					fastestServer = serverUrl
				}
				previousResponseTime = responseTime
			}
		}

		// If we found a server that is online, parse the request body and send it off to the server
		if len(fastestServer) > 0 {
			body, err := ioutil.ReadAll(req.Body)

			if err != nil {
				log.Printf("Error reading body: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Build a new URI from our request URI
			url := fmt.Sprintf("%s%s", fastestServer, req.RequestURI)

			// Create a new http request using information from initial request
			proxyReq, err := http.NewRequest(req.Method, url, bytes.NewReader(body))
			if err != nil {
				log.Printf("Error creating proxy request: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// Set the headers of the 'proxy request' and send off the request
			proxyReq.Header = req.Header
			resp, err := httpClient.Do(proxyReq)

			// If no errors - return the response from the business logic server back to the client
			if err == nil {
				defer resp.Body.Close()

				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					log.Printf("Error reading body: %v", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
					
				fmt.Fprintf(w, "%v", string(body))
				return
			}
		}	
	}

	// If it makes it to this point - it wasn't able to make a request within 30 seconds and responds with an error message
	respondWithErrorMessage(w, serverUnavailableError)
}

// This method constantly polls a serverURL on a separate thread and communicates the status through it's status channel
// The server status records the time difference between sending and receiving the request on the server
// If there is a failure - the status is set to -1
func pollServerStatus(serverUrl string, c chan int64) {
	for {
		currentTime := time.Now().UnixNano()
		resp, err := http.Get(serverUrl + "/status")
		if err == nil {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatalln(err)
			}
			serverTime, err := strconv.ParseInt(string(body), 10, 64)
			if err != nil {
				log.Fatalln(err)
			}
			c <- serverTime - currentTime
		} else {
			c <- -1
		}
		
		time.Sleep((1 * time.Second) / 4)
	}
}

// Helper method for returning an Error message. This gives us the '{ "error": "message" }' format that we're looking for
func respondWithErrorMessage(w http.ResponseWriter, err HttpError) {
	errorMessage := ErrorMessage{ Error: err.Msg }
	em, _ := json.Marshal(errorMessage)
	http.Error(w, string(em), err.Status)
}

// Main method: Get port from environment variables and start the server
// Create status channels for each server URL and start polling the server status
func main() {
	portArg := os.Getenv("PORT")
	port, err := strconv.Atoi(portArg)
	
	if err != nil {
		log.Fatalf("load_balancer: Environment variable 'PORT' cannot be converted to integer")
		panic(err)
	}

	// Statically define the server URLs for the business servers
	serverUrls := make([]string, 2)
	serverUrls[0] = "http://0.0.0.0:8001"
	serverUrls[1] = "http://0.0.0.0:8002"

	// Make the map of statusChannels
	statusChannels = make(map[string]chan int64)

	// Create channels for each server in the map - and then start polling
	for _, s := range serverUrls {
		statusChannels[s] = make(chan int64, 1)
		go pollServerStatus(s, statusChannels[s])
	}

	// Create wait group for the server requests
	httpWaitGroup := &sync.WaitGroup{}
	httpWaitGroup.Add(1)

	// Start the http server
	srv := startHttpServer(httpWaitGroup, port)
	log.Printf("business_server: listening on port %d", port)

	// Wait for waitgroup to finish before exiting
	httpWaitGroup.Wait()
}
