package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"

	"github.com/30x/gozerian/pipeline"
)

/*
#include <stdlib.h>
*/
import "C"

const (
	bodyBufSize = 32767
)

/*
 * This represents a single request. The request, in turn, drives HTTP.
 * It is assumed that all function calls for a single request happen in the same
 * goroutine (that will be the case for an Nginx worker). However, request
 * processing itself may happen in a different goroutine.
 */

const (
	commandQueueSize = 100
	bodyQueueSize    = 2
)

type request struct {
	pipeDef      pipeline.Definition
	req          *http.Request
	resp         *httpResponse
	origHeaders  http.Header
	origURL      *url.URL
	origBody     io.ReadCloser
	id           uint32
	cmds         chan command
	bodies       chan []byte
	proxying     bool
	readerClosed bool
}

func newRequest(id uint32, pd pipeline.Definition) *request {
	r := request{
		pipeDef:  pd,
		id:       id,
		proxying: true,
	}
	return &r
}

func (r *request) Commands() chan command {
	return r.cmds
}

func (r *request) Bodies() chan []byte {
	return r.bodies
}

func (r *request) Headers() http.Header {
	return r.req.Header
}

func (r *request) ResponseWritten() {
	r.proxying = false
}

func (r *request) StartRead() {
}

func (r *request) begin(rawHeaders string) error {
	r.cmds = make(chan command, commandQueueSize)
	r.bodies = make(chan []byte, bodyQueueSize)
	go r.startRequest(rawHeaders)
	return nil
}

func (r *request) pollNB() string {
	select {
	case cmd := <-r.cmds:
		return cmd.String()
	default:
		return ""
	}
}

func (r *request) poll() string {
	cmd := <-r.cmds
	return cmd.String()
}

func (r *request) startRequest(rawHeaders string) {
	req, err := parseHTTPHeaders(rawHeaders, true)
	if err != nil {
		r.cmds <- createErrorCommand(err)
		return
	}
	// Save headers for later
	r.origHeaders = copyHeaders(req.Header)
	r.origURL = req.URL
	r.req = req

	resp := &httpResponse{
		handler: r,
	}
	r.resp = resp

	req.Body = &requestBody{
		handler: r,
	}
	r.origBody = req.Body

	// Call handlers. They may write the request body or headers, or start
	// to write out a response.
	pipe := r.pipeDef.CreatePipe(string(r.id))
	pipe.RequestHandlerFunc()(resp, req)

	// It's possible that not everything was cleaned up here.
	if r.proxying {
		r.flush()
	} else {
		r.resp.flush(http.StatusOK)
	}

	// This signals that everything is done.
	r.cmds <- command{id: DONE}
}

func readAndSend(handler commandHandler, body io.ReadCloser) {
	defer body.Close()
	buf := make([]byte, bodyBufSize)
	len, _ := body.Read(buf)
	for len > 0 {
		sendBodyChunk(handler, buf[:len])
		len, _ = body.Read(buf)
	}
}

func sendBodyChunk(handler commandHandler, chunk []byte) {
	if len(chunk) == 0 {
		return
	}

	chunkID := allocateChunk(chunk)

	cmd := command{
		id:  WBOD,
		msg: fmt.Sprintf("%x", chunkID),
	}
	handler.Commands() <- cmd
}

func allocateChunk(chunk []byte) int32 {
	chunkLen := uint32(len(chunk))
	chunkPtr := C.malloc(C.size_t(chunkLen))
	copy((*[1 << 30]byte)(chunkPtr)[:], chunk[:])
	chunkID := GoStoreChunk(chunkPtr, chunkLen)
	return chunkID
}

func (r *request) flush() {
	if r.origURL.String() != r.req.URL.String() {
		uriCmd := command{
			id:  WURI,
			msg: r.req.URL.String(),
		}
		r.cmds <- uriCmd
	}
	if !reflect.DeepEqual(r.origHeaders, r.req.Header) {
		hdrCmd := command{
			id:  WHDR,
			msg: serializeHeaders(r.req.Header),
		}
		r.cmds <- hdrCmd
	}
	if r.req.Body != r.origBody {
		readAndSend(r, r.req.Body)
	}
}

func copyHeaders(hdr http.Header) http.Header {
	newHeaders := http.Header{}
	for k, v := range hdr {
		newVal := make([]string, len(v))
		copy(newVal, v)
		newHeaders[k] = newVal
	}
	return newHeaders
}
