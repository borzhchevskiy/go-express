package GoExpress

import (
	"os"
	"net"
	"time"
	"strconv"
	"strings"
	"sync"
	"bytes"
	"github.com/gabriel-vasile/mimetype"
	"github.com/borzhchevskiy/go-express/internal/status"
)

type Response struct {
	Proto      string
	Statuscode int
	Statusmsg  string
	Headers    map[string]string
	body       string
	socket     net.Conn
	status.Status
}

var responsePool = sync.Pool {
	New: func() interface{} {
		return new(Response)
	},
}

// newResponse(CONN) creates a basic Response objects and returns it
func getResponse(conn net.Conn) *Response {
	res := responsePool.Get().(*Response)
	res.Headers = make(map[string]string)
	res.socket = conn
	return res
}

func putResponse(response *Response) {
	responsePool.Put(response)
}

// Response.toString() its a private method that returns a string, send it client
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

// Response.Header(KEY, VALUE) sets header with given name and value
func (res *Response) Header(key string, value string) {
	res.Headers[key] = value
}

// Response.SetCookie(COOKIE) sets cookie, it take data from cookie object
func (res *Response) SetCookie(c *Cookie) {
	if c.MaxAge == "" {
		c.MaxAge = "86400"
	}
	res.Header("Set-Cookie", c.String())
}

// Response.DelCookie(NAME) immediately deletes cookie
func (res *Response) DelCookie(name string) {
	res.Header("Set-Cookie", name + "=0; Max-Age=0")
}

// Response.Error(NAME, MESSAGE) sends response with error to client
func (res *Response) Error(status [3]string) {
	res.Proto = "HTTP/1.1"
	res.Statuscode, _ = strconv.Atoi(status[0])
	res.Statusmsg = status[1]
	res.body = status[2]
	res.Header("Server", "GoExpress")
	res.Header("Date", time.Now().In(time.FixedZone("GMT", 0)).Format(time.RFC1123))
	res.Header("Content-Type", mimetype.Detect([]byte(res.body)).String())
	res.Header("Content-Length", strconv.Itoa(len([]byte(res.body))))
	res.Header("Connection", "close")
	res.socket.Write([]byte(res.toString()))
	res.socket.Close()
}

// Response.Send(DATA) sends data to client
func (res *Response) Send(body string) {
	res.Proto = "HTTP/1.1"
	if res.Statuscode == 0 {
		res.Statuscode = 200
	}
	if res.Statusmsg == "" {
		res.Statusmsg = "OK"
	}
	res.body = body
	res.Header("Server", "GoExpress")
	res.Header("Date", time.Now().In(time.FixedZone("GMT", 0)).Format(time.RFC1123))
	res.Header("Content-Type", mimetype.Detect([]byte(res.body)).String())
	res.Header("Content-Length", strconv.Itoa(len([]byte(res.body))))
	res.socket.Write([]byte(res.toString()))
}

// Response.SendFile(FILE_NAME) sends file to client
func (res *Response) SendFile(path string) error {
	res.Proto = "HTTP/1.1"
	if res.Statuscode == 0 {
		res.Statuscode = 200
	}
	if res.Statusmsg == "" {
		res.Statusmsg = "OK"
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	var body bytes.Buffer
	body.ReadFrom(file)
	res.body = body.String()
	res.Header("Server", "GoExpress")
	res.Header("Date", time.Now().In(time.FixedZone("GMT", 0)).Format(time.RFC1123))
	res.Header("Content-Type", mimetype.Detect([]byte(res.body)).String())
	res.Header("Content-Length", strconv.Itoa(len([]byte(res.body))))
	res.socket.Write([]byte(res.toString()))
	return nil
}