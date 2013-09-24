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
	"github.com/kr/rspdy"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	redialPause = 2 * time.Second
)

var tlsConfig = &tls.Config{
	InsecureSkipVerify: true,
}

var once sync.Once

func main() {
	log.SetFlags(0)
	log.SetPrefix("runxd: ")
	log.Println("starting")
	routerURL, err := url.Parse(os.Getenv("RUNX_URL"))
	if err != nil {
		log.Fatal("parse url:", err)
	}
	mustSanityCheckURL(routerURL)

	handshake := func(w http.ResponseWriter, r *http.Request) {
		webxName := routerURL.User.Username()
		password, _ := routerURL.User.Password()
		cmd := BackendCommand{"add", webxName, password}
		err = json.NewEncoder(w).Encode(cmd)
		if err != nil {
			log.Fatal("handshake:", err)
		}
		log.Println("handshake complete")
		select {}
	}

	innerAddr := "localhost:" + os.Getenv("PORT")
	innerURL, err := url.Parse("http://" + innerAddr)
	if err != nil {
		log.Fatal("parse url:", err)
	}

	rp := httputil.NewSingleHostReverseProxy(innerURL)
	rp.Transport = new(WebsocketTransport)
	http.Handle("/", rp)
	http.Handle("/run", OnceHandler{websocket.Handler(Run)})
	http.HandleFunc("backend.webx.io/names", handshake)
	addr := routerURL.Host
	if p := strings.LastIndex(addr, ":"); p == -1 {
		addr += ":https"
	}
	go func() {
		log.Fatal(http.ListenAndServe(innerAddr, nil))
	}()
	for {
		log.Println("dialing")
		err = rspdy.DialAndServeTLS("tcp", addr, tlsConfig, nil)
		if err != nil {
			log.Println("DialAndServe:", err)
			time.Sleep(redialPause)
		}
	}
}

func Catchall(w http.ResponseWriter, r *http.Request) {
	log.Println("catchall 404")
	dump, err := httputil.DumpRequest(r, false)
	if err != nil {
		log.Println(err)
	} else {
		os.Stdout.Write(dump)
	}
	http.NotFound(w, r)
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

func mustSanityCheckURL(u *url.URL) {
	if u.User == nil {
		log.Fatal("url has no userinfo")
	}
	if u.Scheme != "https" {
		log.Fatal("scheme must be https")
	}
	if u.Path != "/" {
		log.Fatal("path must be /")
	}
	if u.RawQuery != "" {
		log.Fatal("query must be empty")
	}
	if u.Fragment != "" {
		log.Fatal("fragment must be empty")
	}
}

type BackendCommand struct {
	Op       string // "add" or "remove"
	Name     string // e.g. "foo" for foo.webxapp.io
	Password string
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
