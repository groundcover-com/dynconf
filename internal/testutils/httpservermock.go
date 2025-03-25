package testutils

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
)

type HTTPServerMock struct {
	response         atomic.Pointer[string]
	requestsReceived atomic.Uint64
}

func NewHTTPServerMock() *HTTPServerMock {
	return &HTTPServerMock{}
}

func (server *HTTPServerMock) SetResponse(response string) {
	respCopy := strings.Clone(response)
	server.response.Store(&respCopy)
}

func (server *HTTPServerMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	server.requestsReceived.Add(1)
	response := *(server.response.Load())
	fmt.Fprintln(w, response)
}
