package main

import (
	"fmt"
	"os"
	"io"
	"net/http"
	"io/ioutil"
	"log"
	"sync"
	"time"
	"strconv"
	"errors"
	"strings"
	"bytes"
	"unicode"
	"encoding/json"
	"github.com/golang/gddo/httputil/header"
)

type Message struct {
	Data string `json:"data"`
}

type ErrorMessage struct {
	Error string `json:"error"`
}

type HttpError struct {
	Msg string
	Status int
}

// Define the error code for an integer string
integerError := HttpError{ Msg: "data is an int and not a string", Status: http.StatusInternalServerError }

// Start the http server - create a go-routine which defers a WaitGroup until after finished processing
func startHttpServer(wg *sync.WaitGroup, port int) *http.Server {
	srv := &http.Server{Addr: fmt.Sprintf("%s%s", ":", strconv.Itoa(port))}

	// Request handlers
	http.HandleFunc("/", reverseString)
	http.HandleFunc("/status", getStatus)

	// Concurrent go-routine for serving requests
	go func() {
		defer wg.Done()

		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("business_server: ListenAndServe(): %v", err)
		}
	}()

	return srv
}

// Request handler: This method returns the current server time in Unix Nanoseconds so our client application can count
// the round-trip time of the request and choose the faster server
func getStatus(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "%v", time.Now().UnixNano())
}

// Request handler: This method validates the incoming packet and returns a reversed string if able.
// If the string is an integer - we return an error response
func reverseString(w http.ResponseWriter, req *http.Request) {

	// Validate json headers and throw an error if it fails
	if err := validateJSONHeaders(req); err.Status != 0 {
		respondWithErrorMessage(w, err)
		return
	}
	
	// Set max bytes to 1MB. Anything larger will return an error
	req.Body = http.MaxBytesReader(w, req.Body, 1048576)

	// Validate and Decode the JSON request and return the message
	message, err := decodeJSONRequest(req)
	if err.Status != 0 {
		respondWithErrorMessage(w, err)
		return
	}

	// Check if the message data is an integer - return an error
	if isInteger(message.Data) {
		respondWithErrorMessage(w, integerError)
		return
	}

	// Copy message data into current and create a byte buffer to hold the reversed string
	var current string = message.Data
	var reversed bytes.Buffer 

	// Loop through each byte in the string in reversed order and write to byte buffer
	for i := len(current) - 1; i >= 0; i-- {
		reversed.WriteByte(current[i])
	}

	// Convert byte buffer to string and encode the response
	message.Data = reversed.String()
	encodeJSONResponse(w, message)
}

// This method validates the JSON headers to ensure our content-type is indeed application/json
func validateJSONHeaders(req *http.Request) HttpError {
	var httpError HttpError
	if req.Header.Get("Content-Type") != "" {
		contentType, _ := header.ParseValueAndParams(req.Header, "Content-Type")
		if contentType != "application/json" {
			httpError.Msg = "Content-Type header is not application/json"
			httpError.Status = http.StatusUnsupportedMediaType
		}
	}
	
	return httpError
}

// This method encodes a JSON object using our ResponseWriter to send it back to the client
func encodeJSONResponse(w http.ResponseWriter, message Message) {
	encoder := json.NewEncoder(w)
	encoder.Encode(message)
}

// This method decodes a JSON object and checks to ensure it is properly formed
func decodeJSONRequest(req *http.Request) (Message, HttpError) {
	var message Message
	var httpError HttpError

	// Read all data from the request body
	body, err := ioutil.ReadAll(req.Body)
	
	if err != nil {
		log.Printf("Error reading body: %v", err)
		httpError.Msg = "Can't read request body: " + err.Error()
		httpError.Status = http.StatusBadRequest
	} else {
		// Initialize the JSON decoder which will throw an error if keys do not match decoded type
		jsonStream := strings.NewReader(string(body))
		decoder := json.NewDecoder(jsonStream)
		decoder.DisallowUnknownFields()

		// If here is an error from decoding our message - check to see what was wrong and return the appropriate error response to the client
		if err := decoder.Decode(&message); err != nil {
			var syntaxError *json.SyntaxError
			var unmarshalTypeError *json.UnmarshalTypeError

			switch {
			case errors.As(err, &syntaxError):
				httpError.Msg = fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
				httpError.Status = http.StatusBadRequest
			case errors.Is(err, io.ErrUnexpectedEOF):
				httpError.Msg = fmt.Sprintf("Request body contains badly-formed JSON")
				httpError.Status = http.StatusBadRequest
			case errors.As(err, &unmarshalTypeError):
				httpError.Msg = fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
				httpError.Status = http.StatusBadRequest
			case strings.HasPrefix(err.Error(), "json: unknown field "):
				fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
				httpError.Msg = fmt.Sprintf("Request body contains unknown field %s", fieldName)
				httpError.Status = http.StatusBadRequest
			case errors.Is(err, io.EOF):
				httpError.Msg = "Request body must not be empty"
				httpError.Status = http.StatusBadRequest
			case err.Error() == "http: request body too large":
				httpError.Msg = "Request body must not be larger than 1MB"
				httpError.Status = http.StatusRequestEntityTooLarge
			default:
				log.Println(err.Error())
				httpError.Msg = http.StatusText(http.StatusInternalServerError)
				httpError.Status = http.StatusInternalServerError
			}
		} else if err := decoder.Decode(&struct{}{}); err != io.EOF {
			httpError.Msg = "Request body must only contain a single JSON object"
			httpError.Status = http.StatusBadRequest
		}	
	}
	
	return message, httpError
}

// Helper method for returning an Error message. This gives us the '{ "error": "message" }' format that we're looking for
func respondWithErrorMessage(w http.ResponseWriter, err HttpError) {
	errorMessage := ErrorMessage{ Error: err.Msg }
	em, _ := json.Marshal(errorMessage)
	http.Error(w, string(em), err.Status)
}

// Helper method to determine if all characters in a string are numeric
func isInteger(s string) bool {
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

// Main method: Get port from the environment variables and start the server. Wait for responses to finish before exiting if the server is killed. This helps avoid missed packets.
func main() {
	portArg := os.Getenv("PORT")
	port, err := strconv.Atoi(portArg)
	
	if err != nil {
		log.Fatalf("business_server: Environment variable 'PORT' cannot be converted to integer")
		panic(err)
	}

	// Create wait group for the server requests
	httpWaitGroup := &sync.WaitGroup{}
	httpWaitGroup.Add(1)

	// Start the http server
	startHttpServer(httpWaitGroup, port)
	log.Printf("business_server: listening on port %d", port)

	// Wait for waitgroup to finish before exiting
	httpWaitGroup.Wait()
}

