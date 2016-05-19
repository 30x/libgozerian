package main

import (
  "fmt"
  "net/http"
)

/*
#include <stdlib.h>
*/
import "C"

/*
 * This represents a single request. The request, in turn, drives HTTP.
 * It is assumed that all function calls for a single request happen in the same
 * goroutine (that will be the case for an Nginx worker). However, request
 * processing itself may happen in a different goroutine.
 */

const (
  commandQueueSize = 100
  bodyQueueSize = 2
)

type request struct {
  req *http.Request
  resp *httpResponse
  proxyReq *ProxyRequest
  headerFilter func (hdrs http.Header) http.Header
  bodyFilter func (body []byte, last bool) []byte
  id uint32
  cmds chan command
  bodies chan []byte
  proxying bool
  readerClosed bool
}

func newRequest(id uint32) *request {
  r := request{
    id: id,
    proxying: true,
  }
  return &r
}

func (r *request) begin(rawHeaders string) error {
  r.cmds = make(chan command, commandQueueSize)
  r.bodies = make(chan []byte, bodyQueueSize)
  go r.startRequest(rawHeaders)
  return nil
}

func (r *request) pollNB() string {
  select {
  case cmd := <- r.cmds:
    return cmd.String()
  default:
    return ""
  }
}

func (r* request) poll() string {
  cmd := <- r.cmds
  return cmd.String()
}

func (r *request) startRequest(rawHeaders string) {
  req, err := parseHTTPHeaders(rawHeaders, true)
  if err != nil {
    r.cmds <- createErrorCommand(err)
    return
  }
  r.req = req

  r.resp = &httpResponse{
    req: r,
    httpReq: req,
  }

  req.Body = &requestBody{
    req: r,
  }

  r.proxyReq = &ProxyRequest{
    req: r,
    httpReq: req,
  }

  // Call handlers. They may write the request body or headers, or start
  // to write out a response.
  mainHandler.HandleRequest(r)

  // It's possible that not everything was cleaned up here.
  if r.proxying {
    r.proxyReq.flush()
  } else {
    r.resp.flush(http.StatusOK)
  }

  // This signals that everything is done.
  r.cmds <- command{id: CmdDone}
}

func (r *request) sendBodyChunk(chunk []byte) {
  if len(chunk) == 0 {
    return
  }

  chunkID := allocateChunk(chunk)

  cmd := command{
    id: WBOD,
    msg: fmt.Sprintf("%x", chunkID),
  }
  r.cmds <- cmd
}

func allocateChunk(chunk []byte) int32 {
  chunkLen := uint32(len(chunk))
  chunkPtr := C.malloc(C.size_t(chunkLen))
  copy((*[1<<30]byte)(chunkPtr)[:], chunk[:])
  chunkID := GoStoreChunk(chunkPtr, chunkLen)
  return chunkID
}

func (r *request) Request() *http.Request {
  return r.req
}

func (r *request) Response() http.ResponseWriter {
  return r.resp
}

func (r *request) ProxyRequest() *ProxyRequest {
  return r.proxyReq
}

func (r *request) SetHeaderFilter(filterFunc func (hdrs http.Header) http.Header) {
  r.headerFilter = filterFunc
}

func (r *request) SetBodyFilter(filterFunc func (body []byte, last bool) []byte) {
  r.bodyFilter = filterFunc
}
