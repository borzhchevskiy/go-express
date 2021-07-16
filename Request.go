package balda

import (
	"errors"
	"strings"
	"sync"
)

// Request type
type Request struct {
	Type    string
	Path    string
	Proto   string
	Headers map[string]string
	Body    map[string]string
	Params  map[interface{}]interface{}
	Static  bool
}

var requestPool = sync.Pool{
	New: func() interface{} {
		return new(Request)
	},
}

// getRequest(request string) (req Request, closed bool, err bool) parses type, path, protocol, headers from RAW_REQUEST and returns Request object
func getRequest(request string) (req *Request, closed bool, err error) {
	req = requestPool.Get().(*Request)
	splittedReq := strings.Split(request, "\r\n")
	headers := make(map[string]string)
	reqFirstLn := strings.Split(splittedReq[0], " ")
	if len(reqFirstLn) < 3 {
		err = errors.New("reqFirstLn < 3")
		return
	}
	req.Type = reqFirstLn[0]
	var bodyPos int
	for i, raw := range splittedReq[1:] {
		if i < len(splittedReq[1:])-2 {
			if req.Type == "POST" {
				bodyPos = i
			}
			headers[strings.Split(raw, ":")[0]] = strings.TrimSpace(strings.Split(raw, ":")[1])
		}
	}
	path := reqFirstLn[1]
	if path[len(path)-1] != []byte("/")[0] {
		path += "/"
	}
	req.Path = path
	req.Proto = strings.Split(splittedReq[0], " ")[2]
	req.Headers = headers
	if req.Type == "POST" {
		body := make(map[string]string)
		for _, v := range strings.Split(splittedReq[bodyPos+3 : bodyPos+4][0], "&") {
			body[strings.Split(v, "=")[0]] = strings.Split(v, "=")[1]
		}
		req.Body = body
	}

	if req.Headers["Connection"] == "close" {
		closed = true
	} else {
		closed = false
	}
	return
}

func putRequest(request *Request) {
	requestPool.Put(request)
}
