package main

import (
  "errors"
  "net/http"
  "net/url"
)

/*
 * This struct is used to represent the proxy request. Modifications to it
 * will affect what is passed on to the target.
 */
type ProxyRequest struct {
  req *request
  httpReq *http.Request
  headers *http.Header
  url *url.URL
  headersFlushed bool
}

func (p *ProxyRequest) Header() http.Header {
  if p.headers == nil {
    // Copy headers from the original request, because they will change.
    newHeaders := copyHeaders(p.httpReq.Header)
    p.headers = &newHeaders
  }
  return *(p.headers)
}

func (p *ProxyRequest) SetURL(url *url.URL) {
  p.url = url
}

/*
 * Send back a chunk of the new request body. Once this has been called,
 * changes to the header and URI will no longer take effect.
 */
func (p *ProxyRequest) Write(chunk []byte) (int, error) {
  if !p.req.proxying {
    return 0, errors.New("Cannot write request body: Response already sent")
  }
  p.flush()
  p.req.sendBodyChunk(chunk)
  return len(chunk), nil
}

func (p *ProxyRequest) flush() {
  if p.headersFlushed {
    return
  }
  p.headersFlushed = true

  if p.url != nil {
    uriCmd := command{
      id: WURI,
      msg: p.url.String(),
    }
    p.req.cmds <- uriCmd
  }
  if p.headers != nil {
    hdrCmd := command{
      id: WHDR,
      msg: serializeHeaders(*p.headers),
    }
    p.req.cmds <- hdrCmd
  }
}

func copyHeaders(hdr http.Header) http.Header {
  newHeaders := http.Header{}
  for k, v := range(hdr) {
    newVal := make([]string, len(v))
    for i := range(v) {
      newVal[i] = v[i]
    }
    newHeaders[k] = newVal
  }
  return newHeaders
}
