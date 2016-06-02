package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const (
	// HTTP grammar regexps borrowed from Trireme source
	ctl       = "\\x00-\\x1f\\x7f"
	digits    = "[0-9]"
	lws       = "[ \\t]"
	notCtl    = "[^" + ctl + "]"
	separator = "\\(\\)<>@,;:\"/\\[\\]?+{} \t\\\\"
	// Huh? Texts =       "[[ \t][^" + Ctl + "]]"
	texts  = "[^" + ctl + "]"
	tokens = "[^" + separator + ctl + "]"

	headerLine  = "^(" + tokens + "+):" + lws + "*(" + notCtl + "*)" + lws + "*$"
	requestLine = "^(" + tokens + "+) (" + texts + "+) HTTP/(" + digits + ").(" + digits + ")" + lws + "*$"
)

var requestLineRe = regexp.MustCompile(requestLine)
var headerLineRe = regexp.MustCompile(headerLine)

func parseHTTPHeaders(rawHeaders string, hasRequestLine bool) (*http.Request, error) {
	req := http.Request{
		Header: make(map[string][]string),
	}

	lines := strings.Split(rawHeaders, "\r\n")

	for i, line := range lines {
		if hasRequestLine && (i == 0) {
			err := parseRequestLine(line, &req)
			if err != nil {
				return nil, err
			}
		} else {
			err := parseHeaderLine(line, &req)
			if err != nil {
				return nil, err
			}
		}
	}

	return &req, nil
}

func parseHTTPResponse(status uint32, rawHeaders string) (*http.Response, error) {
	resp := http.Response{
		Header:     make(map[string][]string),
		StatusCode: int(status),
		Status:     http.StatusText(int(status)),
		// Faking this for now
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}

	parseHeaders(resp.Header, rawHeaders)

	clHeader := resp.Header.Get("Content-Length")
	if clHeader != "" {
		cl, err := strconv.ParseInt(clHeader, 10, 64)
		if err != nil {
			resp.ContentLength = cl
		}
	}

	closeHeader := resp.Header.Get("Connection")
	if closeHeader == "close" {
		resp.Close = true
	}

	return &resp, nil
}

//serialize the headersMap back to a string
func serializeHeaders(headerMap http.Header) string {
	var buffer bytes.Buffer
	for key := range headerMap {
		values := headerMap[key]
		var valuesBuffer bytes.Buffer
		for i := 0; i < len(values); i++ {
			if values[i] != "" {
				if i > 0 {
					valuesBuffer.WriteString(",")
				}
				valuesBuffer.WriteString(values[i])
			}
		}
		val := valuesBuffer.String()
		buffer.WriteString(key)
		buffer.WriteString(": ")
		buffer.WriteString(val)
		buffer.WriteString("\n")
	}
	serializedHeaders := buffer.String()

	return serializedHeaders
}

/*
 * Parse the simplified header serialization format supported by
 * "serializeHeaders." This format is not the same as the HTTP standard.
 */
func parseHeaders(headerMap http.Header, rawHeaders string) {
	headerValues := strings.Split(rawHeaders, "\n")
	for _, header := range headerValues {
		keyValue := strings.Split(header, ": ")
		if len(keyValue) == 2 {
			key := keyValue[0]
			valueString := keyValue[1]
			if valueString != "" {
				values := strings.Split(valueString, ",")
				if _, ok := headerMap[key]; !ok {
					headerMap[key] = make([]string, len(values))
				}
				for i := 0; i < len(values); i++ {
					headerMap[key][i] = values[i]
				}
			}
		}

	}
}

func parseRequestLine(line string, req *http.Request) error {
	matches := requestLineRe.FindStringSubmatch(line)
	if matches == nil {
		return fmt.Errorf("Invalid HTTP request line: \"%s\"", line)
	}

	url, err := url.ParseRequestURI(matches[2])
	if err != nil {
		return err
	}

	major, err := strconv.Atoi(matches[3])
	if err != nil {
		return err
	}
	minor, err := strconv.Atoi(matches[4])
	if err != nil {
		return err
	}

	req.URL = url
	req.RequestURI = matches[2]
	req.Method = matches[1]
	req.ProtoMajor = major
	req.ProtoMinor = minor
	req.Proto = fmt.Sprintf("HTTP/%d.%d", major, minor)
	return nil
}

func parseHeaderLine(line string, req *http.Request) error {
	if "" == line {
		return nil
	}
	matches := headerLineRe.FindStringSubmatch(line)
	if matches == nil {
		return fmt.Errorf("Invalid HTTP header line: \"%s\"", line)
	}

	key := http.CanonicalHeaderKey(matches[1])
	val := matches[2]
	req.Header.Add(key, val)

	switch key {
	case "Host":
		req.Host = val
	case "Content-Length":
		len, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
		req.ContentLength = len
	}

	return nil
}
