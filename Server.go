package GoExpress

import (
	"os"
	"net"
	"strconv"
	log "github.com/sirupsen/logrus"
	"crypto/tls"
	"github.com/borzhchevskiy/go-express/internal/static"
)

type Server struct {
	Host         string
	Port         int
	Socket       net.Listener
	STATIC       map[string]string
	Middleware   []func(req *Request, res *Response)
	GET          map[string]func(req *Request, res *Response)
	POST         map[string]func(req *Request, res *Response)
}

// Express(Host, Port) returns a Server object
func Express(host string, port int, logLevel string) *Server {
	switch logLevel {
		case "info":
			log.SetLevel(log.InfoLevel)
		case "warn":
			log.SetLevel(log.WarnLevel)
		case "fatal":
			log.SetLevel(log.FatalLevel)
		default:
			log.SetLevel(log.FatalLevel)
	}
	s := &Server {
		Host:       host,
		Port:       port,
		STATIC:     make(map[string]string),
		Middleware: make([]func(req *Request, res *Response), 0),
		GET:        make(map[string]func(req *Request, res *Response)),
		POST:       make(map[string]func(req *Request, res *Response)),
	}
	return s
}

// Server.Use(Middleware) appends given middleware to server
func (s *Server) Use(middleware func(req *Request, res *Response)) {
	s.Middleware = append(s.Middleware, middleware)
}

// Server.Listen() listens for connections
func (s *Server) Listen() {
	log.WithFields(log.Fields{
		"GET": s.GET,
		"POST": s.POST,
		"STATIC": s.STATIC,

	}).Info("")
	var err error
	s.Socket, err = net.Listen("tcp4", s.Host + ":" + strconv.Itoa(s.Port))
	if err != nil {
		log.Warn(err)
		os.Exit(1)
	}
	for {
		c, _ := s.Socket.Accept()
		go s.serveClient(c)
	}
	s.Socket.Close()
}

// Server.ListenTLS(CERTIFICATE, KEY) listens for connections, and process it with tls
func (s *Server) ListenTLS(certificate string, key string) {
	cert, err := tls.LoadX509KeyPair(certificate, key)
	if err != nil {
		log.Warn(err)
		os.Exit(1)
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}}

	sock, err := tls.Listen("tcp4", s.Host + ":" + strconv.Itoa(s.Port), config)
	if err != nil {
		log.Warn(err)
		os.Exit(1)
	}

	for {
		c, err := sock.Accept()
		if err != nil {
			log.Warn(err)
		}
		go s.serveClient(c)
	}
	sock.Close()
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
	s.GET[path] = handler
}

// Server.Post(PATH, HANDLER) appends given handler to Post routes
func (s *Server) Post(path string, handler func(req *Request, res *Response)) {
	if path[len(path)-1] != []byte("/")[0] {
		path += "/"
	}
	s.POST[path] = handler
}

// Server.serveClient(CONN) its a private method that process request in goroutine
func (s *Server) serveClient(c net.Conn) {
	for {
		buf := make([]byte, 1024)
		c.Read(buf)
		req, closed, err := getRequest(string(buf))
		res := getResponse(c)
		if err != nil {
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

func (s *Server) processRequest(closed bool, c net.Conn, req *Request, res *Response) {
	switch req.Type {
		case "GET":
			Static, filePath := static.ProcessStatic(s.STATIC, req.Path)
			if _, OK := s.GET[req.Path]; OK {
				s.callMiddleware(req, res)
				s.GET[req.Path](req, res)
				if closed {
					c.Close()
				}
			} else if Static {
				err := res.SendFile(filePath)
				if err != nil {
					log.Warn(err)
				}
			} else {
				res.Error(res.NotFound("Cannot Proceed " + req.Path + "\nNot Found"))
			}
		case "POST":
			if _, OK := s.POST[req.Path]; OK {
				s.callMiddleware(req, res)
				s.POST[req.Path](req, res)
				if closed {
					c.Close()
				}
			} else {
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