package main

import (
  "fmt"
  "time"
  "io/ioutil"
  "net/http"
  "net/url"
)

var mainHandler RequestHandler

/*
 * Set the function that will handle all incoming HTTP requests.
 */
func SetRequestHandler(h RequestHandler) {
  mainHandler = h
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

func (h *testRequestHandler) HandleRequest(resp http.ResponseWriter, req *http.Request, proxyReq *ProxyRequest) {
  switch req.URL.Path {
  case "/pass":
    // Nothing to do

  case "/slowpass":
    time.Sleep(time.Second)

  case "/readbody":
    buf, err := ioutil.ReadAll(req.Body)
    if err != nil {
      fmt.Printf("Error reading body: %v\n", err)
    }
    h.lastBody = buf
    req.Body.Close()

  case "/replacebody":
    proxyReq.Write([]byte("Hello! I am the server!"))

  case "/writeheaders":
    proxyReq.Header().Add("Server", "Go Test Stuff")
    proxyReq.Header().Add("X-Apigee-Test", "HeaderTest")

  case "/writepath":
    newURL, _ := url.Parse("/newpath")
    proxyReq.SetURL(newURL)

  case "/return201":
    resp.WriteHeader(http.StatusCreated)

  case "/returnheaders":
    resp.Header().Add("X-Apigee-Test", "Return Header Test")
    resp.WriteHeader(http.StatusOK)

  case "/returnbody":
    resp.Write([]byte("Hello! I am the server!"))

  default:
    resp.WriteHeader(http.StatusNotFound)
  }
}
