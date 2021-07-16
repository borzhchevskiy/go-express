package balda

import (
	"bytes"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/borzhchevskiy/balda/internal/status"
	hmap "github.com/cornelk/hashmap"
)

// Response type
type Response struct {
	Proto      string
	Statuscode int
	Statusmsg  string
	Headers    map[string]string
	body       string
	socket     net.Conn
	status.Status
	server *Server
}

var responsePool = sync.Pool{
	New: func() interface{} {
		return new(Response)
	},
}

func getResponse(conn net.Conn, s *Server) *Response {
	res := responsePool.Get().(*Response)
	res.server = s
	res.Headers = make(map[string]string)
	res.socket = conn
	return res
}

func putResponse(response *Response) {
	responsePool.Put(response)
}

func (res *Response) toString() string {
	var headers strings.Builder
	for k, v := range res.Headers {
		headers.WriteString(k)
		headers.WriteRune(':')
		headers.WriteString(v)
		headers.WriteString("\r\n")
	}
	response := res.Proto + " " + strconv.Itoa(res.Statuscode) + " " + res.Statusmsg + "\r\n" + headers.String() + "\r\n" + res.body
	return response
}

// Header (key string, value string) sets header with given name and value
func (res *Response) Header(key string, value string) {
	res.Headers[key] = value
}

// SetCookie (c *Cookie) sets cookie, it takes data from cookie object
func (res *Response) SetCookie(c Cookie) {
	if c.MaxAge == "" {
		c.MaxAge = "86400"
	}
	res.Header("Set-Cookie", c.String())
}

// DelCookie (name string) immediately deletes cookie
func (res *Response) DelCookie(name string) {
	res.Header("Set-Cookie", name+"=0; Max-Age=0")
}

// Redirect (to string) redirect user to given path
func (res *Response) Redirect(to string) {
	res.Statuscode = 301
	res.Statusmsg = "Moved Permanently"
	res.Header("Location", to)
}

// Error (status []string) sends response with error to client
func (res *Response) Error(status [3]string) {
	res.Proto = "HTTP/1.1"
	res.Statuscode, _ = strconv.Atoi(status[0])
	res.Statusmsg = status[1]
	res.body = status[2]
	res.Header("Server", "Balda-Golang")
	res.Header("Date", time.Now().In(time.FixedZone("GMT", 0)).Format(time.RFC1123))
	res.Header("Content-Type", http.DetectContentType([]byte(res.body)))
	res.Header("Content-Length", strconv.Itoa(len([]byte(res.body))))
	res.Header("Connection", "close")
	res.socket.Write([]byte(res.toString()))
	res.socket.Close()
}

// Send (body string) sends data to client
func (res *Response) Send(body string) {
	res.Proto = "HTTP/1.1"
	if res.Statuscode == 0 {
		res.Statuscode = 200
	}
	if res.Statusmsg == "" {
		res.Statusmsg = "OK"
	}
	res.body = body
	res.Header("Server", "Balda-Golang")
	res.Header("Date", time.Now().In(time.FixedZone("GMT", 0)).Format(time.RFC1123))
	res.Header("Content-Type", http.DetectContentType([]byte(res.body)))
	res.Header("Content-Length", strconv.Itoa(len([]byte(res.body))))
	res.socket.Write([]byte(res.toString()))
}

// SendFile (path string) (error) sends file to client
func (res *Response) SendFile(path string) error {
	res.Proto = "HTTP/1.1"
	if res.Statuscode == 0 {
		res.Statuscode = 200
	}
	if res.Statusmsg == "" {
		res.Statusmsg = "OK"
	}
	data, ok := res.server.FileCache.Get(path)
	var body bytes.Buffer
	if ok {
		body = *bytes.NewBuffer(data.([]byte))
	} else {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		body.ReadFrom(file)
		res.server.FileCache.Set(path, body.Bytes())
		go func() {
			for {
				if res.server.Config.CacheMaxAge == 0 {
					return
				}
				time.Sleep(time.Duration(res.server.Config.CacheMaxAge) * time.Second)
				res.server.FileCache = hmap.HashMap{}
			}
		}()
	}

	res.body = body.String()
	res.Header("Server", "Balda-Golang")
	res.Header("Date", time.Now().In(time.FixedZone("GMT", 0)).Format(time.RFC1123))
	res.Header("Content-Type", http.DetectContentType([]byte(res.body)))
	res.Header("Content-Length", strconv.Itoa(len([]byte(res.body))))
	res.socket.Write([]byte(res.toString()))
	return nil
}
