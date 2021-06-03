package GoExpress

import (
	"os"
	"net"
	"strconv"
	"crypto/tls"
	"github.com/joomcode/errorx"
	pathToRegexp "github.com/soongo/path-to-regexp"
	"github.com/borzhchevskiy/go-express/internal/static"
)

var (
	ServerErr = errorx.NewNamespace("server")
)

type Server struct {
	Host         string
	Port         int
	Socket       net.Listener
	STATIC       map[string]string
	Middleware   []func(req *Request, res *Response)
	GET          [][]interface{}
	POST         [][]interface{}
}

// Express(Host, Port) returns a Server object
func Express(host string, port int) *Server {
	s := &Server {
		Host:       host,
		Port:       port,
		STATIC:     make(map[string]string),
		Middleware: make([]func(req *Request, res *Response), 0),
		GET:        make([][]interface{}, 0),
		POST:       make([][]interface{}, 0),
	}
	return s
}

// Server.Use(Middleware) appends given middleware to server
func (s *Server) Use(middleware func(req *Request, res *Response)) {
	s.Middleware = append(s.Middleware, middleware)
}

// Server.Listen() listens for connections
func (s *Server) Listen() error {
	var err error
	s.Socket, err = net.Listen("tcp4", s.Host + ":" + strconv.Itoa(s.Port))
	if err != nil {
		return errorx.Decorate(err, "failed to start server")
		os.Exit(1)
	}
	for {
		c, _ := s.Socket.Accept()
		go s.serveClient(c)
	}
	s.Socket.Close()
	return nil
}

// Server.ListenTLS(CERTIFICATE, KEY) listens for connections, and process it with tls
func (s *Server) ListenTLS(certificate string, key string) error {
	cert, err := tls.LoadX509KeyPair(certificate, key)
	if err != nil {
		return errorx.Decorate(err, "failed to load tls keys")
		os.Exit(1)
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}}

	sock, err := tls.Listen("tcp4", s.Host + ":" + strconv.Itoa(s.Port), config)
	if err != nil {
		return errorx.Decorate(err, "failed to start server")
		os.Exit(1)
	}

	for {
		c, err := sock.Accept()
		if err != nil {
			continue
		}
		go s.serveClient(c)
	}
	sock.Close()
	return nil
}

// Server.Static(PATH, REAL_PATH) serves static files
func (s *Server) Static(path string, real_path string) {
	if path[len(path)-1] != []byte("/")[0] {
		path += "/"
	}
	s.STATIC[path] = real_path
}

// Server.Get(PATH, HANDLER) appends given handler to Get routes
func (s *Server) Get(path string, handler func(req *Request, res *Response)) {
	if path[len(path)-1] != []byte("/")[0] {
		path += "/"
	}
	match := pathToRegexp.MustMatch(path, &pathToRegexp.Options{Decode: func(str string, token interface{}) (string, error) {
		return pathToRegexp.DecodeURIComponent(str)
	}})
	s.GET = append(s.GET, []interface{}{match, handler})
}

// Server.Post(PATH, HANDLER) appends given handler to Post routes
func (s *Server) Post(path string, handler func(req *Request, res *Response)) {
	if path[len(path)-1] != []byte("/")[0] {
		path += "/"
	}
	match := pathToRegexp.MustMatch(path, &pathToRegexp.Options{Decode: func(str string, token interface{}) (string, error) {
		return pathToRegexp.DecodeURIComponent(str)
	}})
	s.POST = append(s.POST, []interface{}{match, handler})
}

// Server.serveClient(CONN) its a private method that process request in goroutine
func (s *Server) serveClient(c net.Conn) {
	for {
		buf := make([]byte, 1024)
		c.Read(buf)
		req, closed, err := getRequest(string(buf))
		res := getResponse(c)
		if err == true {
			res.Error(res.BadRequest("Cannot Proceed " + req.Path + "\nBad Request"))
			return
		}
		if closed {
			res.Header("Connection", "closed")
			s.processRequest(closed, c, req, res)
			putRequest(req)
			putResponse(res)
			break
		} else {
			res.Header("Connection", "keep-alive")
			s.processRequest(closed, c, req, res)
			continue
		}
	}
}

func (s *Server) processRequest(closed bool, c net.Conn, req *Request, res *Response) error {
	switch req.Type {
	case "GET":
		Static, filePath := static.ProcessStatic(s.STATIC, req.Path)
		if Static {
			err := res.SendFile(filePath)
			if err != nil {
				return errorx.Decorate(err, "failed to send static file")
			} else {
				return nil
			}
		}
		for _, v := range s.GET {
			if match, _ := v[0].(func(string)(*pathToRegexp.MatchResult, error))(req.Path); match != nil {
				req.Params = match.Params
				s.callMiddleware(req, res)
				v[1].(func(req *Request, res *Response))(req, res)
				if closed {
					c.Close()
					return nil
				}
			}
		}
		res.Error(res.NotFound("Cannot Proceed " + req.Path + "\nNot Found"))
	}
	return nil
}

func (s *Server) callMiddleware(req *Request, res *Response) (*Request, *Response) {
	for _, v := range s.Middleware {
		v(req, res)
	}
	return req, res
}