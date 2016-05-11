package main

import (
  "bytes"
  "fmt"
  "time"
  . "github.com/onsi/ginkgo"
  . "github.com/onsi/gomega"
)

const (
  maxPolls = 20
  pollDelay = 100 * time.Millisecond
)

var _ = Describe("Go Management Interface", func() {
  It("Basic Request", func() {
    id := CreateRequest()
    Expect(id).ShouldNot(BeZero())
    err := BeginRequest(id, makeHeaders("GET", "/pass", "", 0))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Slow Request", func() {
    id := CreateRequest()
    Expect(id).ShouldNot(BeZero())
    err := BeginRequest(id, makeHeaders("GET", "/slowpass", "", 0))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Invalid Request", func() {
    id := CreateRequest()
    Expect(id).ShouldNot(BeZero())
    err := BeginRequest(id, InvalidRequest)
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(MatchRegexp("^ERRR.+"))
  })

  It("POST Request Body", func() {
    msg := []byte("Hello, World!")
    id := CreateRequest()
    Expect(id).ShouldNot(BeZero())
    err := BeginRequest(id, makeHeaders("POST", "/readbody", "text/plain", len(msg)))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(Equal("RBOD"))
    SendRequestBodyChunk(id, msg)
    SendLastRequestBodyChunk(id)
    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
    // TODO verify that the body received is what we sent.
  })

  It("GET Response Body", func() {
    id := CreateRequest()
    Expect(id).ShouldNot(BeZero())
    err := BeginRequest(id, makeHeaders("POST", "/writebody", "text/plain", 12))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(MatchRegexp("^WBOD.*"))
    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
  })
})

func doPoll(id uint32) string {
  for c := 0; c < maxPolls; c++ {
    cmd := PollRequest(id)
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
