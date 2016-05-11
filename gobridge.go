package main

import "C"

//export GoCreateRequest
func GoCreateRequest() uint32 {
  return CreateRequest()
}

//export GoBeginRequest
func GoBeginRequest(id uint32, rawHeaders *C.char) {
  BeginRequest(id, C.GoString(rawHeaders))
}

//export GoPollRequest
func GoPollRequest(id uint32) *C.char {
  cmd := PollRequest(id)
  if cmd == "" {
    return nil
  }
  return C.CString(cmd)
}

//export GoCancelRequest
func GoCancelRequest(id uint32) {
  CancelRequest(id)
}

//export GoSendRequestBodyChunk
func GoSendRequestBodyChunk(id uint32, chunk *C.char) {
  // TODO not a string
  SendRequestBodyChunk(id, []byte(C.GoString(chunk)))
}

//export GoSendLastRequestBodyChunk
func GoSendLastRequestBodyChunk(id uint32) {
  SendLastRequestBodyChunk(id)
}

//export GoInstallTestHandler
func GoInstallTestHandler() {
  SetTestRequestHandler()
}

func main() {
  panic("This is a library. No main.");
}
