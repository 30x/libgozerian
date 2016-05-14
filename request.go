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

type Request struct {
  id uint32
  cmds chan command
  bodies chan []byte
  proxying bool
  readerClosed bool
}

func NewRequest(id uint32) *Request {
  r := Request{
    id: id,
    proxying: true,
  }
  return &r
}

func (r *Request) Begin(rawHeaders string) error {
  r.cmds = make(chan command, commandQueueSize)
  r.bodies = make(chan []byte, bodyQueueSize)
  go r.startRequest(rawHeaders)
  return nil
}

func (r *Request) PollNB() string {
  select {
  case cmd := <- r.cmds:
    return cmd.String()
  default:
    return ""
  }
}

func (r* Request) Poll() string {
  cmd := <- r.cmds
  return cmd.String()
}

func (r *Request) startRequest(rawHeaders string) {
  req, err := parseHTTPHeaders(rawHeaders, true)
  if err != nil {
    r.cmds <- createErrorCommand(err)
    return
  }

  resp := &httpResponse{
    req: r,
    httpReq: req,
  }

  req.Body = &requestBody{
    req: r,
  }

  proxyReq := &ProxyRequest{
    req: r,
    httpReq: req,
  }

  // Call handlers. They may write the request body or headers, or start
  // to write out a response.
  mainHandler.HandleRequest(resp, req, proxyReq)

  // It's possible that not everything was cleaned up here.
  if r.proxying {
    proxyReq.flush()
  } else {
    resp.flush(http.StatusOK)
  }

  // This signals that everything is done.
  r.cmds <- command{id: CmdDone}
}

func (r *Request) sendBodyChunk(chunk []byte) {
  if len(chunk) == 0 {
    return
  }

  chunkLen := uint32(len(chunk))
  chunkPtr := C.malloc(C.size_t(chunkLen))
  copy((*[1<<30]byte)(chunkPtr)[:], chunk[:])
  chunkID := GoStoreChunk(chunkPtr, chunkLen)

  cmd := command{
    id: WBOD,
    msg: fmt.Sprintf("%x", chunkID),
  }
  r.cmds <- cmd
}
