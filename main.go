package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"unsafe"
)

/*
#include <stdlib.h>
*/
import "C"

const (
	defaultHandlerID = "default"
)

var defaultHandlerName = C.CString(defaultHandlerID)

/*
 * The weaver project is designed to build a shared library, not a "main."
 * However, for testing purposes we can start it up as an executable which
 * listens on an HTTP port.
 * In this mode, it can either act in echo mode by returning what is sent,
 * or it can act in proxy mode.
 */

type gozerianServer struct {
	listener *net.TCPListener
	target   string
	debug    bool
}

/*
 * Start the server listening on the specified HTTP port. If "proxyTarget"
 * is empty, then echo back all requests. Otherwise, proxy to that URL.
 * If "testHandler" is true, install a test handler for unit test purposes.
 * If "port" is 0, then listen on an ephemeral port.
 */
func startGozerianServer(port int, proxyTarget, handlerURL string) (*gozerianServer, error) {
	cURL := C.CString(handlerURL)
	defer C.free(unsafe.Pointer(cURL))

	errStr := GoCreateHandler(defaultHandlerName, cURL)
	if errStr != nil {
		defer C.free(errStr)
		return nil, errors.New(C.GoString(errStr))
	}

	addr := net.TCPAddr{
		Port: port,
	}
	listener, err := net.ListenTCP("tcp", &addr)
	if err != nil {
		return nil, err
	}

	svr := gozerianServer{
		listener: listener,
		target:   proxyTarget,
	}

	return &svr, nil
}

func (s *gozerianServer) run() {
	handler := weaverHandler{
		target: s.target,
		debug:  s.debug,
	}
	http.Serve(s.listener, &handler)
}

func (s *gozerianServer) stop() {
	s.listener.Close()
	GoDestroyHandler(defaultHandlerName)
}

func (s *gozerianServer) setDebug(d bool) {
	s.debug = d
}

func (s *gozerianServer) getPort() int {
	_, port, err := net.SplitHostPort(s.listener.Addr().String())
	if err != nil {
		return 0
	}
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return 0
	}
	return portNum
}

type weaverHandler struct {
	target string
	debug  bool
}

func (m *weaverHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	// Although we have nice Go ways to call all these internal functions,
	// use the public C API so that we can get good test coverage.
	id := GoCreateRequest(defaultHandlerName)
	defer GoFreeRequest(id)
	rid := GoCreateResponse(defaultHandlerName)
	defer GoFreeResponse(rid)

	requestBody := &bytes.Buffer{}
	done := m.processRequest(resp, req, id, rid, requestBody)
	if !done {
		m.processResponse(resp, req, id, rid, requestBody)
	}
}

func (m *weaverHandler) processRequest(
	resp http.ResponseWriter, req *http.Request,
	id, rid uint32, requestBody *bytes.Buffer) bool {

	reqHdrs := &bytes.Buffer{}
	fmt.Fprintf(reqHdrs, "%s %s HTTP/1.1\r\n", req.Method, req.URL.Path)
	req.Header.Write(reqHdrs)

	cReqHdrs := C.CString(reqHdrs.String())
	defer C.free(unsafe.Pointer(cReqHdrs))
	GoBeginRequest(id, cReqHdrs)

	var cmd string
	proxying := true
	writingRequest := false
	responseCode := http.StatusOK
	proxyHeaders := req.Header
	//proxyPath := req.URL.Path
	sentHeaders := false

	for cmd != cmdDone && cmd != cmdErrr {
		rawCmd := GoPollRequest(id, 1)
		cmdBuf := C.GoString(rawCmd)
		C.free(unsafe.Pointer(rawCmd))
		cmd = cmdBuf[:4]
		msg := cmdBuf[4:]

		if m.debug {
			fmt.Printf("Command: \"%s\"\n", cmd)
		}

		switch cmd {
		case cmdErrr:
			resp.WriteHeader(http.StatusInternalServerError)
			resp.Write([]byte(msg))
			return true
		case cmdRbod:
			requestBody.ReadFrom(req.Body)
			ptr, len := sliceToPtr(requestBody.Bytes())
			GoSendRequestBodyChunk(id, 1, ptr, len)
			C.free(ptr)
		case cmdWhdr:
			if proxying {
				parseHeaders(proxyHeaders, msg)
			} else {
				parseHeaders(resp.Header(), msg)
			}
		case cmdWURI:
			//proxyPath = msg
		case cmdWbod:
			chunk := getChunkData(msg)
			if proxying {
				if !writingRequest {
					requestBody.Reset()
					writingRequest = true
				}
				requestBody.Write(chunk)
			} else {
				if !sentHeaders {
					resp.WriteHeader(responseCode)
					sentHeaders = true
				}
				resp.Write(chunk)
			}
		case cmdSwch:
			proxying = false
			responseCode, _ = strconv.Atoi(msg)
		case cmdDone:
		default:
			sendHTTPError(fmt.Errorf("Unexpected command %s", cmd), resp)
			return true
		}
	}

	if !proxying {
		// Request path decided immediately to send a response
		if !sentHeaders {
			resp.WriteHeader(responseCode)
		}
		return true
	}

	if requestBody.Len() == 0 {
		requestBody.ReadFrom(req.Body)
	}
	return false
}

func (m *weaverHandler) processResponse(
	resp http.ResponseWriter, req *http.Request,
	id, rid uint32, requestBody *bytes.Buffer) {

	// TODO in target proxy mode, actually get target headers
	respHdrs := &bytes.Buffer{}
	respHdrs.WriteString("Server: Weaver Test Main\r\n")

	cRespHdrs := C.CString(respHdrs.String())
	defer C.free(unsafe.Pointer(cRespHdrs))

	GoBeginResponse(rid, id, http.StatusOK, cRespHdrs)

	var cmd string
	responseCode := http.StatusOK
	sentHeaders := false
	wroteBody := false

	for cmd != cmdDone && cmd != cmdErrr {
		rawCmd := GoPollResponse(rid, 1)
		cmdBuf := C.GoString(rawCmd)
		C.free(unsafe.Pointer(rawCmd))
		cmd = cmdBuf[:4]
		msg := cmdBuf[4:]

		if m.debug {
			fmt.Printf("Command: \"%s\"\n", cmd)
		}

		switch cmd {
		case cmdErrr:
			resp.WriteHeader(http.StatusInternalServerError)
			resp.Write([]byte(msg))
			return
		case cmdWsta, cmdSwch:
			responseCode, _ = strconv.Atoi(msg)
		case cmdWhdr:
			parseHeaders(resp.Header(), msg)
		case cmdRbod:
			ptr, len := sliceToPtr(requestBody.Bytes())
			GoSendResponseBodyChunk(rid, 1, ptr, len)
			C.free(ptr)
		case cmdWbod:
			if !sentHeaders {
				resp.WriteHeader(responseCode)
				sentHeaders = true
			}
			chunk := getChunkData(msg)
			wroteBody = true
			resp.Write(chunk)
		case cmdDone:
		default:
			sendHTTPError(fmt.Errorf("Unexpected command %s", cmd), resp)
		}
	}

	if m.target == "" {
		// Pretend that we are a proxy for another server by echoing the request
		if !sentHeaders {
			resp.WriteHeader(http.StatusOK)
		}
		if !wroteBody {
			requestBody.WriteTo(resp)
		}
	} else {
		sendHTTPError(fmt.Errorf("Didn't implement proxying to target yet"), resp)
	}
}

func getChunkData(rawID string) []byte {
	id, err := strconv.ParseInt(rawID, 16, 32)
	if err != nil {
		return nil
	}
	return getChunkDataByID(int32(id))
}

func getChunkDataByID(id int32) []byte {
	ptr := GoGetChunk(id)
	len := GoGetChunkLength(id)
	buf := C.GoBytes(ptr, C.int(len))
	GoReleaseChunk(int32(id))
	C.free(ptr)
	return buf
}

func sendHTTPError(err error, resp http.ResponseWriter) {
	fmt.Printf("Error: %s\n", err.Error())
	resp.Header().Set("Content-Type", "text/plain")
	resp.WriteHeader(http.StatusInternalServerError)
	resp.Write([]byte(err.Error()))
}

func main() {
	var port int
	var target string
	var testHandler bool
	var handlerURI string

	flag.IntVar(&port, "p", 0, "(required) Port to listen on")
	flag.StringVar(&target, "u", "", "(optional) Target proxy URL")
	flag.StringVar(&handlerURI, "h", "", "(optional) URL of handler to create")
	flag.BoolVar(&testHandler, "t", false, "(optional) Install a set of test handlers")
	flag.Parse()

	if !flag.Parsed() {
		flag.PrintDefaults()
		os.Exit(2)
	}

	if testHandler {
		handlerURI = TestHandlerURI
	}

	server, err := startGozerianServer(port, target, handlerURI)
	if err != nil {
		fmt.Printf("Cannot start server: %s\n", err)
		os.Exit(3)
	}

	fmt.Printf("Listening on port %d\n", server.getPort())

	doneChan := make(chan bool, 1)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT)
	signal.Notify(signalChan, syscall.SIGTERM)

	go func() {
		<-signalChan
		doneChan <- true
	}()

	go server.run()

	<-doneChan
	server.stop()
}
