package balda

import (
	"crypto/tls"
	"net"

	// "net/http"
	// _ "net/http/pprof"
	"strconv"
	"unsafe"

	hmap "github.com/cornelk/hashmap"
	"github.com/joomcode/errorx"
	pathToRegexp "github.com/soongo/path-to-regexp"
)

type handler func(req *Request, res *Response)

type staticRoute struct {
	Match func(string) (*pathToRegexp.MatchResult, error)
	Dir   string
}

type route struct {
	Match   func(string) (*pathToRegexp.MatchResult, error)
	Handler handler
}

// Config type
type Config struct {
	Host         string
	Port         int
	ReuseConn    bool
	MaxReuseConn int
	Cache        bool
	CacheMaxAge  int
}

// Server type
type Server struct {
	Host       string
	Port       int
	Socket     net.Listener
	STATIC     []staticRoute
	Middleware []handler
	GET        []route
	POST       []route
	FileCache  hmap.HashMap
	Config     Config
}

// Express (cfg *Config) (Server) returns a Server object
func Express(cfg Config) Server {
	return Server{
		Host:       cfg.Host,
		Port:       cfg.Port,
		STATIC:     make([]staticRoute, 0, 8),
		Middleware: make([]handler, 0, 8),
		GET:        make([]route, 0, 8),
		POST:       make([]route, 0, 8),
		FileCache:  hmap.HashMap{},
		Config:     cfg,
	}
}

// Use (middleware func(req *Request, res *Response)) appends given middleware to server
func (s *Server) Use(middleware handler) {
	s.Middleware = append(s.Middleware, middleware)
}

// Listen () (error) listens for connections
func (s *Server) Listen() error {
	// go func() {
	// 	http.ListenAndServe(":1234", nil)
	// }()
	var err error
	s.Socket, err = net.Listen("tcp4", s.Host+":"+strconv.Itoa(s.Port))
	if err != nil {
		return errorx.Decorate(err, "failed to start server")

	}
	for {
		c, _ := s.Socket.Accept()
		go s.serveClient(c, s.Config.MaxReuseConn)
	}

}

// ListenTLS (certificate string, key string) listens for connections, and process it with tls
func (s *Server) ListenTLS(certificate string, key string) error {
	cert, err := tls.LoadX509KeyPair(certificate, key)
	if err != nil {
		return errorx.Decorate(err, "failed to load tls keys")

	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}}

	sock, err := tls.Listen("tcp4", s.Host+":"+strconv.Itoa(s.Port), config)
	if err != nil {
		return errorx.Decorate(err, "failed to start server")

	}

	for {
		c, err := sock.Accept()
		if err != nil {
			continue
		}
		go s.serveClient(c, s.Config.MaxReuseConn)
	}

}

// Static (path string, realPath string) serves static files
func (s *Server) Static(path string, realPath string) {
	if path[len(path)-1] == []byte("/")[0] {
		path += "(.*)"
	} else if path[len(path)-1] != []byte("*")[0] {
		path += "/(.*)"
	}
	if realPath[len(realPath)-1] == []byte("/")[0] {
		realPath = realPath[:len(realPath)-1]
	}
	match := pathToRegexp.MustMatch(path, &pathToRegexp.Options{Decode: func(str string, token interface{}) (string, error) {
		return pathToRegexp.DecodeURIComponent(str)
	}})
	s.STATIC = append(s.STATIC, staticRoute{match, realPath})
}

// Get (path string, handler func(req *Request, res *Response)) appends given handler to GET routes
func (s *Server) Get(path string, handler func(req *Request, res *Response)) {
	path += "(\\?.*=.*)?"
	match := pathToRegexp.MustMatch(path, &pathToRegexp.Options{Decode: func(str string, token interface{}) (string, error) {
		return pathToRegexp.DecodeURIComponent(str)
	}})
	s.GET = append(s.GET, route{match, handler})
}

// Post (path string, handler func(req *Request, res *Response)) appends given handler to POST routes
func (s *Server) Post(path string, handler func(req *Request, res *Response)) {
	match := pathToRegexp.MustMatch(path, &pathToRegexp.Options{Decode: func(str string, token interface{}) (string, error) {
		return pathToRegexp.DecodeURIComponent(str)
	}})
	s.POST = append(s.POST, route{match, handler})
}

func (s *Server) serveClient(c net.Conn, reuse int) {
	var finished bool
	if reuse == 0 {
		reuse = 1024
	}
	for i := 0; i < reuse; i++ {
		buf := make([]byte, 256)
		c.Read(buf)
		req, closed, err := getRequest(*(*string)(unsafe.Pointer(&buf)))
		res := getResponse(c, s)
		if err != nil {
			res.Error(res.BadRequest("Cannot Proceed " + req.Path + "\nBad Request"))
			return
		}
		if closed {
			res.Header("Connection", "closed")
			s.processRequest(closed, c, req, res)
			putRequest(req)
			putResponse(res)
			finished = true
			break
		} else {
			if !s.Config.ReuseConn {
				res.Header("Connection", "close")
				s.processRequest(closed, c, req, res)
				finished = true
				c.Close()
				break
			} else {
				res.Header("Connection", "keep-alive")
				s.processRequest(closed, c, req, res)
				continue
			}
		}
	}
	if finished {
		c.Close()
	} else {
		buf := make([]byte, 256)
		c.Read(buf)
		req, _, err := getRequest(*(*string)(unsafe.Pointer(&buf)))
		res := getResponse(c, s)
		if err != nil {
			res.Error(res.BadRequest("Cannot Proceed " + req.Path + "\nBad Request"))
			return
		}
		res.Header("Connection", "close")
		s.processRequest(true, c, req, res)
	}
}

func (s *Server) processRequest(closed bool, c net.Conn, req *Request, res *Response) {
	switch req.Type {
	case "GET":
		var Match *pathToRegexp.MatchResult
		var found bool
		for _, v := range s.STATIC {
			Match, _ = v.Match(req.Path)
			if Match != nil {
				found = true
				if Match.Params[0] == "" {
					res.Error(res.NotFound("Cannot Proceed " + req.Path + "\nNot Found, try doing this with file path"))
					return
				}
				req.Static = true
				req.Params = Match.Params
				s.callMiddleware(req, res)
				filepath := Match.Params[0].(string)
				err := res.SendFile(v.Dir + "/" + filepath[:len(filepath)-1])
				if err != nil {
					res.Error(res.NotFound("Cannot Proceed " + req.Path + "\nNot Found"))
				}
				if closed {
					c.Close()
					return
				}
				return
			}
		}
		for _, v := range s.GET {
			Match, _ = v.Match(req.Path)
			if Match != nil {
				found = true
				req.Params = Match.Params
				s.callMiddleware(req, res)
				v.Handler(req, res)
				if closed {
					c.Close()
					return
				}
				continue
			}
		}
		if !found {
			res.Error(res.NotFound("Cannot Proceed " + req.Path + "\nNot Found"))
		}
	}
}

func (s *Server) callMiddleware(req *Request, res *Response) (*Request, *Response) {
	for _, v := range s.Middleware {
		v(req, res)
	}
	return req, res
}
