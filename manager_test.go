package main

import (
  "bytes"
  "fmt"
  "regexp"
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
  It("Basic Request", func() {
    id := CreateRequest()
    Expect(id).ShouldNot(BeZero())
    err := BeginRequest(id, makeHeaders("GET", "/pass", "", 0))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Slow Basic Request", func() {
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

  It("Not Found", func() {
    id := CreateRequest()
    Expect(id).ShouldNot(BeZero())
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

  It("Modify request headers", func() {
    id := CreateRequest()
    Expect(id).ShouldNot(BeZero())
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
    id := CreateRequest()
    Expect(id).ShouldNot(BeZero())
    err := BeginRequest(id, makeHeaders("GET", "/writepath", "", 0))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(MatchRegexp("^WURI.*"))
    Expect(cmd[4:]).Should(Equal("/newpath"))
    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Modify request body no read", func() {
    id := CreateRequest()
    Expect(id).ShouldNot(BeZero())
    err := BeginRequest(id, makeHeaders("POST", "/replacebody", "text/plain", 12))
    Expect(err).Should(Succeed())
    cmd := doPoll(id)
    Expect(cmd).Should(MatchRegexp("^WBOD.*"))
    re, err := regexp.Compile("^WBOD([0-9a-f]+) (.+)$")
    Expect(err).Should(Succeed())

    expectedBod := "Hello! I am the server!"
    matches := re.FindStringSubmatch(cmd)
    Expect(matches).ShouldNot(BeNil())
    Expect(matches[1]).Should(Equal(fmt.Sprintf("%x", len(expectedBod))))
    Expect(matches[2]).Should(Equal(expectedBod))

    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Modify response only", func() {
    id := CreateRequest()
    Expect(id).ShouldNot(BeZero())
    err := BeginRequest(id, makeHeaders("GET", "/return201", "", 0))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(MatchRegexp("^SWCH.*"))
    Expect(cmd[4:]).Should(Equal("201"))
    cmd = doPoll(id)
    Expect(cmd).Should(Equal("DONE"))
  })

  It("Send response with headers", func() {
    id := CreateRequest()
    Expect(id).ShouldNot(BeZero())
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
    id := CreateRequest()
    Expect(id).ShouldNot(BeZero())
    err := BeginRequest(id, makeHeaders("GET", "/returnbody", "", 0))
    Expect(err).Should(Succeed())

    cmd := doPoll(id)
    Expect(cmd).Should(MatchRegexp("^SWCH.*"))
    Expect(cmd[4:]).Should(Equal("200"))

    cmd = doPoll(id)
    Expect(cmd).Should(MatchRegexp("^WBOD.*"))
    re, err := regexp.Compile("^WBOD([0-9a-f]+) (.+)$")
    Expect(err).Should(Succeed())

    expectedBod := "Hello! I am the server!"
    matches := re.FindStringSubmatch(cmd)
    Expect(matches).ShouldNot(BeNil())
    Expect(matches[1]).Should(Equal(fmt.Sprintf("%x", len(expectedBod))))
    Expect(matches[2]).Should(Equal(expectedBod))

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
