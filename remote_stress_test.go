package main

import (
  "bytes"
  "fmt"
  "io/ioutil"
  "net/http"
  "testing/quick"
  . "github.com/onsi/ginkgo"
  . "github.com/onsi/gomega"
)

var _ = Describe("Remote Stress Test", func() {
  It("Pass POST", func() {
    url := fmt.Sprintf("%s/pass", testURL)
    err := quick.Check(func(msg string) bool {
      return testPOST(url, msg)
    }, nil)
    Expect(err).Should(Succeed())
  })
})

func testPOST(url, msg string) bool {
  buf := bytes.NewBufferString(msg)
  resp, err := http.Post(url, "text/plain", buf)
  if err != nil {
    fmt.Fprintf(GinkgoWriter, "POST error: %s\n", err)
    return false
  }
  if resp.StatusCode != http.StatusOK {
    fmt.Fprintf(GinkgoWriter, "Bad status %d\n", resp.StatusCode)
    return false
  }

  defer resp.Body.Close()
  rbuf, err := ioutil.ReadAll(resp.Body)
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
