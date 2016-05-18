package main

import (
  "fmt"
  "sync"
  "net/http"
)

/*
 * This is code that processes requests from C code. It takes in a request and returns
 * an ID, and then it has an API for that particular request.
 *
 */

/*
 * The table of requests. It is global. For maximum flexibility we will put
 * a lock around it.
 */

var requests = make(map[uint32]*request)
var requestsLock = &sync.Mutex{}
var lastRequestID uint32

/*
 * Create a new request object. It should be used once and only once.
 */
func CreateRequest() uint32 {
  requestsLock.Lock()
  defer requestsLock.Unlock()

  // After 2BB requests we will roll over. That should not be a problem.
  lastRequestID++
  id := lastRequestID
  req := newRequest(id)
  requests[id] = req
  return id
}

/*
 * Begin the request by sending in a set of headers.
 */
func BeginRequest(id uint32, rawHeaders string) error {
  req := getRequest(id)
  if req == nil {
    return fmt.Errorf("Unknown request: %d", id)
  }

  return req.begin(rawHeaders)
}

/*
 * Get status of the request, without blocking. The result will be a single
 * string that represents a command, or an empty string if there is none.
 * Commands are defined in commands.go.
 */
func PollRequest(id uint32, block bool) string {
  req := getRequest(id)
  if req == nil { return "" }

  if block {
    return req.poll()
  }
  return req.pollNB()
}

/*
 * Free the slot for a request.
 */
func FreeRequest(id uint32) {
  deleteRequest(id)
}

/*
 * Send some data to act as the request body.
 */
func SendRequestBodyChunk(id uint32, last bool, chunk []byte) {
  req := getRequest(id)
  if req == nil { return }
  if len(chunk) > 0 {
    req.bodies <- chunk
  }
  if last {
    close(req.bodies)
  }
}

func TransformHeaders(id uint32, hdrString string) string {
  req := getRequest(id)
  if req == nil { return "" }
  if req.headerFilter == nil { return "" }

  hdrs := http.Header{}
  parseHeaders(hdrs, hdrString)
  outHdrs := req.headerFilter(hdrs)
  if outHdrs == nil {
    return ""
  }
  return serializeHeaders(outHdrs)
}

func getRequest(id uint32) *request {
  requestsLock.Lock()
  defer requestsLock.Unlock()
  return requests[id]
}

func deleteRequest(id uint32) {
  requestsLock.Lock()
  delete(requests, id)
  requestsLock.Unlock()
}
