package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	CompleteRequestLength = "GET /foo/bar/baz HTTP/1.1\r\n" +
		"Host: mybox\r\n" +
		"User-Agent: Myself\r\n" +
		"Content-Length: 13\r\n" +
		"\r\n"
	CompleteRequestLengthBlankHeader = "GET /foo/bar/baz HTTP/1.1\r\n" +
		"Host: mybox\r\n" +
		"User-Agent:\r\n" +
		"Content-Length: 13\r\n" +
		"\r\n"
	CompleteRequestNoLength = "GET /foo/bar/baz HTTP/1.1\r\n" +
		"Host: mybox\r\n" +
		"User-Agent: Myself\r\n" +
		"\r\n"
	InvalidRequest = "GET yourself TOANUNNERY/2.0\r\n" +
		"Go: Right Now\r\n" +
		"\r\n"
)

var _ = Describe("HTTP Parsing", func() {
	It("Complete Request With Length", func() {
		req, err := parseHTTPHeaders(CompleteRequestLength, true)
		Expect(err).Should(Succeed())
		Expect(req.Method).Should(Equal("GET"))
		Expect(req.RequestURI).Should(Equal("/foo/bar/baz"))
		Expect(req.URL.Path).Should(Equal("/foo/bar/baz"))
		Expect(req.Proto).Should(Equal("HTTP/1.1"))
		Expect(req.ProtoMajor).Should(Equal(1))
		Expect(req.ProtoMinor).Should(Equal(1))
		Expect(req.Header.Get("Host")).Should(Equal("mybox"))
		Expect(req.Header.Get("User-Agent")).Should(Equal("Myself"))
		Expect(req.Header.Get("Content-Length")).Should(Equal("13"))
		Expect(req.Host).Should(Equal("mybox"))
		Expect(req.ContentLength).Should(BeEquivalentTo(13))
	})

	It("Complete Request With No Length", func() {
		req, err := parseHTTPHeaders(CompleteRequestNoLength, true)
		Expect(err).Should(Succeed())
		Expect(req.Method).Should(Equal("GET"))
		Expect(req.RequestURI).Should(Equal("/foo/bar/baz"))
		Expect(req.URL.Path).Should(Equal("/foo/bar/baz"))
		Expect(req.Proto).Should(Equal("HTTP/1.1"))
		Expect(req.ProtoMajor).Should(Equal(1))
		Expect(req.ProtoMinor).Should(Equal(1))
		Expect(req.Header.Get("Host")).Should(Equal("mybox"))
		Expect(req.Header.Get("User-Agent")).Should(Equal("Myself"))
		Expect(req.Host).Should(Equal("mybox"))
		Expect(req.ContentLength).Should(BeZero())
	})

	It("Complete Request With Blank Header", func() {
		req, err := parseHTTPHeaders(CompleteRequestLengthBlankHeader, true)
		Expect(err).Should(Succeed())
		Expect(req.Method).Should(Equal("GET"))
		Expect(req.RequestURI).Should(Equal("/foo/bar/baz"))
		Expect(req.URL.Path).Should(Equal("/foo/bar/baz"))
		Expect(req.Proto).Should(Equal("HTTP/1.1"))
		Expect(req.ProtoMajor).Should(Equal(1))
		Expect(req.ProtoMinor).Should(Equal(1))
		Expect(req.Header.Get("Host")).Should(Equal("mybox"))
		Expect(req.Header.Get("Content-Length")).Should(Equal("13"))
		Expect(req.Host).Should(Equal("mybox"))
		Expect(req.ContentLength).Should(BeEquivalentTo(13))
	})

	It("Invalid Request", func() {
		_, err := parseHTTPHeaders(InvalidRequest, true)
		Expect(err).ShouldNot(Succeed())
	})
})
