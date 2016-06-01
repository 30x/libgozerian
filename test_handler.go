package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type TestPipeDef struct {}
func (self *TestPipeDef) CreatePipe(reqId string) Pipe {
	return &TestPipe{}
}

type TestPipe struct {}
func (self *TestPipe) RequestHandlerFunc() http.HandlerFunc {
	return testHandleRequest
}
func (self *TestPipe) ResponseHandlerFunc() ResponseHandlerFunc {
	return testHandleResponse
}

// help us a bit by saving test results for internal comparison
var lastTestBody []byte

func testHandleRequest(resp http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/pass":
		// Nothing to do

	case "/slowpass":
		time.Sleep(time.Second)

	case "/readbody":
		buf, err := ioutil.ReadAll(req.Body)
		if err != nil {
			fmt.Printf("Error reading body: %v\n", err)
		}
		lastTestBody = buf
		req.Body.Close()

	case "/readbodyslow":
		tmp := make([]byte, 2)
		buf := &bytes.Buffer{}
		len, _ := req.Body.Read(tmp)
		for len > 0 {
			buf.Write(tmp[0:len])
			len, _ = req.Body.Read(tmp)
		}
		lastTestBody = buf.Bytes()
		req.Body.Close()

	case "/readanddiscard":
		tmp := make([]byte, 2)
		req.Body.Read(tmp)
		req.Body.Close()

	case "/replacebody":
		req.Body = ioutil.NopCloser(bytes.NewReader([]byte("Hello! I am the server!")))

	case "/writeheaders":
		req.Header.Add("Server", "Go Test Stuff")
		req.Header.Add("X-Apigee-Test", "HeaderTest")

	case "/writepath":
		newURL, _ := url.Parse("/newpath")
		req.URL = newURL

	case "/return201":
		resp.WriteHeader(http.StatusCreated)

	case "/returnheaders":
		resp.Header().Add("X-Apigee-Test", "Return Header Test")
		resp.WriteHeader(http.StatusOK)

	case "/returnbody":
		resp.Write([]byte("Hello! I am the server!"))

	case "/completerequest":
		newURL, _ := url.Parse("/totallynewurl")
		req.URL = newURL
		req.Header.Add("X-Apigee-Test", "Complete")
		// TODO would like reader to return in two chunks
		req.Body = ioutil.NopCloser(
			bytes.NewReader([]byte("Hello Again! Time for a complete rewrite!")))
		//ctx.ProxyRequest().Write([]byte("Hello Again! "))
		//ctx.ProxyRequest().Write([]byte("Time for a complete rewrite!"))

	case "/completeresponse":
		ioutil.ReadAll(req.Body)
		req.Body.Close()
		resp.Header().Add("X-Apigee-Test", "Complete")
		resp.WriteHeader(http.StatusCreated)
		resp.Write([]byte("Hello Again! "))
		resp.Write([]byte("Time for a complete rewrite!"))

	case "/writeresponseheaders":
	case "/transformbody":
	case "/transformbodychunks":
	case "/responseerror":
	case "/responseerror2":

	default:
		resp.WriteHeader(http.StatusNotFound)
	}
}

func testHandleResponse(w http.ResponseWriter, req *http.Request, resp *http.Response) {
	switch req.URL.Path {
	case "/writeresponseheaders":
		resp.Header.Set("X-Apigee-ResponseHeader", "yes")

	case "/transformbody":
		resp.Body = ioutil.NopCloser(
			bytes.NewReader([]byte("We have transformed the response!")))

	case "/responseerror":
		resp.StatusCode = http.StatusInternalServerError
		resp.Body = ioutil.NopCloser(
			bytes.NewReader([]byte("Error in the server!")))

	case "/responseerror2":
		w.Header().Set("X-Apigee-Response", "error")
		w.WriteHeader(http.StatusGatewayTimeout)
		w.Write([]byte("Response Error"))

	case "/transformbodychunks":
		resp.Header.Set("X-Apigee-Transformed", "yes")
		defer resp.Body.Close()

		buf := &bytes.Buffer{}
		rb := make([]byte, 128)
		len, _ := resp.Body.Read(rb)
		for len > 0 {
			buf.WriteString("{")
			buf.Write(rb[:len])
			buf.WriteString("}")
			len, _ = resp.Body.Read(rb)
		}
		resp.Body = ioutil.NopCloser(buf)

		resp.Header.Set("X-Apigee-Invisible", "yes")
	}
}
