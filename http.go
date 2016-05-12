package main

import (
  "bytes"
  "fmt"
  "regexp"
  "strconv"
  "strings"
  "net/http"
  "net/url"
)

const (
  // HTTP grammar regexps borrowed from Trireme source
  Ctl =         "\\x00-\\x1f\\x7f"
  Digits =      "[0-9]"
  LWS =         "[ \\t]"
  NotCtl =      "[^" + Ctl + "]"
  Separator =   "\\(\\)<>@,;:\"/\\[\\]?+{} \t\\\\"
  // Huh? Texts =       "[[ \t][^" + Ctl + "]]"
  Texts =       "[^" + Ctl + "]"
  Tokens =      "[^" + Separator + Ctl + "]"

  HeaderLine =    "^(" + Tokens + "+):" + LWS + "*(" + NotCtl + "*)" + LWS + "*$"
  RequestLine =   "^(" + Tokens + "+) (" + Texts + "+) HTTP/(" + Digits + ").(" + Digits + ")" + LWS + "*$"
)

var requestLineRe = regexp.MustCompile(RequestLine)
var headerLineRe = regexp.MustCompile(HeaderLine)

func parseHTTPRequest(rawHeaders string) (*http.Request, error) {
  req := http.Request{
    Header: make(map[string][]string),
  }

  lines := strings.Split(rawHeaders, "\r\n")

  for i, line := range(lines) {
    if i == 0 {
      err := parseRequestLine(line, &req)
      if err != nil { return nil, err }
    } else {
      err := parseHeaderLine(line, &req)
      if err != nil { return nil, err }
    }
  }

  return &req, nil
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

func parseRequestLine(line string, req *http.Request) error {
  matches := requestLineRe.FindStringSubmatch(line)
  if matches == nil {
    return fmt.Errorf("Invalid HTTP request line: \"%s\"", line)
  }

  url, err := url.ParseRequestURI(matches[2])
  if err != nil { return err }

  major, err := strconv.Atoi(matches[3])
  if err != nil { return err }
  minor, err := strconv.Atoi(matches[4])
  if err != nil { return err }

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
      if err != nil { return err }
      req.ContentLength = len
  }

  return nil
}
