package main

import (
  "io"
  "net/http"
)

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
}

func (r *Request) Begin(rawHeaders string) error {
  r.cmds = make(chan command, commandQueueSize)
  r.bodies = make(chan []byte, bodyQueueSize)
  go r.startRequest(rawHeaders)
  return nil
}

func (r *Request) Poll() string {
  select {
  case cmd := <- r.cmds:
    if cmd.id == CmdDone {
      deleteRequest(r.id)
    }
    return cmd.String()
  default:
    return ""
  }
}

func (r *Request) Cancel() {
  // TODO what exactly?
}

func (r *Request) startRequest(rawHeaders string) {
  req, err := parseHTTPRequest(rawHeaders)
  if err != nil {
    r.cmds <- createErrorCommand(err)
    return
  }

  resp := &httpResponse{
    req: r,
  }

  req.Body = &requestBody{
    req: r,
  }
  requestHandler.ServeHTTP(resp, req)

  r.cmds <- command{id: CmdDone}
}

type requestBody struct {
  req *Request
  started bool
  curBuf []byte
}

func (b *requestBody) Read(buf []byte) (int, error) {
  if !b.started {
    // First tell the caller that we need some data.
    b.req.cmds <- command{id: CmdGetBody}
    b.started = true
  }

  cb := b.curBuf
  if cb == nil {
    // Will return nil at end of channel.
    cb = <- b.req.bodies
  }

  if cb == nil {
    return 0, io.EOF
  }

  if len(cb) <= len(buf) {
    copy(buf, cb)
    //copy((*[1<<30]byte)(buf)[:], cb)
    b.curBuf = nil
    return len(cb), nil
  }

  copy(buf, cb[:len(buf)])
  return len(buf), nil
}

func (b *requestBody) Close() error {
  if b.started {
    // Need to clear the channel.
    b.curBuf = nil
    drained := <- b.req.bodies
    for drained != nil {
      drained = <- b.req.bodies
    }
  }
  return nil
}

type httpResponse struct {
  req *Request
  headers http.Header
}

func (h *httpResponse) Header() http.Header {
  // TODO plug in to "request" object, copy on write the headers, send back changes
  return h.headers
}

func (h *httpResponse) Write(buf []byte) (int, error) {
  // TODO need to respond with bytes, not a UTF-8-encoded string
  cmd := command{
    id: CmdWriteBody,
    msg: string(buf),
  }
  h.req.cmds <- cmd
  return len(buf), nil
}

func (h *httpResponse) WriteHeader(hdr int) {
  // TODO
}
