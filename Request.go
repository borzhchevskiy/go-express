package GoExpress

import (
	"sync"
	"strings"
)

type Request struct {
	Type       string           
	Path       string           
	Proto      string           
	Headers    map[string]string
	Body       map[string]string
	Params     map[interface{}]interface{}
}

var requestPool = sync.Pool {
	New: func() interface{} {
		return new(Request)
	},
}

// getRequest(RAW_REQUEST) parses type, path, protocol, headers from RAW_REQUEST and returns Request object
func getRequest(request string) (req *Request, closed bool, err bool) {
	req = requestPool.Get().(*Request)
	splitted_req := strings.Split(request, "\r\n")
	headers := make(map[string]string)
	req_first_ln := strings.Split(splitted_req[0], " ")
	if len(req_first_ln) < 3 {
		err = true
		return
	}
	req.Type = req_first_ln[0]
	var body_pos int
	for i, raw := range splitted_req[1:] {
		if i < len(splitted_req[1:])-2 {
			if req.Type == "POST" {
				body_pos = i
			}
			headers[strings.Split(raw, ":")[0]] = strings.TrimSpace(strings.Split(raw, ":")[1])
		}
	}
	path := req_first_ln[1]
	if path[len(path)-1] != []byte("/")[0] {
		path += "/"
	}
	req.Path = path
	req.Proto = strings.Split(splitted_req[0], " ")[2]
	req.Headers = headers
	if len(strings.Split(req.Path, "/")) != 0 && len(strings.Split(req.Path, "/")) != 2 {
		splitted_path := strings.Split(req.Path, "/")
		splitted_path = splitted_path[1 : len(splitted_path)-1]
	}
	if req.Type == "POST" {
		body := make(map[string]string)
		for _, v := range strings.Split(splitted_req[body_pos+3:body_pos+4][0], "&") {
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