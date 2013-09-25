// Usage: runxd
// Environment:
//   RUNX_URL - location and credentials for RSPDY connection
//              e.g. https://name:token@route.webx.io/
package main

import (
	"code.google.com/p/go.net/websocket"
	"crypto/tls"
	"encoding/json"
	"github.com/kr/pty"
	"github.com/kr/webx"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"sync"
	"time"
)

var tlsConfig = &tls.Config{
	InsecureSkipVerify: true,
}

var once sync.Once

func main() {
	log.SetFlags(0)
	log.SetPrefix("runxd: ")
	log.Println("starting")

	innerAddr := "localhost:" + os.Getenv("PORT")
	innerURL, err := url.Parse("http://" + innerAddr)
	if err != nil {
		log.Fatal("parse url:", err)
	}

	rp := httputil.NewSingleHostReverseProxy(innerURL)
	rp.Transport = new(WebsocketTransport)
	http.Handle("/", rp)
	http.Handle("/run", OnceHandler{websocket.Handler(Run)})
	go func() {
		log.Fatal(http.ListenAndServe(innerAddr, nil))
	}()
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

func Run(ws *websocket.Conn) {
	defer ws.Close()
	log.Println("running request")
	dec := json.NewDecoder(ws)
	var p struct{ Args, Env []string }
	err := dec.Decode(&p)
	if err != nil {
		log.Println("bad request", err)
		return
	}
	cmd := exec.Command(p.Args[0], p.Args[1:]...)
	f, err := pty.Start(cmd)
	if err != nil {
		io.WriteString(ws, err.Error())
		return
	}
	go io.Copy(f, io.MultiReader(dec.Buffered(), ws))
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		io.Copy(ws, f)
		wg.Done()
	}()
	err = cmd.Wait()
	if err != nil {
		log.Println(err)
	}
	wg.Wait()
}

type WebsocketTransport struct{}

func (w WebsocketTransport) Proxy(req *http.Request) (*http.Response, error) {
	conn, err := net.DialTimeout("tcp", req.URL.Host, 50*time.Millisecond)
	if err != nil {
		return nil, err
	}
	go io.Copy(conn, req.Body)
	resp := &http.Response{
		StatusCode: 200,
		Body:       conn,
	}
	return resp, nil
}

func (w WebsocketTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == "WEBSOCKET" {
		return w.Proxy(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}
