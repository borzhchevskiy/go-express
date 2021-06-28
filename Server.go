package GoExpress

import (
	"os"
	"net"
	"strconv"
	"crypto/tls"
	"github.com/joomcode/errorx"
	pathToRegexp "github.com/soongo/path-to-regexp"
	"github.com/borzhchevskiy/go-express/internal/static"
	hmap "github.com/cornelk/hashmap"
)

// Type to configure the server
type Config struct {
	Host  string
	Port  int
	Cache bool
	CacheMaxAge int
}

// Server type
type Server struct {
	Host         string
	Port         int
	Socket       net.Listener
	STATIC       map[string]string
	Middleware   []func(req *Request, res *Response)
	GET          [][]interface{}
	POST         [][]interface{}
	FileCache    *hmap.HashMap
	Config       *Config
}

// Express(cfg *Config) (*Server) returns a Server object
func Express(cfg *Config) *Server {
	s := &Server {
		Host:       cfg.Host,
		Port:       cfg.Port,
		STATIC:     make(map[string]string),
		Middleware: make([]func(req *Request, res *Response), 0),
		GET:        make([][]interface{}, 0),
		POST:       make([][]interface{}, 0),
		FileCache:  &hmap.HashMap{},
		Config: cfg,
	}
	return s
}

// Server.Use(middleware func(req *Request, res *Response)) appends given middleware to server
func (s *Server) Use(middleware func(req *Request, res *Response)) {
	s.Middleware = append(s.Middleware, middleware)
}

// Server.Listen() (error) listens for connections
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

// Server.ListenTLS(certificate string, key string) listens for connections, and process it with tls
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

// Server.Static(path string, real_path string) serves static files
func (s *Server) Static(path string, real_path string) {
	if path[len(path)-1] == []byte("/")[0] {
		path = string(path[:len(path)-1])
	}
	if real_path[len(real_path)-1] == []byte("/")[0] {
		real_path = string(real_path[:len(real_path)-1])
	}
	s.STATIC[path] = real_path
}

// Server.Get(path string, handler func(req *Request, res *Response)) appends given handler to GET routes
func (s *Server) Get(path string, handler func(req *Request, res *Response)) {
	match := pathToRegexp.MustMatch(path, &pathToRegexp.Options{Decode: func(str string, token interface{}) (string, error) {
		return pathToRegexp.DecodeURIComponent(str)
	}})
	s.GET = append(s.GET, []interface{}{match, handler})
}

// Server.Post(path string, handler func(req *Request, res *Response)) appends given handler to POST routes
func (s *Server) Post(path string, handler func(req *Request, res *Response)) {
	match := pathToRegexp.MustMatch(path, &pathToRegexp.Options{Decode: func(str string, token interface{}) (string, error) {
		return pathToRegexp.DecodeURIComponent(str)
	}})
	s.POST = append(s.POST, []interface{}{match, handler})
}

// Server.serveClient(c net.Conn) processes request in goroutine
func (s *Server) serveClient(c net.Conn) {
	for {
		buf := make([]byte, 1024)
		c.Read(buf)
		req, closed, err := getRequest(string(buf))
		res := getResponse(c, s)
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

// Server.processRequest(closed bool, c net.Conn, req *Request, res *Response) (error) MAGIC IS DONE HERE
func (s *Server) processRequest(closed bool, c net.Conn, req *Request, res *Response) error {
	switch req.Type {
	case "GET":
		Static, filePath := static.ProcessStatic(s.STATIC, req.Path)
		if Static {
			err := res.SendFile(filePath)
			if err != nil {
				res.Error(res.NotFound("Cannot Proceed " + req.Path + "\nFile Not Found"))
				return errorx.Decorate(err, "failed to send static file")
			} else {
				return nil
			}
		}
		var Match *pathToRegexp.MatchResult
		var found bool
		for _, v := range s.GET {
			Match, _ = v[0].(func(string)(*pathToRegexp.MatchResult, error))(req.Path)
			if Match != nil {
				found = true
				req.Params = Match.Params
				s.callMiddleware(req, res)
				v[1].(func(req *Request, res *Response))(req, res)
				if closed {
					c.Close()
					return nil
				} else {
					continue
				}
			}
		}
		if !found {
			res.Error(res.NotFound("Cannot Proceed " + req.Path + "\nNot Found"))
		}
	}
	return nil
}

// Server.callMiddleware(req *Request, res *Response) (*Request, *Response) calls each middleware
func (s *Server) callMiddleware(req *Request, res *Response) (*Request, *Response) {
	for _, v := range s.Middleware {
		v(req, res)
	}
	return req, res
}