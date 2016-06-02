package main

import (
	cryptoRand "crypto/rand"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/30x/gozerian/c_gateway"
	"github.com/30x/gozerian/pipeline"
)

/*
 * This is code that processes requests from C code. It takes in a request and returns
 * an ID, and then it has an API for that particular request.
 */

/*
 * The table of requests. It is global. For maximum flexibility we will put
 * a lock around it.
 */

var requests = make(map[uint32]*request)
var responses = make(map[uint32]*response)
var pipeDefs = make(map[string]pipeline.Definition)
var managerLatch = &sync.Mutex{}
var lastID uint32
var oneInit sync.Once

/*
 * Common interface for requests and responses
 */
type commandHandler interface {
	Commands() chan command
	Bodies() chan []byte
	Headers() http.Header
	ResponseWritten()
	StartRead()
}

/*
 * Initialize the library. Not necessary but useful in testing.
 */
func Initialize() {
	oneInit.Do(initRand)
}

/*
 * Create a new handler. It will be necessary in order to send a request.
 */
func CreateHandler(id, cfgURI string) error {
	Initialize()

	configURI, err := url.Parse(cfgURI)
	if err != nil {
		return err
	}

	var pipeDef pipeline.Definition
	if configURI.Scheme == URNScheme && configURI.Opaque == TestHandlerURIName {
		pipeDef = &TestPipeDef{}
	} else if configURI.Scheme == URNScheme && configURI.Opaque == BadHandlerURIName {
		// This is a pre-defined "bad handler" so that we can unit-test an error from this routine.
		return fmt.Errorf("Invalid handler from %s", cfgURI)
	} else {
		pipeDef, err = c_gateway.DefinePipe(configURI)
		if err != nil {
			return err
		}
	}

	managerLatch.Lock()
	pipeDefs[id] = pipeDef
	managerLatch.Unlock()
	return nil
}

/*
 * Destroy an existing handler.
 */
func DestroyHandler(id string) {
	managerLatch.Lock()
	delete(pipeDefs, id)
	managerLatch.Unlock()
}

/*
 * Create a new request object. It should be used once and only once.
 */
func CreateRequest(handlerID string) uint32 {
	managerLatch.Lock()
	defer managerLatch.Unlock()

	pd := pipeDefs[handlerID]
	if pd == nil {
		return 0
	}
	// After 2BB requests we will roll over. That should not be a problem.
	lastID++
	id := lastID
	req := newRequest(id, pd)
	requests[id] = req
	return id
}

/*
 * Create a new response object. It should be used once and only once.
 */
func CreateResponse(handlerID string) uint32 {
	managerLatch.Lock()
	defer managerLatch.Unlock()

	handler := pipeDefs[handlerID]
	if handler == nil {
		return 0
	}
	lastID++
	id := lastID
	r := newResponse(id, pipeDefs[handlerID])
	responses[id] = r
	return id
}

/*
 * Begin the request by sending in a set of headers.
 */
func BeginRequest(id uint32, rawHeaders string) error {
	req := getRequest(id)
	if req == nil {
		return fmt.Errorf("Unknown request: %d", id)
	}

	return req.begin(rawHeaders)
}

func BeginResponse(responseID, requestID, status uint32, rawHeaders string) error {
	r := getResponse(responseID)
	if r == nil {
		return fmt.Errorf("Unknown response: %d", responseID)
	}
	req := getRequest(requestID)
	if req == nil {
		return fmt.Errorf("Unknown request: %d", requestID)
	}

	return r.begin(status, rawHeaders, req)
}

/*
 * Get status of the request, without blocking. The result will be a single
 * string that represents a command, or an empty string if there is none.
 * Commands are defined in commands.go.
 */
func PollRequest(id uint32, block bool) string {
	req := getRequest(id)
	if req == nil {
		return "ERRRUnknown request"
	}

	if block {
		return req.poll()
	}
	return req.pollNB()
}

func PollResponse(id uint32, block bool) string {
	resp := getResponse(id)
	if resp == nil {
		return "ERRRUnknown response"
	}

	if block {
		return resp.poll()
	}
	return resp.pollNB()
}

/*
 * Free the slot for a request.
 */
func FreeRequest(id uint32) {
	managerLatch.Lock()
	delete(requests, id)
	managerLatch.Unlock()
}

func FreeResponse(id uint32) {
	managerLatch.Lock()
	delete(responses, id)
	managerLatch.Unlock()
}

/*
 * Send some data to act as the request body.
 */
func SendRequestBodyChunk(id uint32, last bool, chunk []byte) {
	req := getRequest(id)
	sendChunk(req, last, chunk)
}

func SendResponseBodyChunk(id uint32, last bool, chunk []byte) {
	resp := getResponse(id)
	sendChunk(resp, last, chunk)
}

/*
 * One-time seeding of the global random-number generator so that we can
 * quickly generate unique request IDs.
 * Use the crypto random number generator to do a good job of initializing
 * the faster one in the "math" package.
 */
func initRand() {
	maxRand := big.NewInt(math.MaxInt64)
	seed, err := cryptoRand.Int(cryptoRand.Reader, maxRand)
	if err != nil {
		panic(fmt.Sprintf("Error initializing random numbers: %s", err))
	}
	rand.Seed(seed.Int64())
}

func makeMessageID() string {
	// Make timestamp into milliseconds since Unix epoch
	ts := time.Now().UnixNano() / 1000000
	// Make a random segment too
	r := rand.Uint32()
	return strconv.FormatInt(ts, 16) + "." + strconv.FormatUint(uint64(r), 16)
}

func sendChunk(h commandHandler, last bool, chunk []byte) {
	if h == nil {
		return
	}
	if len(chunk) > 0 {
		h.Bodies() <- chunk
	}
	if last {
		close(h.Bodies())
	}
}

func getRequest(id uint32) *request {
	managerLatch.Lock()
	defer managerLatch.Unlock()
	return requests[id]
}

func getResponse(id uint32) *response {
	managerLatch.Lock()
	defer managerLatch.Unlock()
	return responses[id]
}
