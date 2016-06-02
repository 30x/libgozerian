package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Remote HTTP Tests", func() {
	It("Pass GET", func() {
		resp, err := http.Get(fmt.Sprintf("%s/pass", testURL))
		Expect(err).Should(Succeed())
		Expect(resp.StatusCode).Should(Equal(200))
	})

	It("Pass slow GET", func() {
		resp, err := http.Get(fmt.Sprintf("%s/slowpass", testURL))
		Expect(err).Should(Succeed())
		Expect(resp.StatusCode).Should(Equal(200))
	})

	It("Return 404", func() {
		resp, err := http.Get(fmt.Sprintf("%s/notfound", testURL))
		Expect(err).Should(Succeed())
		Expect(resp.StatusCode).Should(Equal(404))
	})

	It("Pass echo back POST", func() {
		body := []byte("Hello, World!")
		bodyBuf := bytes.NewBuffer(body)
		resp, err :=
			http.Post(fmt.Sprintf("%s/pass", testURL),
				"text/plain", bodyBuf)
		Expect(err).Should(Succeed())
		Expect(resp.StatusCode).Should(Equal(200))

		defer resp.Body.Close()
		readBody, err := ioutil.ReadAll(resp.Body)
		Expect(err).Should(Succeed())
		Expect(bytes.Equal(body, readBody)).Should(BeTrue())
	})

	It("Discard body POST", func() {
		body := []byte("Hello, World!")
		bodyBuf := bytes.NewBuffer(body)
		resp, err :=
			http.Post(fmt.Sprintf("%s/readanddiscard", testURL),
				"text/plain", bodyBuf)
		Expect(err).Should(Succeed())
		Expect(resp.StatusCode).Should(Equal(200))

		defer resp.Body.Close()
		readBody, err := ioutil.ReadAll(resp.Body)
		Expect(err).Should(Succeed())
		Expect(bytes.Equal(body, readBody)).Should(BeTrue())
	})

	It("Replace body POST", func() {
		body := []byte("Hello, World!")
		bodyBuf := bytes.NewBuffer(body)
		resp, err :=
			http.Post(fmt.Sprintf("%s/replacebody", testURL),
				"text/plain", bodyBuf)
		Expect(err).Should(Succeed())
		Expect(resp.StatusCode).Should(Equal(200))

		defer resp.Body.Close()
		readBody, err := ioutil.ReadAll(resp.Body)
		Expect(err).Should(Succeed())

		expectedBody := []byte("Hello! I am the server!")
		fmt.Fprintf(GinkgoWriter, "Body: %s\n", string(readBody))
		Expect(bytes.Equal(expectedBody, readBody)).Should(BeTrue())
	})

	It("Write headers GET", func() {
		resp, err := http.Get(fmt.Sprintf("%s/writeheaders", testURL))
		Expect(err).Should(Succeed())
		defer resp.Body.Close()
		Expect(resp.StatusCode).Should(Equal(200))
		// TODO check for something!
	})

	It("Write path GET", func() {
		resp, err := http.Get(fmt.Sprintf("%s/writepath", testURL))
		Expect(err).Should(Succeed())
		defer resp.Body.Close()
		Expect(resp.StatusCode).Should(Equal(200))
		// TODO check for something!
	})

	It("Return 201 GET", func() {
		resp, err := http.Get(fmt.Sprintf("%s/return201", testURL))
		Expect(err).Should(Succeed())
		defer resp.Body.Close()
		Expect(resp.StatusCode).Should(Equal(201))
	})

	It("Return Headers GET", func() {
		resp, err := http.Get(fmt.Sprintf("%s/returnheaders", testURL))
		Expect(err).Should(Succeed())
		Expect(resp.StatusCode).Should(Equal(200))
		Expect(resp.Header.Get("X-Apigee-Test")).Should(Equal("Return Header Test"))
	})

	It("Return Body GET", func() {
		resp, err := http.Get(fmt.Sprintf("%s/returnbody", testURL))
		Expect(err).Should(Succeed())
		defer resp.Body.Close()
		Expect(resp.StatusCode).Should(Equal(200))
		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).Should(Succeed())
		expectedBody := []byte("Hello! I am the server!")
		Expect(bytes.Equal(expectedBody, body)).Should(BeTrue())
	})

	It("Return MessageID GET", func() {
		resp, err := http.Get(fmt.Sprintf("%s/replacewithid", testURL))
		Expect(err).Should(Succeed())
		defer resp.Body.Close()
		Expect(resp.StatusCode).Should(Equal(200))
		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).Should(Succeed())
		msgID := string(body)
		fmt.Fprintf(GinkgoWriter, "Message ID: \"%s\"\n", msgID)
		Expect(resp.Header.Get("X-Apigee-MsgID")).Should(Equal(msgID))
	})

	It("Complete request POST", func() {
		reqBody := []byte("Hello, World!")
		bodyBuf := bytes.NewBuffer(reqBody)
		resp, err :=
			http.Post(fmt.Sprintf("%s/completerequest", testURL),
				"text/plain", bodyBuf)
		Expect(err).Should(Succeed())
		defer resp.Body.Close()
		Expect(resp.StatusCode).Should(Equal(200))

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).Should(Succeed())
		expectedBody := []byte("Hello Again! Time for a complete rewrite!")
		Expect(bytes.Equal(expectedBody, body)).Should(BeTrue())
	})

	It("Complete response GET", func() {
		resp, err := http.Get(fmt.Sprintf("%s/completeresponse", testURL))
		Expect(err).Should(Succeed())
		defer resp.Body.Close()
		Expect(resp.StatusCode).Should(Equal(http.StatusCreated))
		Expect(resp.Header.Get("X-Apigee-Test")).Should(Equal("Complete"))

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).Should(Succeed())
		expectedBody := []byte("Hello Again! Time for a complete rewrite!")
		Expect(bytes.Equal(expectedBody, body)).Should(BeTrue())
	})

	It("Write response headers GET", func() {
		resp, err := http.Get(fmt.Sprintf("%s/writeresponseheaders", testURL))
		Expect(err).Should(Succeed())
		defer resp.Body.Close()
		Expect(resp.StatusCode).Should(Equal(200))
		Expect(resp.Header.Get("X-Apigee-ResponseHeader")).Should(Equal("yes"))
	})

	It("Write response body POST", func() {
		reqBody := []byte("Hello, World!")
		bodyBuf := bytes.NewBuffer(reqBody)
		resp, err :=
			http.Post(fmt.Sprintf("%s/transformbody", testURL),
				"text/plain", bodyBuf)
		Expect(err).Should(Succeed())
		defer resp.Body.Close()
		Expect(resp.StatusCode).Should(Equal(200))

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).Should(Succeed())
		expectedBody := []byte("We have transformed the response!")
		fmt.Fprintf(GinkgoWriter, "Transformed body: %s\n", string(body))
		Expect(bytes.Equal(expectedBody, body)).Should(BeTrue())
	})

	It("Write response status GET", func() {
		resp, err := http.Get(fmt.Sprintf("%s/responseerror", testURL))
		Expect(err).Should(Succeed())
		defer resp.Body.Close()
		Expect(resp.StatusCode).Should(Equal(http.StatusInternalServerError))

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).Should(Succeed())
		expectedBody := []byte("Error in the server!")
		fmt.Fprintf(GinkgoWriter, "Transformed body: %s\n", string(body))
		Expect(bytes.Equal(expectedBody, body)).Should(BeTrue())
	})

	It("Write response status GET 2", func() {
		resp, err := http.Get(fmt.Sprintf("%s/responseerror2", testURL))
		Expect(err).Should(Succeed())
		defer resp.Body.Close()
		Expect(resp.StatusCode).Should(Equal(http.StatusGatewayTimeout))
		Expect(resp.Header.Get("X-Apigee-Response")).Should(Equal("error"))

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).Should(Succeed())
		expectedBody := []byte("Response Error")
		fmt.Fprintf(GinkgoWriter, "Transformed body: %s\n", string(body))
		Expect(bytes.Equal(expectedBody, body)).Should(BeTrue())
	})

	It("Transform response body POST", func() {
		err := testResponseTransformation()
		Expect(err).Should(Succeed())
	})

	It("Transform response body in parallel", func() {
		testParallelTransformation(2, 10)
	})

	It("Transform response body in more parallel", func() {
		testParallelTransformation(100, 10)
	})
})

func testResponseTransformation() error {
	reqBody := []byte("Hello, World!")
	bodyBuf := bytes.NewBuffer(reqBody)
	resp, err :=
		http.Post(fmt.Sprintf("%s/transformbodychunks", testURL),
			"text/plain", bodyBuf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Invalid status code: %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Apigee-Transformed") != "yes" {
		return fmt.Errorf("X-Apigee-Transformed header not set")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(body) != "{Hello, World!}" {
		return fmt.Errorf("Body is invalid: \"%s\"", string(body))
	}
	return nil
}

func testParallelTransformation(concurrency, count int) {
	doneCh := make(chan error, concurrency)

	for c := 0; c < concurrency; c++ {
		go func() {
			var err error
			for i := 0; err != nil && i < count; i++ {
				err = testResponseTransformation()
			}
			doneCh <- err
		}()
	}

	for c := 0; c < concurrency; c++ {
		err := <-doneCh
		Expect(err).Should(Succeed())
	}
}
