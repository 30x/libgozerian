package main

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	testHandler = "testHandler"
	testDebug   = false
)

func TestGo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Go Test")
}

var testHTTPServer *gozerianServer
var testURL string

var _ = BeforeSuite(func() {
	err := createHandler(testHandler, TestHandlerURI)
	Expect(err).Should(Succeed())

	testURL = os.Getenv("WEAVER_TEST_URL")

	if testURL == "" {
		testHTTPServer, err = startGozerianServer(0, "", TestHandlerURI)
		Expect(err).Should(Succeed())
		testHTTPServer.setDebug(testDebug)
		testPort := testHTTPServer.getPort()
		Expect(testPort).ShouldNot(BeZero())
		testURL = fmt.Sprintf("http://localhost:%d", testPort)
		go testHTTPServer.run()
		fmt.Fprintf(GinkgoWriter, "Running test against local server on port %d\n", testPort)
	} else {
		fmt.Printf("Running test against remote server at %s\n", testURL)
	}
})

var _ = AfterSuite(func() {
	destroyHandler(testHandler)

	if testHTTPServer != nil {
		testHTTPServer.stop()
	}
})
