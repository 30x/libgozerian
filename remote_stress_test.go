package main

import (
  "bytes"
  "fmt"
  "hash"
  "hash/crc64"
  "io"
  "io/ioutil"
  "math/rand"
  "net/http"
  "testing/quick"
  "time"
  . "github.com/onsi/ginkgo"
  . "github.com/onsi/gomega"
)

var _ = Describe("Remote Stress Test", func() {
  It("Pass POST", func() {
    url := fmt.Sprintf("%s/pass", testURL)
    err := quick.Check(func(msg string) bool {
      return testPOSTString(url, msg)
    }, nil)
    Expect(err).Should(Succeed())
  })

  It("Pass POST small binary", func() {
    url := fmt.Sprintf("%s/pass", testURL)
    success := testPOSTBinary(url, 100)
    Expect(success).Should(BeTrue())
  })
  It("Pass POST medium binary", func() {
    url := fmt.Sprintf("%s/pass", testURL)
    success := testPOSTBinary(url, 65539)
    Expect(success).Should(BeTrue())
  })
  It("Pass POST large binary", func() {
    url := fmt.Sprintf("%s/pass", testURL)
    success := testPOSTBinary(url, 2 * 1024 * 1024)
    Expect(success).Should(BeTrue())
  })
})

func testPOST(url string, bod io.Reader) (bool, io.ReadCloser) {
  resp, err := http.Post(url, "text/plain", bod)
  if err != nil {
    fmt.Fprintf(GinkgoWriter, "POST error: %s\n", err)
    return false, nil
  }
  if resp.StatusCode != http.StatusOK {
    fmt.Fprintf(GinkgoWriter, "Bad status %d\n", resp.StatusCode)
    return false, nil
  }
  return true, resp.Body
}

func testPOSTString(url, msg string) bool {
  reqBody := bytes.NewBufferString(msg)
  success, body := testPOST(url, reqBody)
  if !success { return false }

  defer body.Close()
  rbuf, err := ioutil.ReadAll(body)
  if err != nil {
    fmt.Fprintf(GinkgoWriter, "Body read error: %s\n", err)
    return false
  }

  respMsg := string(rbuf)
  if msg != respMsg {
    fmt.Fprintf(GinkgoWriter, "Invalid response: Expected \"%s\" got \"%s\"\n",
      msg, respMsg)
    return false
  }
  return true
}

func testPOSTBinary(url string, len int) bool {
  testRdr := newTestReader(len)
  success, body := testPOST(url, testRdr)
  if !success { return false }

  crc := crc64.New(crcTable)
  defer body.Close()
  buf := make([]byte, 8192)
  l, _ := body.Read(buf)
  for l > 0 {
    crc.Write(buf[:l])
    l, _ = body.Read(buf)
  }

  if crc.Sum64() != testRdr.hash.Sum64() {
    fmt.Fprintf(GinkgoWriter, "CRC %d != %d\n", testRdr.hash.Sum64(), crc.Sum64())
    return false
  }
  return true
}

var crcTable = crc64.MakeTable(crc64.ISO)
var randSrc = rand.NewSource(time.Now().UnixNano())

/*
 * This is a reader that returns random bytes forever.
 */
type bigTestReader struct {
  remaining int
  hash hash.Hash64
  rnd *rand.Rand
}

func newTestReader(len int) *bigTestReader {
  return &bigTestReader{
    remaining: len,
    hash: crc64.New(crcTable),
    rnd: rand.New(randSrc),
  }
}

func (r *bigTestReader) Read(buf []byte) (n int, err error) {
  toRead := len(buf)
  if toRead > r.remaining {
    toRead = r.remaining
  }

  if toRead > 0 {
    r.remaining -= toRead
    r.rnd.Read(buf[:toRead])
    r.hash.Write(buf[:toRead])
  }

  if r.remaining > 0 {
    return toRead, nil
  }
  return toRead, io.EOF
}
