package main

import (
  "fmt"
  "time"
  "io/ioutil"
  "net/http"
)

var requestHandler http.Handler

/*
 * Set the function that will handle all incoming HTTP requests.
 */
func SetRequestHandler(h http.Handler) {
  requestHandler = h
}

/*
 * Install a request handler that supports a canonical API that we can
 * use for testing.
 */
func SetTestRequestHandler() {
  SetRequestHandler(&testRequestHandler{});
}

/*
 * This is a built-in request handler that may be installed for testing.
 */
type testRequestHandler struct {
  lastBody []byte
}

func (h *testRequestHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
  if req.URL.Path == "/pass" {
    // Nothing to do
  } else if req.URL.Path == "/slowpass" {
    time.Sleep(time.Second)
  } else if req.URL.Path == "/readbody" && req.Method == "POST" {
    buf, err := ioutil.ReadAll(req.Body)
    if err != nil {
      fmt.Printf("Error reading body: %v\n", err)
    }
    h.lastBody = buf
    req.Body.Close()
  } else if req.URL.Path == "/writebody" {
    resp.Write([]byte("Hello! I am the server!"))
  } else {
    // TODO return 404
  }
}
