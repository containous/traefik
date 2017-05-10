package middlewares

import (
	"io"
	"net"
	"net/http"
	"os"

	"github.com/vulcand/oxy/utils"
)

//ErrorPages to hold the custom error page.  Can be extended later for multiple error pages if need be
type ErrorPages struct {
	ErrorPage string
}

//NewErrorPagesHandler initializes the utils.ErrorHandler for the custom error page
func NewErrorPagesHandler(errorPage string) utils.ErrorHandler {
	if _, err := os.Stat(errorPage); err == nil {
		return &ErrorPages{errorPage}
	}
	return &ErrorPages{}
}

func (ep *ErrorPages) ServeHTTP(w http.ResponseWriter, req *http.Request, err error) {
	statusCode := http.StatusInternalServerError
	if e, ok := err.(net.Error); ok {
		if e.Timeout() {
			statusCode = http.StatusGatewayTimeout
		} else {
			statusCode = http.StatusBadGateway
		}
	} else if err == io.EOF {
		statusCode = http.StatusBadGateway
	}
	w.WriteHeader(statusCode)
	if statusCode >= 500 && statusCode < 600 && ep.ErrorPage != "" {
		http.ServeFile(w, req, ep.ErrorPage)
	} else {
		w.Write([]byte(http.StatusText(statusCode)))
	}
}
