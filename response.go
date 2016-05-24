package main

import (
  "io"
  "reflect"
  "strconv"
  "net/http"
)

type response struct {
  id uint32
  handler Handler
  cmds chan command
  bodies chan []byte
  resp *http.Response
  req *http.Request
  origStatus int
  origHeaders http.Header
  origBody io.Reader
  readStarted bool
}

func newResponse(id uint32, h Handler) *response {
  r := response{
    id: id,
    handler: h,
    cmds: make(chan command, commandQueueSize),
    bodies: make(chan []byte, bodyQueueSize),
  }
  return &r
}

func (r *response) Commands() chan command {
  return r.cmds
}

func (r *response) Bodies() chan []byte {
  return r.bodies
}

func (r *response) StartRead() {
  // In this model, once body is read, we can no longer change headers or status.
  // This limitation may be specific to nginx -- if so then we will make it
  // configurable.
  r.readStarted = true
  r.flushHeaders()
}

func (r *response) begin(status uint32, rawHeaders string, req *request) error {
  r.req = req.req
  go r.startResponse(status, rawHeaders)
  return nil
}

func (r *response) pollNB() string {
  select {
  case cmd := <- r.cmds:
    return cmd.String()
  default:
    return ""
  }
}

func (r *response) poll() string {
  cmd := <- r.cmds
  return cmd.String()
}

func (r *response) startResponse(status uint32, rawHeaders string) {
  resp, err := parseHTTPResponse(status, rawHeaders)
  if err != nil {
    r.cmds <- createErrorCommand(err)
    return
  }

  resp.Request = r.req
  r.resp = resp
  r.origStatus = resp.StatusCode
  r.origHeaders = copyHeaders(resp.Header)

  resp.Body = &requestBody{
    handler: r,
  }
  r.origBody = resp.Body

  r.handler.HandleResponse(resp)

  if !r.readStarted {
    r.flushHeaders()
  }
  r.flushBody()

  r.cmds <- command{id: CmdDone}
}

func (r *response) flushHeaders() {
  if r.origStatus != r.resp.StatusCode {
    staCmd := command{
      id: WSTA,
      msg: strconv.Itoa(r.resp.StatusCode),
    }
    r.cmds <- staCmd
  }
  if !reflect.DeepEqual(r.origHeaders, r.resp.Header) {
    hdrCmd := command{
      id: WHDR,
      msg: serializeHeaders(r.resp.Header),
    }
    r.cmds <- hdrCmd
  }
}

func (r *response) flushBody() {
  if r.origBody != r.resp.Body {
    readAndSend(r, r.resp.Body)
  }
}
