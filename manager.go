package main

import (
  "fmt"
  "sync"
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

var requests = make(map[uint32]*Request)
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
  req := Request{
    id: id,
  }
  requests[id] = &req

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

  return req.Begin(rawHeaders)
}

/*
 * Get status of the request, without blocking. The result will be a single
 * string that represents a command, or an empty string if there is none.
 * Commands are defined in commands.go.
 */
func PollRequest(id uint32) string {
  req := getRequest(id)
  if req == nil { return "" }

  return req.Poll()
}

/*
 * Cancel a request that has not yet completed.
 */
func CancelRequest(id uint32) {
  req := getRequest(id)
  if req != nil {
    req.Cancel()
    deleteRequest(id)
  }
}

/*
 * Send some data to act as the request body.
 */
func SendRequestBodyChunk(id uint32, chunk []byte) {
  req := getRequest(id)
  if req != nil {
    req.bodies <- chunk
  }
}

func SendLastRequestBodyChunk(id uint32) {
  req := getRequest(id)
  if req != nil {
    close(req.bodies)
  }
}

func getRequest(id uint32) *Request {
  requestsLock.Lock()
  defer requestsLock.Unlock()
  return requests[id]
}

func deleteRequest(id uint32) {
  requestsLock.Lock()
  delete(requests, id)
  requestsLock.Unlock()
}
