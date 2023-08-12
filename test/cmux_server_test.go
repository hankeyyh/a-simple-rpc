package test

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/soheilhy/cmux"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"testing"
)

var (
	name string = "yuhan"
)

func TestCmuxServer(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:1234")
	if err != nil {
		log.Panic(err)
	}
	m := cmux.New(l)

	httpl := m.Match(cmux.HTTP1())

	go serveHttp(httpl)

	if err = m.Serve(); err != nil {
		log.Panic(err)
	}
}

func serveHttp(ln net.Listener) {
	router := httprouter.New()
	router.GET("/get-name", getName)
	router.POST("/up-name", upName)
	s := &http.Server{
		Handler: router,
	}
	if err := s.Serve(ln); err != nil {
		log.Panic(err)
	}
}

func getName(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, name)
	log.Printf("get name: %s", name)
}

func upName(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	b := make([]byte, 2)
	tmp := ""
	for {
		n, err := r.Body.Read(b)
		tmp += string(b[:n])
		if err == io.EOF {
			break
		}
	}
	if tmp == "" {
		wh := w.Header()
		wh.Set("X-SIMPLE-ERRMSG", "name can't not be empty")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	name = tmp
	log.Printf("update name: %s", name)
}

func TestQuery(t *testing.T) {
	mm, err := url.ParseQuery("?a=10&b=20")
	if err != nil {
		log.Panic(err)
	}
	for k, v := range mm {
		fmt.Printf("%s:%s\n", k, v[0])
	}
}
