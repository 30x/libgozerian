package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Go Management Interface", func() {
	var id uint32
	var rid uint32

	BeforeEach(func() {
		id = createRequest(testHandler)
		Expect(id).ShouldNot(BeZero())
		rid = createResponse(testHandler)
		Expect(rid).ShouldNot(BeZero())
	})

	AfterEach(func() {
		freeRequest(id)
		freeResponse(rid)
	})

	It("Basic Request", func() {
		err := beginRequest(id, makeRequestHeaders("GET", "/pass", "", 0))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))

		err = beginResponse(rid, id, 200, makeResponseHeaders("", 0))
		Expect(err).Should(Succeed())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Slow Basic Request", func() {
		err := beginRequest(id, makeRequestHeaders("GET", "/slowpass", "", 0))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))

		err = beginResponse(rid, id, 200, makeResponseHeaders("", 0))
		Expect(err).Should(Succeed())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Invalid Request", func() {
		err := beginRequest(id, InvalidRequest)
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(MatchRegexp("^ERRR.+"))
	})

	It("Not Found", func() {
		err := beginRequest(id, makeRequestHeaders("GET", "/notFoundAtAllNoWay", "", 0))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(MatchRegexp("^SWCH.*"))
		Expect(cmd[4:]).Should(Equal("404"))
		cmd = pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Read request body no modify", func() {
		msg := []byte("Hello, World!")
		err := beginRequest(id, makeRequestHeaders("POST", "/readbody", "text/plain", len(msg)))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(Equal("RBOD"))
		sendRequestBodyChunk(id, true, msg)
		cmd = pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))
		Expect(bytes.Equal(msg, lastTestBody)).Should(BeTrue())

		err = beginResponse(rid, id, 200, makeResponseHeaders("", 0))
		Expect(err).Should(Succeed())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Read request body slowly", func() {
		msg := []byte("Hello, World!")
		err := beginRequest(id, makeRequestHeaders("POST", "/readbodyslow", "text/plain", len(msg)))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(Equal("RBOD"))
		sendRequestBodyChunk(id, true, msg)
		cmd = pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))
		fmt.Fprintf(GinkgoWriter, "Expected: %s\n", string(msg))
		fmt.Fprintf(GinkgoWriter, "Got:      %s\n", string(lastTestBody))
		Expect(bytes.Equal(msg, lastTestBody)).Should(BeTrue())

		err = beginResponse(rid, id, 200, makeResponseHeaders("", 0))
		Expect(err).Should(Succeed())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Read larger request body", func() {
		msg1 := []byte("Hello, World! ")
		msg2 := []byte("This is a slightly longer message")
		err := beginRequest(id, makeRequestHeaders("POST", "/readbody", "text/plain", len(msg1)+len(msg2)))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(Equal("RBOD"))
		sendRequestBodyChunk(id, false, msg1)
		sendRequestBodyChunk(id, true, msg2)
		cmd = pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))
		fullMsg := append(msg1, msg2...)
		Expect(len(fullMsg)).Should(Equal(len(msg1) + len(msg2)))
		Expect(bytes.Equal(fullMsg, lastTestBody)).Should(BeTrue())

		err = beginResponse(rid, id, 200, makeResponseHeaders("", 0))
		Expect(err).Should(Succeed())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Read larger request body slowly", func() {
		msg1 := []byte("Hello, World! ")
		msg2 := []byte("This is a slightly longer message")
		err := beginRequest(id, makeRequestHeaders("POST", "/readbodyslow", "text/plain", len(msg1)+len(msg2)))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(Equal("RBOD"))
		sendRequestBodyChunk(id, false, msg1)
		sendRequestBodyChunk(id, true, msg2)
		cmd = pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))
		fullMsg := append(msg1, msg2...)
		Expect(len(fullMsg)).Should(Equal(len(msg1) + len(msg2)))
		Expect(bytes.Equal(fullMsg, lastTestBody)).Should(BeTrue())

		err = beginResponse(rid, id, 200, makeResponseHeaders("", 0))
		Expect(err).Should(Succeed())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Read and discard request body", func() {
		msg1 := []byte("Hello, World! ")
		msg2 := []byte("This is a slightly longer message")
		err := beginRequest(id, makeRequestHeaders("POST", "/readanddiscard", "text/plain", len(msg1)+len(msg2)))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(Equal("RBOD"))
		sendRequestBodyChunk(id, false, msg1)
		sendRequestBodyChunk(id, true, msg2)
		cmd = pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))
		// Don't care about final body since we discarded it

		err = beginResponse(rid, id, 200, makeResponseHeaders("", 0))
		Expect(err).Should(Succeed())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Modify request headers", func() {
		err := beginRequest(id, makeRequestHeaders("GET", "/writeheaders", "", 0))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(MatchRegexp("^WHDR.*"))
		hdrs := http.Header{}
		parseHeaders(hdrs, cmd[4:])
		Expect(hdrs.Get("X-Apigee-Test")).Should(Equal("HeaderTest"))
		cmd = pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))

		err = beginResponse(rid, id, 200, makeResponseHeaders("", 0))
		Expect(err).Should(Succeed())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Modify request URL", func() {
		err := beginRequest(id, makeRequestHeaders("GET", "/writepath", "", 0))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(MatchRegexp("^WURI.*"))
		Expect(cmd[4:]).Should(Equal("/newpath"))
		cmd = pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))

		err = beginResponse(rid, id, 200, makeResponseHeaders("", 0))
		Expect(err).Should(Succeed())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Modify request body no read", func() {
		err := beginRequest(id, makeRequestHeaders("POST", "/replacebody", "text/plain", 12))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(MatchRegexp("^WBOD.*"))
		body := readBodyData(cmd)

		expectedBod := []byte("Hello! I am the server!")
		Expect(bytes.Equal(expectedBod, body)).Should(BeTrue())

		cmd = pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))

		err = beginResponse(rid, id, 200, makeResponseHeaders("", 0))
		Expect(err).Should(Succeed())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Modify response only", func() {
		err := beginRequest(id, makeRequestHeaders("GET", "/return201", "", 0))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(MatchRegexp("^SWCH.*"))
		Expect(cmd[4:]).Should(Equal("201"))
		cmd = pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Send response with headers", func() {
		err := beginRequest(id, makeRequestHeaders("GET", "/returnheaders", "", 0))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(MatchRegexp("^SWCH.*"))
		Expect(cmd[4:]).Should(Equal("200"))
		cmd = pollRequest(id, true)
		Expect(cmd).Should(MatchRegexp("^WHDR.*"))
		hdrs := http.Header{}
		parseHeaders(hdrs, cmd[4:])
		Expect(hdrs.Get("X-Apigee-Test")).Should(Equal("Return Header Test"))
		cmd = pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Send response body", func() {
		err := beginRequest(id, makeRequestHeaders("GET", "/returnbody", "", 0))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(MatchRegexp("^SWCH.*"))
		Expect(cmd[4:]).Should(Equal("200"))

		cmd = pollRequest(id, true)
		Expect(cmd).Should(MatchRegexp("^WBOD.*"))
		body := readBodyData(cmd)

		expectedBod := []byte("Hello! I am the server!")
		Expect(bytes.Equal(expectedBod, body)).Should(BeTrue())

		cmd = pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Complete request modification", func() {
		err := beginRequest(id, makeRequestHeaders("POST", "/completerequest", "text/plain", 12))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(MatchRegexp("^WURI.*"))
		Expect(cmd[4:]).Should(Equal("/totallynewurl"))

		cmd = pollRequest(id, true)
		Expect(cmd).Should(MatchRegexp("^WHDR.*"))
		hdrs := http.Header{}
		parseHeaders(hdrs, cmd[4:])
		Expect(hdrs.Get("X-Apigee-Test")).Should(Equal("Complete"))

		cmd = pollRequest(id, true)
		Expect(cmd).Should(MatchRegexp("^WBOD.*"))
		expectedBod := []byte("Hello Again! Time for a complete rewrite!")
		bod := readBodyData(cmd)
		Expect(bytes.Equal(expectedBod, bod)).Should(BeTrue())

		cmd = pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Complete response modification", func() {
		err := beginRequest(id, makeRequestHeaders("POST", "/completeresponse", "text/plain", 12))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(Equal("RBOD"))
		sendRequestBodyChunk(id, false, []byte("Hello, "))
		sendRequestBodyChunk(id, true, []byte("World!"))

		cmd = pollRequest(id, true)
		Expect(cmd).Should(MatchRegexp("^SWCH.*"))
		Expect(cmd[4:]).Should(Equal("201"))

		cmd = pollRequest(id, true)
		Expect(cmd).Should(MatchRegexp("^WHDR.*"))
		hdrs := http.Header{}
		parseHeaders(hdrs, cmd[4:])
		Expect(hdrs.Get("X-Apigee-Test")).Should(Equal("Complete"))

		cmd = pollRequest(id, true)
		Expect(cmd).Should(MatchRegexp("^WBOD.*"))
		expectedBod := []byte("Hello Again! ")
		bod := readBodyData(cmd)
		Expect(bytes.Equal(expectedBod, bod)).Should(BeTrue())

		cmd = pollRequest(id, true)
		Expect(cmd).Should(MatchRegexp("^WBOD.*"))
		expectedBod = []byte("Time for a complete rewrite!")
		bod = readBodyData(cmd)
		Expect(bytes.Equal(expectedBod, bod)).Should(BeTrue())

		cmd = pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))

		err = beginResponse(rid, id, 200, makeResponseHeaders("", 0))
		Expect(err).Should(Succeed())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Modify Response Headers", func() {
		err := beginRequest(id, makeRequestHeaders("GET", "/writeresponseheaders", "", 0))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))

		err = beginResponse(rid, id, 200, makeResponseHeaders("", 0))
		Expect(err).Should(Succeed())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(MatchRegexp("^WHDR.+"))
		hdrs := http.Header{}
		parseHeaders(hdrs, cmd[4:])
		Expect(hdrs.Get("X-Apigee-ResponseHeader")).Should(Equal("yes"))

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Modify Response Body", func() {
		err := beginRequest(id, makeRequestHeaders("GET", "/transformbody", "", 0))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))

		err = beginResponse(rid, id, 200, makeResponseHeaders("", 0))
		Expect(err).Should(Succeed())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(MatchRegexp("^WBOD.*"))
		body := readBodyData(cmd)

		expectedBod := []byte("We have transformed the response!")
		Expect(bytes.Equal(expectedBod, body)).Should(BeTrue())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Modify Response Status", func() {
		err := beginRequest(id, makeRequestHeaders("GET", "/responseerror", "", 0))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))

		err = beginResponse(rid, id, 200, makeResponseHeaders("", 0))
		Expect(err).Should(Succeed())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(MatchRegexp("^WSTA.+"))
		Expect(cmd[4:]).Should(Equal("500"))

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(MatchRegexp("^WBOD.*"))
		body := readBodyData(cmd)

		expectedBod := []byte("Error in the server!")
		Expect(bytes.Equal(expectedBod, body)).Should(BeTrue())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Modify Response Using Writer", func() {
		err := beginRequest(id, makeRequestHeaders("GET", "/responseerror2", "", 0))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))

		err = beginResponse(rid, id, 200, makeResponseHeaders("", 0))
		Expect(err).Should(Succeed())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(MatchRegexp("^SWCH.+"))
		Expect(cmd[4:]).Should(Equal("504"))

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(MatchRegexp("^WHDR.+"))
		hdrs := http.Header{}
		parseHeaders(hdrs, cmd[4:])
		Expect(hdrs.Get("X-Apigee-Response")).Should(Equal("error"))

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(MatchRegexp("^WBOD.*"))
		body := readBodyData(cmd)

		expectedBod := []byte("Response Error")
		Expect(bytes.Equal(expectedBod, body)).Should(BeTrue())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Transform Body Chunks", func() {
		err := beginRequest(id, makeRequestHeaders("GET", "/transformbodychunks", "", 0))
		Expect(err).Should(Succeed())

		cmd := pollRequest(id, true)
		Expect(cmd).Should(Equal("DONE"))

		err = beginResponse(rid, id, 200, makeResponseHeaders("", 0))
		Expect(err).Should(Succeed())

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(MatchRegexp("^WHDR.+"))
		hdrs := http.Header{}
		parseHeaders(hdrs, cmd[4:])
		Expect(hdrs.Get("X-Apigee-Transformed")).Should(Equal("yes"))
		Expect(hdrs.Get("X-Apigee-Invisible")).Should(BeEmpty())

		msg := []byte("Hello, Response Server!")
		cmd = pollResponse(rid, true)
		Expect(cmd).Should(Equal("RBOD"))
		sendResponseBodyChunk(rid, true, msg)

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(MatchRegexp("^WBOD.*"))
		body := readBodyData(cmd)

		fmt.Fprintf(GinkgoWriter, "Response body: %s", string(body))
		Expect(string(body)).Should(Equal("{Hello, Response Server!}"))

		cmd = pollResponse(rid, true)
		Expect(cmd).Should(Equal("DONE"))
	})

	It("Bad handler", func() {
		err := createHandler("bad", BadHandlerURI)
		Expect(err).ShouldNot(Succeed())
		id := createRequest("bad")
		Expect(id).Should(BeZero())
		id = createResponse("bad")
		Expect(id).Should(BeZero())
	})
})

var _ = Describe("Unique ID test", func() {
	It("ID format", func() {
		// Unique ID format is "ttttt.rrrr" where "ttttt" is time in milliseconds since
		// Unix Epoch.
		id := makeMessageID()
		fmt.Fprintf(GinkgoWriter, "Message ID: %s\n", id)
		Expect(id).Should(MatchRegexp("[0-9a-f]+\\.[0-9a-f]+"))
		sid := strings.Split(id, ".")
		Expect(len(sid)).Should(Equal(2))
		ts, err := strconv.ParseInt(sid[0], 16, 64)
		Expect(err).Should(Succeed())
		nt := time.Unix(ts/1000, ts%1000)
		// Sanity check that the timestamp is reasonable
		Expect(nt.Year()).Should(BeNumerically(">=", 2016))
	})

	It("Unique IDs", func() {
		numChannels := 20
		numIDs := 1000
		totalIDs := numChannels * numIDs

		allIDs := make(map[string]bool)
		newIDs := make(chan string, 1000)

		// Generate IDs in many goroutines to test parallel generation
		for i := 0; i < numChannels; i++ {
			go func() {
				for c := 0; c < numIDs; c++ {
					newIDs <- makeMessageID()
				}
			}()
		}

		// Receive new IDs and put them in a map
		for i := 0; i < totalIDs; i++ {
			id := <-newIDs
			allIDs[id] = true
		}

		// If there were any duplicates, then the map will be too small
		Expect(len(allIDs)).Should(Equal(totalIDs))
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
