package mars

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

var (
	MainRouter         *Router
	MainTemplateLoader *TemplateLoader
	MainWatcher        *Watcher
	Server             *http.Server
)

// Handler is a http.HandlerFunc which exposes Mars' filtering, routing, and
// interception functionality for you to use with custom HTTP servers.
var Handler = http.HandlerFunc(handle)

// This method handles all requests.  It dispatches to handleInternal after
// handling / adapting websocket connections.
func handle(w http.ResponseWriter, r *http.Request) {
	if maxRequestSize := int64(Config.IntDefault("http.maxrequestsize", 0)); maxRequestSize > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)
	}

	upgrade := r.Header.Get("Upgrade")
	if upgrade == "websocket" || upgrade == "Websocket" {
		websocket.Handler(func(ws *websocket.Conn) {
			//Override default Read/Write timeout with sane value for a web socket request
			ws.SetDeadline(time.Now().Add(time.Hour * 24))
			r.Method = "WS"
			handleInternal(w, r, ws)
		}).ServeHTTP(w, r)
	} else {
		handleInternal(w, r, nil)
	}
}

func handleInternal(w http.ResponseWriter, r *http.Request, ws *websocket.Conn) {
	var (
		req  = NewRequest(r)
		resp = NewResponse(w)
		c    = NewController(req, resp)
	)
	req.Websocket = ws

	Filters[0](c, Filters[1:])
	if c.Result != nil {
		c.Result.Apply(req, resp)
	} else if c.Response.Status != 0 {
		c.Response.Out.WriteHeader(c.Response.Status)
	}
	// Close the Writer if we can
	if w, ok := resp.Out.(io.Closer); ok {
		w.Close()
	}
}

// Run the server.
// This is called from the generated main file.
// If port is non-zero, use that.  Else, read the port from app.conf.
func Run(port int) {
	address := HttpAddr
	if port == 0 {
		port = HttpPort
	}

	var network = "tcp"
	var localAddress string

	// If the port is zero, treat the address as a fully qualified local address.
	// This address must be prefixed with the network type followed by a colon,
	// e.g. unix:/tmp/app.socket or tcp6:::1 (equivalent to tcp6:0:0:0:0:0:0:0:1)
	if port == 0 {
		parts := strings.SplitN(address, ":", 2)
		network = parts[0]
		localAddress = parts[1]
	} else {
		localAddress = address + ":" + strconv.Itoa(port)
	}

	Server = &http.Server{
		Addr:         localAddress,
		Handler:      Handler,
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
	}

	go func() {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("Listening on %s...\n", localAddress)
	}()

	if HttpSsl {
		if network != "tcp" {
			// This limitation is just to reduce complexity, since it is standard
			// to terminate SSL upstream when using unix domain sockets.
			ERROR.Fatalln("SSL is only supported for TCP sockets. Specify a port to listen on.")
		}
		ERROR.Fatalln("Failed to listen:",
			Server.ListenAndServeTLS(HttpSslCert, HttpSslKey))
	} else {
		listener, err := net.Listen(network, localAddress)
		if err != nil {
			ERROR.Fatalln("Failed to listen:", err)
		}
		ERROR.Fatalln("Failed to serve:", Server.Serve(listener))
	}
}
