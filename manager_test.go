package main

import (
  "bytes"
  "fmt"
  "strconv"
  "net/http"
  . "github.com/onsi/ginkgo"
  . "github.com/onsi/gomega"
)

var _ = Describe("Go Management Interface", func() {
  var id uint32
  var rid uint32

  BeforeEach(func() {
    id = CreateRequest(testHandler)
    Expect(id).ShouldNot(BeZero())
    rid = CreateResponse(testHandler)
    Expect(rid).ShouldNot(BeZero())
  })

  AfterEach(func() {
    FreeRequest(id)
    FreeResponse(rid)
  })

  It("Basic Request", func() {
    err := BeginRequest(id, makeRequestHeaders("GET", "/pass", "", 0))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))

    err = BeginResponse(rid, id, 200, makeResponseHeaders("", 0))
    Expect(err).Should(Succeed())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Slow Basic Request", func() {
    err := BeginRequest(id, makeRequestHeaders("GET", "/slowpass", "", 0))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))

    err = BeginResponse(rid, id, 200, makeResponseHeaders("", 0))
    Expect(err).Should(Succeed())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Invalid Request", func() {
    err := BeginRequest(id, InvalidRequest)
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(MatchRegexp("^ERRR.+"))
  })

  It("Not Found", func() {
    err := BeginRequest(id, makeRequestHeaders("GET", "/notFoundAtAllNoWay", "", 0))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(MatchRegexp("^SWCH.*"))
    Expect(cmd[4:]).Should(Equal("404"))
    cmd = PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Read request body no modify", func() {
    msg := []byte("Hello, World!")
    err := BeginRequest(id, makeRequestHeaders("POST", "/readbody", "text/plain", len(msg)))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(Equal("RBOD"))
    SendRequestBodyChunk(id, true, msg)
    cmd = PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))
    Expect(bytes.Equal(msg, lastTestBody)).Should(BeTrue())

    err = BeginResponse(rid, id, 200, makeResponseHeaders("", 0))
    Expect(err).Should(Succeed())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Read request body slowly", func() {
    msg := []byte("Hello, World!")
    err := BeginRequest(id, makeRequestHeaders("POST", "/readbodyslow", "text/plain", len(msg)))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(Equal("RBOD"))
    SendRequestBodyChunk(id, true, msg)
    cmd = PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))
    fmt.Fprintf(GinkgoWriter, "Expected: %s\n", string(msg))
    fmt.Fprintf(GinkgoWriter, "Got:      %s\n", string(lastTestBody))
    Expect(bytes.Equal(msg, lastTestBody)).Should(BeTrue())

    err = BeginResponse(rid, id, 200, makeResponseHeaders("", 0))
    Expect(err).Should(Succeed())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Read larger request body", func() {
    msg1 := []byte("Hello, World! ")
    msg2 := []byte("This is a slightly longer message")
    err := BeginRequest(id, makeRequestHeaders("POST", "/readbody", "text/plain", len(msg1) + len(msg2)))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(Equal("RBOD"))
    SendRequestBodyChunk(id, false ,msg1)
    SendRequestBodyChunk(id, true, msg2)
    cmd = PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))
    fullMsg := append(msg1, msg2...)
    Expect(len(fullMsg)).Should(Equal(len(msg1) + len(msg2)))
    Expect(bytes.Equal(fullMsg, lastTestBody)).Should(BeTrue())

    err = BeginResponse(rid, id, 200, makeResponseHeaders("", 0))
    Expect(err).Should(Succeed())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Read larger request body slowly", func() {
    msg1 := []byte("Hello, World! ")
    msg2 := []byte("This is a slightly longer message")
    err := BeginRequest(id, makeRequestHeaders("POST", "/readbodyslow", "text/plain", len(msg1) + len(msg2)))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(Equal("RBOD"))
    SendRequestBodyChunk(id, false, msg1)
    SendRequestBodyChunk(id, true, msg2)
    cmd = PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))
    fullMsg := append(msg1, msg2...)
    Expect(len(fullMsg)).Should(Equal(len(msg1) + len(msg2)))
    Expect(bytes.Equal(fullMsg, lastTestBody)).Should(BeTrue())

    err = BeginResponse(rid, id, 200, makeResponseHeaders("", 0))
    Expect(err).Should(Succeed())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Read and discard request body", func() {
    msg1 := []byte("Hello, World! ")
    msg2 := []byte("This is a slightly longer message")
    err := BeginRequest(id, makeRequestHeaders("POST", "/readanddiscard", "text/plain", len(msg1) + len(msg2)))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(Equal("RBOD"))
    SendRequestBodyChunk(id, false, msg1)
    SendRequestBodyChunk(id, true, msg2)
    cmd = PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))
    // Don't care about final body since we discarded it

    err = BeginResponse(rid, id, 200, makeResponseHeaders("", 0))
    Expect(err).Should(Succeed())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Modify request headers", func() {
    err := BeginRequest(id, makeRequestHeaders("GET", "/writeheaders", "", 0))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(MatchRegexp("^WHDR.*"))
    hdrs := http.Header{}
    parseHeaders(hdrs, cmd[4:])
    Expect(hdrs.Get("X-Apigee-Test")).Should(Equal("HeaderTest"))
    cmd = PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))

    err = BeginResponse(rid, id, 200, makeResponseHeaders("", 0))
    Expect(err).Should(Succeed())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Modify request URL", func() {
    err := BeginRequest(id, makeRequestHeaders("GET", "/writepath", "", 0))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(MatchRegexp("^WURI.*"))
    Expect(cmd[4:]).Should(Equal("/newpath"))
    cmd = PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))

    err = BeginResponse(rid, id, 200, makeResponseHeaders("", 0))
    Expect(err).Should(Succeed())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Modify request body no read", func() {
    err := BeginRequest(id, makeRequestHeaders("POST", "/replacebody", "text/plain", 12))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(MatchRegexp("^WBOD.*"))
    body := readBodyData(cmd)

    expectedBod := []byte("Hello! I am the server!")
    Expect(bytes.Equal(expectedBod, body)).Should(BeTrue())

    cmd = PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))

    err = BeginResponse(rid, id, 200, makeResponseHeaders("", 0))
    Expect(err).Should(Succeed())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Modify response only", func() {
    err := BeginRequest(id, makeRequestHeaders("GET", "/return201", "", 0))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(MatchRegexp("^SWCH.*"))
    Expect(cmd[4:]).Should(Equal("201"))
    cmd = PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Send response with headers", func() {
    err := BeginRequest(id, makeRequestHeaders("GET", "/returnheaders", "", 0))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(MatchRegexp("^SWCH.*"))
    Expect(cmd[4:]).Should(Equal("200"))
    cmd = PollRequest(id, true)
    Expect(cmd).Should(MatchRegexp("^WHDR.*"))
    hdrs := http.Header{}
    parseHeaders(hdrs, cmd[4:])
    Expect(hdrs.Get("X-Apigee-Test")).Should(Equal("Return Header Test"))
    cmd = PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Send response body", func() {
    err := BeginRequest(id, makeRequestHeaders("GET", "/returnbody", "", 0))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(MatchRegexp("^SWCH.*"))
    Expect(cmd[4:]).Should(Equal("200"))

    cmd = PollRequest(id, true)
    Expect(cmd).Should(MatchRegexp("^WBOD.*"))
    body := readBodyData(cmd)

    expectedBod := []byte("Hello! I am the server!")
    Expect(bytes.Equal(expectedBod, body)).Should(BeTrue())

    cmd = PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Complete request modification", func() {
    err := BeginRequest(id, makeRequestHeaders("POST", "/completerequest", "text/plain", 12))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(MatchRegexp("^WURI.*"))
    Expect(cmd[4:]).Should(Equal("/totallynewurl"))

    cmd = PollRequest(id, true)
    Expect(cmd).Should(MatchRegexp("^WHDR.*"))
    hdrs := http.Header{}
    parseHeaders(hdrs, cmd[4:])
    Expect(hdrs.Get("X-Apigee-Test")).Should(Equal("Complete"))

    cmd = PollRequest(id, true)
    Expect(cmd).Should(MatchRegexp("^WBOD.*"))
    expectedBod := []byte("Hello Again! Time for a complete rewrite!")
    bod := readBodyData(cmd)
    Expect(bytes.Equal(expectedBod, bod)).Should(BeTrue())

    cmd = PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Complete response modification", func() {
    err := BeginRequest(id, makeRequestHeaders("POST", "/completeresponse", "text/plain", 12))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(Equal("RBOD"))
    SendRequestBodyChunk(id, false, []byte("Hello, "))
    SendRequestBodyChunk(id, true, []byte("World!"))

    cmd = PollRequest(id, true)
    Expect(cmd).Should(MatchRegexp("^SWCH.*"))
    Expect(cmd[4:]).Should(Equal("201"))

    cmd = PollRequest(id, true)
    Expect(cmd).Should(MatchRegexp("^WHDR.*"))
    hdrs := http.Header{}
    parseHeaders(hdrs, cmd[4:])
    Expect(hdrs.Get("X-Apigee-Test")).Should(Equal("Complete"))

    cmd = PollRequest(id, true)
    Expect(cmd).Should(MatchRegexp("^WBOD.*"))
    expectedBod := []byte("Hello Again! ")
    bod := readBodyData(cmd)
    Expect(bytes.Equal(expectedBod, bod)).Should(BeTrue())

    cmd = PollRequest(id, true)
    Expect(cmd).Should(MatchRegexp("^WBOD.*"))
    expectedBod = []byte("Time for a complete rewrite!")
    bod = readBodyData(cmd)
    Expect(bytes.Equal(expectedBod, bod)).Should(BeTrue())

    cmd = PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))

    err = BeginResponse(rid, id, 200, makeResponseHeaders("", 0))
    Expect(err).Should(Succeed())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Modify Response Headers", func() {
    err := BeginRequest(id, makeRequestHeaders("GET", "/writeresponseheaders", "", 0))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))

    err = BeginResponse(rid, id, 200, makeResponseHeaders("", 0))
    Expect(err).Should(Succeed())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(MatchRegexp("^WHDR.+"))
    hdrs := http.Header{}
    parseHeaders(hdrs, cmd[4:])
    Expect(hdrs.Get("X-Apigee-ResponseHeader")).Should(Equal("yes"))

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Modify Response Body", func() {
    err := BeginRequest(id, makeRequestHeaders("GET", "/transformbody", "", 0))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))

    err = BeginResponse(rid, id, 200, makeResponseHeaders("", 0))
    Expect(err).Should(Succeed())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(MatchRegexp("^WBOD.*"))
    body := readBodyData(cmd)

    expectedBod := []byte("We have transformed the response!")
    Expect(bytes.Equal(expectedBod, body)).Should(BeTrue())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Modify Response Status", func() {
    err := BeginRequest(id, makeRequestHeaders("GET", "/responseerror", "", 0))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))

    err = BeginResponse(rid, id, 200, makeResponseHeaders("", 0))
    Expect(err).Should(Succeed())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(MatchRegexp("^WSTA.+"))
    Expect(cmd[4:]).Should(Equal("500"))

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(MatchRegexp("^WBOD.*"))
    body := readBodyData(cmd)

    expectedBod := []byte("Error in the server!")
    Expect(bytes.Equal(expectedBod, body)).Should(BeTrue())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Modify Response Using Writer", func() {
    err := BeginRequest(id, makeRequestHeaders("GET", "/responseerror2", "", 0))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))

    err = BeginResponse(rid, id, 200, makeResponseHeaders("", 0))
    Expect(err).Should(Succeed())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(MatchRegexp("^SWCH.+"))
    Expect(cmd[4:]).Should(Equal("504"))

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(MatchRegexp("^WHDR.+"))
    hdrs := http.Header{}
    parseHeaders(hdrs, cmd[4:])
    Expect(hdrs.Get("X-Apigee-Response")).Should(Equal("error"))

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(MatchRegexp("^WBOD.*"))
    body := readBodyData(cmd)

    expectedBod := []byte("Response Error")
    Expect(bytes.Equal(expectedBod, body)).Should(BeTrue())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Transform Body Chunks", func() {
    err := BeginRequest(id, makeRequestHeaders("GET", "/transformbodychunks", "", 0))
    Expect(err).Should(Succeed())

    cmd := PollRequest(id, true)
    Expect(cmd).Should(Equal("DONE"))

    err = BeginResponse(rid, id, 200, makeResponseHeaders("", 0))
    Expect(err).Should(Succeed())

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(MatchRegexp("^WHDR.+"))
    hdrs := http.Header{}
    parseHeaders(hdrs, cmd[4:])
    Expect(hdrs.Get("X-Apigee-Transformed")).Should(Equal("yes"))
    Expect(hdrs.Get("X-Apigee-Invisible")).Should(BeEmpty())

    msg := []byte("Hello, Response Server!")
    cmd = PollResponse(rid, true)
    Expect(cmd).Should(Equal("RBOD"))
    SendResponseBodyChunk(rid, true, msg)

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(MatchRegexp("^WBOD.*"))
    body := readBodyData(cmd)

    fmt.Fprintf(GinkgoWriter, "Response body: %s", string(body))
    Expect(string(body)).Should(MatchRegexp("{ \\[.+\\] }"))

    cmd = PollResponse(rid, true)
    Expect(cmd).Should(Equal("DONE"))
  })
})

func makeRequestHeaders(method, uri, contentType string, bodyLen int) string {
  buf := &bytes.Buffer{}
  fmt.Fprintf(buf, "%s %s HTTP/1.1\r\n", method, uri)
  if bodyLen > 0 {
    fmt.Fprintf(buf, "Content-Length: %d\r\n", bodyLen)
  }
  if contentType != "" {
    fmt.Fprintf(buf, "Content-Type: %s\r\n", contentType)
  }
  fmt.Fprintf(buf, "Host: localhost:1234\r\n")
  fmt.Fprintf(buf, "\r\n")
  return buf.String()
}

func makeResponseHeaders(contentType string, bodyLen int) string {
  buf := &bytes.Buffer{}
  if bodyLen > 0 {
    fmt.Fprintf(buf, "Content-Length: %d\n", bodyLen)
  }
  if contentType != "" {
    fmt.Fprintf(buf, "Content-Type: %s\n", contentType)
  }
  fmt.Fprintf(buf, "Server: Some test thing\n")
  fmt.Fprintf(buf, "\n")
  return buf.String()
}

func readBodyData(cmd string) []byte {
  id, err := strconv.ParseInt(cmd[4:], 16, 32)
  Expect(err).Should(Succeed())
  return readChunk(int32(id), true)
}
