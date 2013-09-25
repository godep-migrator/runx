// Usage: runxd
// Environment:
//   RUNX_URL - location and credentials for RSPDY connection
//              e.g. https://name:token@route.webx.io/
package main

import (
	"code.google.com/p/go.net/websocket"
	"crypto/tls"
	"github.com/kr/webx"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"
)

var (
	sshdConn net.Conn
	once     sync.Once
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("runxd: ")
	log.Println("starting")

	innerAddr := "localhost:" + os.Getenv("PORT")
	innerURL, err := url.Parse("http://" + innerAddr)
	if err != nil {
		log.Fatal("parse url:", err)
	}

	_, sshdConn, err = startSSHD([]byte(os.Getenv("AUTHORIZED_KEYS")))
	if err != nil {
		log.Fatal(err)
	}

	rp := httputil.NewSingleHostReverseProxy(innerURL)
	rp.Transport = new(WebsocketTransport)
	http.Handle("/", rp)
	http.Handle("/run", OnceHandler{websocket.Handler(Run)})
	go func() {
		log.Fatal(http.ListenAndServe(innerAddr, nil))
	}()
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	log.Fatal(webx.DialAndServeTLS(os.Getenv("RUNX_URL"), tlsConfig, nil))
}

type OnceHandler struct {
	http.Handler
}

func (h OnceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var ok bool
	log.Println("request")
	once.Do(func() { ok = true })
	if !ok {
		log.Println(">1 try; returning 404")
		http.NotFound(w, r)
		return
	}
	h.Handler.ServeHTTP(w, r)
}

func ioCopy(w io.Writer, r io.Reader, wg *sync.WaitGroup) {
	_, err := io.Copy(w, r)
	if err != nil {
		log.Println(err)
	}
	wg.Done()
}

func Run(ws *websocket.Conn) {
	log.Println("running request")
	var wg sync.WaitGroup
	wg.Add(2)
	go ioCopy(sshdConn, ws, &wg)
	go ioCopy(ws, sshdConn, &wg)
	wg.Wait()
}

type WebsocketTransport struct{}

func (w WebsocketTransport) Proxy(req *http.Request) (*http.Response, error) {
	c, err := net.Dial("tcp", req.URL.Host)
	if err != nil {
		return nil, err
	}
	go io.Copy(c, req.Body)
	return &http.Response{StatusCode: 200, Body: c}, nil
}

func (w WebsocketTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == "WEBSOCKET" {
		return w.Proxy(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}
