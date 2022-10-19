package router

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type ProxyRouter struct {
	proxy *httputil.ReverseProxy
}

func NewProxyRouter(targetHost string) ProxyRouter {
	pr := ProxyRouter{}
	pr.proxy = newProxy(targetHost)
	return pr
}

func newProxy(targetHost string) *httputil.ReverseProxy {
	url, err := url.Parse(targetHost)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: ", err)
		return nil
	}

	return httputil.NewSingleHostReverseProxy(url)
}

func (router ProxyRouter) ServeHTTP(rw http.ResponseWriter, req *http.Request) (ok bool) {
	chanResponseCode := make(chan int)
	router.proxy.ModifyResponse = func(res *http.Response) error {
		chanResponseCode <- res.StatusCode
		return nil
	}
	go router.proxy.ServeHTTP(rw, req)

	statusCode := <-chanResponseCode
	log.Println("Status from proxy:", statusCode)
	return statusCode >= 200
}

func (router ProxyRouter) FindRoutes(req *http.Request) (res []RouterResponse, ok bool) {
	return
}
