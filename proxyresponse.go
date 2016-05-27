package main

import (
	"fmt"
	"net/http"
)

/*
 * This structure represents the proxy "response." If the code calls any
 * of these functions, then we "switch" and take over sending the
 * response. It matches the http.ResponseWriter interface.
 */

type httpResponse struct {
	handler        commandHandler
	headers        *http.Header
	headersFlushed bool
}

func (h *httpResponse) Header() http.Header {
	// Copy on write the headers the first time
	if h.headers == nil {
		// Copy headers from the original request, because they will change.
		newHeaders := copyHeaders(h.handler.Headers())
		h.headers = &newHeaders
	}
	return *(h.headers)
}

/*
 * Like the standard HTTP ResponseWriter, once the first chunk of the response
 * has been written, subsequent header changes have no effect.
 */
func (h *httpResponse) Write(buf []byte) (int, error) {
	// Flush ensures that headers are written only once and the first time
	h.handler.ResponseWritten()
	h.flush(http.StatusOK)
	sendBodyChunk(h.handler, buf)
	return len(buf), nil
}

func (h *httpResponse) WriteHeader(status int) {
	h.handler.ResponseWritten()
	h.flush(status)
}

func (h *httpResponse) flush(status int) {
	if h.headersFlushed {
		return
	}
	swchCmd := command{
		id:  SWCH,
		msg: fmt.Sprintf("%d", status),
	}
	h.handler.Commands() <- swchCmd

	if h.headers != nil {
		whdrCmd := command{
			id:  WHDR,
			msg: serializeHeaders(*h.headers),
		}
		h.handler.Commands() <- whdrCmd
	}

	h.headersFlushed = true
}
