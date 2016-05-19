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

func (h *testRequestHandler) HandleRequest(ctx RequestContext) {
  switch ctx.Request().URL.Path {
  case "/pass":
    // Nothing to do

  case "/slowpass":
    time.Sleep(time.Second)

  case "/readbody":
    buf, err := ioutil.ReadAll(ctx.Request().Body)
    if err != nil {
      fmt.Printf("Error reading body: %v\n", err)
    }
    lastTestBody = buf
    ctx.Request().Body.Close()

  case "/readbodyslow":
    tmp := make([]byte, 2)
    buf := &bytes.Buffer{}
    len, _ := ctx.Request().Body.Read(tmp)
    for len > 0 {
      buf.Write(tmp[0:len])
      len, _ = ctx.Request().Body.Read(tmp)
    }
    lastTestBody = buf.Bytes()
    ctx.Request().Body.Close()

  case "/readanddiscard":
    tmp := make([]byte, 2)
    ctx.Request().Body.Read(tmp)
    ctx.Request().Body.Close()

  case "/replacebody":
    ctx.ProxyRequest().Write([]byte("Hello! I am the server!"))

  case "/writeheaders":
    ctx.ProxyRequest().Header().Add("Server", "Go Test Stuff")
    ctx.ProxyRequest().Header().Add("X-Apigee-Test", "HeaderTest")

  case "/writepath":
    newURL, _ := url.Parse("/newpath")
    ctx.ProxyRequest().SetURL(newURL)

  case "/return201":
    ctx.Response().WriteHeader(http.StatusCreated)

  case "/returnheaders":
    ctx.Response().Header().Add("X-Apigee-Test", "Return Header Test")
    ctx.Response().WriteHeader(http.StatusOK)

  case "/returnbody":
    ctx.Response().Write([]byte("Hello! I am the server!"))

  case "/completerequest":
    newURL, _ := url.Parse("/totallynewurl")
    ctx.ProxyRequest().SetURL(newURL)
    ctx.ProxyRequest().Header().Add("X-Apigee-Test", "Complete")
    ctx.ProxyRequest().Write([]byte("Hello Again! "))
    ctx.ProxyRequest().Write([]byte("Time for a complete rewrite!"))

  case "/completeresponse":
    ioutil.ReadAll(ctx.Request().Body)
    ctx.Request().Body.Close()
    ctx.Response().Header().Add("X-Apigee-Test", "Complete")
    ctx.Response().WriteHeader(http.StatusCreated)
    ctx.Response().Write([]byte("Hello Again! "))
    ctx.Response().Write([]byte("Time for a complete rewrite!"))

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

  default:
    ctx.Response().WriteHeader(http.StatusNotFound)
  }
}
