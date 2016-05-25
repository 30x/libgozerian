package main

import (
  "bytes"
  "flag"
  "fmt"
  "net"
  "net/http"
  "os"
  "os/signal"
  "strconv"
  "syscall"
  "unsafe"
)

/*
#include <stdlib.h>
*/
import "C"

const (
  defaultHandlerID = "default"
)

var defaultHandlerName = C.CString(defaultHandlerID)

/*
 * The weaver project is designed to build a shared library, not a "main."
 * However, for testing purposes we can start it up as an executable which
 * listens on an HTTP port.
 * In this mode, it can either act in echo mode by returning what is sent,
 * or it can act in proxy mode.
 */

type Server struct {
  listener *net.TCPListener
  target string
  debug bool
}

/*
 * Start the server listening on the specified HTTP port. If "proxyTarget"
 * is empty, then echo back all requests. Otherwise, proxy to that URL.
 * If "testHandler" is true, install a test handler for unit test purposes.
 * If "port" is 0, then listen on an ephemeral port.
 */
func StartWeaverServer(port int, proxyTarget string, testHandler bool) (*Server, error) {
  addr := net.TCPAddr{
    Port: port,
  }
  listener, err := net.ListenTCP("tcp", &addr)
  if err != nil { return nil, err }

  svr := Server{
    listener: listener,
    target: proxyTarget,
  }

  if testHandler {
    SetTestRequestHandler()
  }

  CreateHandler(defaultHandlerID, "")

  return &svr, nil
}

func (s *Server) Run() {
  handler := weaverHandler{
    target: s.target,
    debug: s.debug,
  }
  http.Serve(s.listener, &handler)
}

func (s *Server) Stop() {
  s.listener.Close()
}

func (s *Server) SetDebug(d bool) {
  s.debug = d
}

func (s *Server) GetPort() int {
  _, port, err := net.SplitHostPort(s.listener.Addr().String())
  if err != nil { return 0 }
  portNum, err := strconv.Atoi(port)
  if err != nil { return 0 }
  return portNum
}

type weaverHandler struct {
  target string
  debug bool
}

func (m *weaverHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
  defer req.Body.Close()

  // Although we have nice Go ways to call all these internal functions,
  // use the public C API so that we can get good test coverage.
  id := GoCreateRequest(defaultHandlerName)
  defer GoFreeRequest(id)
  rid := GoCreateResponse(defaultHandlerName)
  defer GoFreeResponse(rid)

  requestBody := &bytes.Buffer{}
  done := m.processRequest(resp, req, id, rid, requestBody)
  if !done {
    m.processResponse(resp, req, id, rid, requestBody)
  }
}

func (m *weaverHandler) processRequest(
  resp http.ResponseWriter, req *http.Request,
  id, rid uint32, requestBody *bytes.Buffer) bool {

  reqHdrs := &bytes.Buffer{}
  fmt.Fprintf(reqHdrs, "%s %s HTTP/1.1\r\n", req.Method, req.URL.Path)
  req.Header.Write(reqHdrs)

  cReqHdrs := C.CString(reqHdrs.String())
  defer C.free(unsafe.Pointer(cReqHdrs))
  GoBeginRequest(id, cReqHdrs)

  var cmd string
  proxying := true
  writingRequest := false
  responseCode := http.StatusOK
  proxyHeaders := req.Header
  //proxyPath := req.URL.Path
  sentHeaders := false

  for cmd != "DONE" && cmd != "ERRR" {
    rawCmd := GoPollRequest(id, 1)
    cmdBuf := C.GoString(rawCmd)
    C.free(unsafe.Pointer(rawCmd))
    cmd = cmdBuf[:4]
    msg := cmdBuf[4:]

    if m.debug {
      fmt.Printf("Command: \"%s\"\n", cmd)
    }

    switch cmd {
    case "ERRR":
      resp.WriteHeader(http.StatusInternalServerError)
      resp.Write([]byte(msg))
      return true
    case "RBOD":
      requestBody.ReadFrom(req.Body)
      ptr, len := sliceToPtr(requestBody.Bytes())
      GoSendRequestBodyChunk(id, 1, ptr, len)
      C.free(ptr)
    case "WHDR":
      if proxying {
        parseHeaders(proxyHeaders, msg)
      } else {
        parseHeaders(resp.Header(), msg)
      }
    case "WURI":
      //proxyPath = msg
    case "WBOD":
      chunk := getChunkData(msg)
      if proxying {
        if !writingRequest {
          requestBody.Reset()
          writingRequest = true
        }
        requestBody.Write(chunk)
      } else {
        if !sentHeaders {
          resp.WriteHeader(responseCode)
          sentHeaders = true
        }
        resp.Write(chunk)
      }
    case "SWCH":
      proxying = false
      responseCode, _ = strconv.Atoi(msg)
    case "DONE":
    default:
      sendHTTPError(fmt.Errorf("Unexpected command %s", cmd), resp)
      return true
    }
  }

  if !proxying {
    // Request path decided immediately to send a response
    if !sentHeaders {
      resp.WriteHeader(responseCode)
    }
    return true
  }

  if requestBody.Len() == 0 {
    requestBody.ReadFrom(req.Body)
  }
  return false
}

func (m *weaverHandler) processResponse(
  resp http.ResponseWriter, req *http.Request,
  id, rid uint32, requestBody *bytes.Buffer) {

  // TODO in target proxy mode, actually get target headers
  respHdrs := &bytes.Buffer{}
  respHdrs.WriteString("Server: Weaver Test Main\r\n")

  cRespHdrs := C.CString(respHdrs.String())
  defer C.free(unsafe.Pointer(cRespHdrs))

  GoBeginResponse(rid, id, http.StatusOK, cRespHdrs)

  var cmd string
  responseCode := http.StatusOK
  sentHeaders := false
  wroteBody := false

  for cmd != "DONE" && cmd != "ERRR" {
    rawCmd := GoPollResponse(rid, 1)
    cmdBuf := C.GoString(rawCmd)
    C.free(unsafe.Pointer(rawCmd))
    cmd = cmdBuf[:4]
    msg := cmdBuf[4:]

    if m.debug {
      fmt.Printf("Command: \"%s\"\n", cmd)
    }

    switch cmd {
    case "ERRR":
      resp.WriteHeader(http.StatusInternalServerError)
      resp.Write([]byte(msg))
      return
    case "WSTA", "SWCH":
      responseCode, _ = strconv.Atoi(msg)
    case "WHDR":
      parseHeaders(resp.Header(), msg)
    case "RBOD":
      ptr, len := sliceToPtr(requestBody.Bytes())
      GoSendResponseBodyChunk(rid, 1, ptr, len)
      C.free(ptr)
    case "WBOD":
      if !sentHeaders {
        resp.WriteHeader(responseCode)
        sentHeaders = true
      }
      chunk := getChunkData(msg)
      wroteBody = true
      resp.Write(chunk)
    case "DONE":
    default:
      sendHTTPError(fmt.Errorf("Unexpected command %s", cmd), resp)
    }
  }

  if m.target == "" {
    // Pretend that we are a proxy for another server by echoing the request
    if !sentHeaders {
      resp.WriteHeader(http.StatusOK)
    }
    if !wroteBody {
      requestBody.WriteTo(resp)
    }
  } else {
    sendHTTPError(fmt.Errorf("Didn't implement proxying to target yet"), resp)
  }
}

func getChunkData(rawID string) []byte {
  id, err := strconv.ParseInt(rawID, 16, 32)
  if err != nil { return nil }
  ptr := GoGetChunk(int32(id))
  len := GoGetChunkLength(int32(id))
  buf := ptrToSlice(ptr, len)
  GoReleaseChunk(int32(id))
  C.free(ptr)
  return buf
}

func sendHTTPError(err error, resp http.ResponseWriter) {
  fmt.Printf("Error: %s\n", err.Error())
  resp.Header().Set("Content-Type", "text/plain")
  resp.WriteHeader(http.StatusInternalServerError)
  resp.Write([]byte(err.Error()))
}

func main() {
  var port int
  var target string
  var testHandler bool

  flag.IntVar(&port, "p", 0, "(required) Port to listen on")
  flag.StringVar(&target, "t", "", "(optional) Target proxy URL")
  flag.BoolVar(&testHandler, "h", false, "(optional) Install a set of test handlers")
  flag.Parse()

  if !flag.Parsed() {
    flag.PrintDefaults()
    os.Exit(2)
  }

  server, err := StartWeaverServer(port, target, testHandler)
  if err != nil {
    fmt.Printf("Cannot start server: %s\n", err)
    os.Exit(3)
  }

  fmt.Printf("Listening on port %d\n", server.GetPort())

  doneChan := make(chan bool, 1)
  signalChan := make(chan os.Signal, 1)
  signal.Notify(signalChan, syscall.SIGINT)
  signal.Notify(signalChan, syscall.SIGTERM)

  go func() {
    <- signalChan
    doneChan <- true
  }()

  go server.Run()

  <- doneChan
  server.Stop()
}
