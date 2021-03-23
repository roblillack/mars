package mars

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/net/websocket"

	"github.com/roblillack/mars/internal/watcher"
)

var (
	MainRouter         *Router
	MainTemplateLoader *TemplateLoader
	mainWatcher        *watcher.Watcher
	Server             *http.Server
	SecureServer       *http.Server
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

func makeServer(addr string) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      Handler,
		ReadTimeout:  time.Duration(Config.IntDefault("timeout.read", 0)) * time.Second,
		WriteTimeout: time.Duration(Config.IntDefault("timeout.write", 0)) * time.Second,
	}
}

func initGracefulShutdown() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stop
		INFO.Println("Shutting down listeners ...")

		ctx := context.Background()
		if timeout := Config.IntDefault("timeout.shutdown", 0); timeout != 0 {
			newCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
			ctx = newCtx
			defer cancel()
		}

		if SecureServer != nil {
			if err := SecureServer.Shutdown(ctx); err != nil {
				ERROR.Println(err)
			}
		}
		if Server != nil {
			if err := Server.Shutdown(ctx); err != nil {
				ERROR.Println(err)
			}
		}
	}()
}

func Run() {
	if !setupDone {
		setup()
	}

	if DevMode {
		INFO.Printf("Development mode enabled.")
	}

	wg := sync.WaitGroup{}
	initializeFallbacks()
	initGracefulShutdown()

	if !HttpSsl || DualStackHTTP {
		go func() {
			time.Sleep(100 * time.Millisecond)
			INFO.Printf("Listening on %s (HTTP) ...\n", HttpAddr)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()

			Server = makeServer(HttpAddr)
			if err := Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				ERROR.Fatalln("Failed to serve:", err)
			}
		}()
	}

	if HttpSsl || DualStackHTTP {
		go func() {
			time.Sleep(100 * time.Millisecond)
			INFO.Printf("Listening on %s (HTTPS) ...\n", SSLAddr)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()

			serveTLS(SSLAddr)
		}()
	}

	wg.Wait()

	runShutdownHooks()
}

func serveTLS(addr string) {
	SecureServer = makeServer(addr)

	SecureServer.TLSConfig = &tls.Config{
		Certificates: make([]tls.Certificate, 1),
	}
	if SelfSignedCert {
		keypair, err := createCertificate(SelfSignedOrganization, SelfSignedDomains)
		if err != nil {
			ERROR.Fatalln("Unable to create key pair:", err)
		}
		SecureServer.TLSConfig.Certificates[0] = keypair
	} else {
		keypair, err := tls.LoadX509KeyPair(HttpSslCert, HttpSslKey)
		if err != nil {
			ERROR.Fatalln("Unable to load key pair:", err)
		}
		SecureServer.TLSConfig.Certificates[0] = keypair
	}

	if err := SecureServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
		ERROR.Fatalln("Failed to serve:", err)
	}
}
