package main

import (
  "bytes"
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
}

// help us a bit by saving test results for internal comparison
var lastTestBody []byte

func (h *testRequestHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
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
    lastTestBody = buf
    req.Body.Close()

  case "/readbodyslow":
    tmp := make([]byte, 2)
    buf := &bytes.Buffer{}
    len, _ := req.Body.Read(tmp)
    for len > 0 {
      buf.Write(tmp[0:len])
      len, _ = req.Body.Read(tmp)
    }
    lastTestBody = buf.Bytes()
    req.Body.Close()

  case "/readanddiscard":
    tmp := make([]byte, 2)
    req.Body.Read(tmp)
    req.Body.Close()

  case "/replacebody":
    req.Body = ioutil.NopCloser(bytes.NewReader([]byte("Hello! I am the server!")))

  case "/writeheaders":
    req.Header.Add("Server", "Go Test Stuff")
    req.Header.Add("X-Apigee-Test", "HeaderTest")

  case "/writepath":
    newURL, _ := url.Parse("/newpath")
    req.URL = newURL

  case "/return201":
    resp.WriteHeader(http.StatusCreated)

  case "/returnheaders":
    resp.Header().Add("X-Apigee-Test", "Return Header Test")
    resp.WriteHeader(http.StatusOK)

  case "/returnbody":
    resp.Write([]byte("Hello! I am the server!"))

  case "/completerequest":
    newURL, _ := url.Parse("/totallynewurl")
    req.URL = newURL
    req.Header.Add("X-Apigee-Test", "Complete")
    // TODO would like reader to return in two chunks
    req.Body = ioutil.NopCloser(
      bytes.NewReader([]byte("Hello Again! Time for a complete rewrite!")))
    //ctx.ProxyRequest().Write([]byte("Hello Again! "))
    //ctx.ProxyRequest().Write([]byte("Time for a complete rewrite!"))

  case "/completeresponse":
    ioutil.ReadAll(req.Body)
    req.Body.Close()
    resp.Header().Add("X-Apigee-Test", "Complete")
    resp.WriteHeader(http.StatusCreated)
    resp.Write([]byte("Hello Again! "))
    resp.Write([]byte("Time for a complete rewrite!"))

  case "/writeresponseheaders":

  case "/transformbody":

  case "/donttransformbody":

  case "/transformbodychunks":

  default:
    resp.WriteHeader(http.StatusNotFound)
  }
}

func (h *testRequestHandler) HandleResponse(req *http.Request, ctx ResponseContext) {
  switch req.URL.Path {
    case "/writeresponseheaders":
      ctx.SetHeaderFilter(func(h http.Header) http.Header {
        h.Add("X-Apigee-ResponseHeader", "yes")
        return h
      })

    case "/transformbody":
      ctx.SetBodyFilter(func(c []byte, last bool) []byte {
        if last {
          return []byte("This body has been transformed.")
        }
        return make([]byte, 0)
      })

    case "/donttransformbody":
      ctx.SetBodyFilter(func(c []byte, last bool) []byte {
        return c
      })

    case "/transformbodychunks":
      ctx.SetBodyFilter(func(c []byte, last bool) []byte {
        s := fmt.Sprintf("{%v} (len %d last %v)", c, len(c), last)
        return []byte(s)
      })
  }
}
