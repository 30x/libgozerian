package main

import (
  "bytes"
  "fmt"
  "strconv"
  "time"
  "net/http"
  . "github.com/onsi/ginkgo"
  . "github.com/onsi/gomega"
)

const (
  maxPolls = 20
  pollDelay = 100 * time.Millisecond
)

var _ = Describe("Go Management Interface", func() {
  var id uint32

  BeforeEach(func() {
    id = CreateRequest()
    Expect(id).ShouldNot(BeZero())
  })

  AfterEach(func() {
    FreeRequest(id)
  })

  It("Basic Request", func() {
    err := BeginRequest(id, makeHeaders("GET", "/pass", "", 0))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Slow Basic Request", func() {
    err := BeginRequest(id, makeHeaders("GET", "/slowpass", "", 0))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Invalid Request", func() {
    err := BeginRequest(id, InvalidRequest)
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(MatchRegexp("^ERRR.+"))
  })

  It("Not Found", func() {
    err := BeginRequest(id, makeHeaders("GET", "/notFoundAtAllNoWay", "", 0))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(MatchRegexp("^SWCH.*"))
    Expect(cmd[4:]).Should(Equal("404"))
    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Read request body no modify", func() {
    msg := []byte("Hello, World!")
    err := BeginRequest(id, makeHeaders("POST", "/readbody", "text/plain", len(msg)))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(Equal("RBOD"))
    SendRequestBodyChunk(id, true, msg)
    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
    Expect(bytes.Equal(msg, lastTestBody)).Should(BeTrue())
  })

  It("Read request body slowly", func() {
    msg := []byte("Hello, World!")
    err := BeginRequest(id, makeHeaders("POST", "/readbodyslow", "text/plain", len(msg)))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(Equal("RBOD"))
    SendRequestBodyChunk(id, true, msg)
    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
    fmt.Fprintf(GinkgoWriter, "Expected: %s\n", string(msg))
    fmt.Fprintf(GinkgoWriter, "Got:      %s\n", string(lastTestBody))
    Expect(bytes.Equal(msg, lastTestBody)).Should(BeTrue())
  })

  It("Read larger request body", func() {
    msg1 := []byte("Hello, World! ")
    msg2 := []byte("This is a slightly longer message")
    err := BeginRequest(id, makeHeaders("POST", "/readbody", "text/plain", len(msg1) + len(msg2)))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(Equal("RBOD"))
    SendRequestBodyChunk(id, false ,msg1)
    SendRequestBodyChunk(id, true, msg2)
    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
    fullMsg := append(msg1, msg2...)
    Expect(len(fullMsg)).Should(Equal(len(msg1) + len(msg2)))
    Expect(bytes.Equal(fullMsg, lastTestBody)).Should(BeTrue())
  })

  It("Read larger request body slowly", func() {
    msg1 := []byte("Hello, World! ")
    msg2 := []byte("This is a slightly longer message")
    err := BeginRequest(id, makeHeaders("POST", "/readbodyslow", "text/plain", len(msg1) + len(msg2)))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(Equal("RBOD"))
    SendRequestBodyChunk(id, false, msg1)
    SendRequestBodyChunk(id, true, msg2)
    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
    fullMsg := append(msg1, msg2...)
    Expect(len(fullMsg)).Should(Equal(len(msg1) + len(msg2)))
    Expect(bytes.Equal(fullMsg, lastTestBody)).Should(BeTrue())
  })

  It("Read and discard request body", func() {
    msg1 := []byte("Hello, World! ")
    msg2 := []byte("This is a slightly longer message")
    err := BeginRequest(id, makeHeaders("POST", "/readanddiscard", "text/plain", len(msg1) + len(msg2)))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(Equal("RBOD"))
    SendRequestBodyChunk(id, false, msg1)
    SendRequestBodyChunk(id, true, msg2)
    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
    // Don't care about final body since we discarded it
  })

  It("Modify request headers", func() {
    err := BeginRequest(id, makeHeaders("GET", "/writeheaders", "", 0))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(MatchRegexp("^WHDR.*"))
    hdrs := http.Header{}
    parseHeaders(hdrs, cmd[4:])
    Expect(hdrs.Get("X-Apigee-Test")).Should(Equal("HeaderTest"))
    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Modify request URL", func() {
    err := BeginRequest(id, makeHeaders("GET", "/writepath", "", 0))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(MatchRegexp("^WURI.*"))
    Expect(cmd[4:]).Should(Equal("/newpath"))
    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Modify request body no read", func() {
    err := BeginRequest(id, makeHeaders("POST", "/replacebody", "text/plain", 12))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(MatchRegexp("^WBOD.*"))
    body := readBodyData(cmd)

    expectedBod := []byte("Hello! I am the server!")
    Expect(bytes.Equal(expectedBod, body)).Should(BeTrue())

    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Modify response only", func() {
    err := BeginRequest(id, makeHeaders("GET", "/return201", "", 0))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(MatchRegexp("^SWCH.*"))
    Expect(cmd[4:]).Should(Equal("201"))
    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Send response with headers", func() {
    err := BeginRequest(id, makeHeaders("GET", "/returnheaders", "", 0))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(MatchRegexp("^SWCH.*"))
    Expect(cmd[4:]).Should(Equal("200"))
    cmd = doPoll(id)
    Expect(cmd).Should(MatchRegexp("^WHDR.*"))
    hdrs := http.Header{}
    parseHeaders(hdrs, cmd[4:])
    Expect(hdrs.Get("X-Apigee-Test")).Should(Equal("Return Header Test"))
    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Send response body", func() {
    err := BeginRequest(id, makeHeaders("GET", "/returnbody", "", 0))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(MatchRegexp("^SWCH.*"))
    Expect(cmd[4:]).Should(Equal("200"))

    cmd = doPoll(id)
    Expect(cmd).Should(MatchRegexp("^WBOD.*"))
    body := readBodyData(cmd)

    expectedBod := []byte("Hello! I am the server!")
    Expect(bytes.Equal(expectedBod, body)).Should(BeTrue())

    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Complete request modification", func() {
    err := BeginRequest(id, makeHeaders("POST", "/completerequest", "text/plain", 12))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(MatchRegexp("^WURI.*"))
    Expect(cmd[4:]).Should(Equal("/totallynewurl"))

    cmd = doPoll(id)
    Expect(cmd).Should(MatchRegexp("^WHDR.*"))
    hdrs := http.Header{}
    parseHeaders(hdrs, cmd[4:])
    Expect(hdrs.Get("X-Apigee-Test")).Should(Equal("Complete"))

    cmd = doPoll(id)
    Expect(cmd).Should(MatchRegexp("^WBOD.*"))
    expectedBod := []byte("Hello Again! Time for a complete rewrite!")
    bod := readBodyData(cmd)
    Expect(bytes.Equal(expectedBod, bod)).Should(BeTrue())

    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Complete response modification", func() {
    err := BeginRequest(id, makeHeaders("POST", "/completeresponse", "text/plain", 12))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(Equal("RBOD"))
    SendRequestBodyChunk(id, false, []byte("Hello, "))
    SendRequestBodyChunk(id, true, []byte("World!"))

    cmd = doPoll(id)
    Expect(cmd).Should(MatchRegexp("^SWCH.*"))
    Expect(cmd[4:]).Should(Equal("201"))

    cmd = doPoll(id)
    Expect(cmd).Should(MatchRegexp("^WHDR.*"))
    hdrs := http.Header{}
    parseHeaders(hdrs, cmd[4:])
    Expect(hdrs.Get("X-Apigee-Test")).Should(Equal("Complete"))

    cmd = doPoll(id)
    Expect(cmd).Should(MatchRegexp("^WBOD.*"))
    expectedBod := []byte("Hello Again! ")
    bod := readBodyData(cmd)
    Expect(bytes.Equal(expectedBod, bod)).Should(BeTrue())

    cmd = doPoll(id)
    Expect(cmd).Should(MatchRegexp("^WBOD.*"))
    expectedBod = []byte("Time for a complete rewrite!")
    bod = readBodyData(cmd)
    Expect(bytes.Equal(expectedBod, bod)).Should(BeTrue())

    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
  })
})

func doPoll(id uint32) string {
  for c := 0; c < maxPolls; c++ {
    cmd := PollRequest(id, false)
    if cmd != "" {
      return cmd
    }
    time.Sleep(pollDelay)
  }
  Expect(false).Should(BeTrue())
  return ""
}

func makeHeaders(method, uri, contentType string, bodyLen int) string {
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

func readBodyData(cmd string) []byte {
  id, err := strconv.ParseInt(cmd[4:], 16, 32)
  Expect(err).Should(Succeed())
  return readChunk(int32(id), true)
}
