package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/buidl-labs/Demux/dataservice"
	// "github.com/buidl-labs/Demux/server"
	// "github.com/buidl-labs/Demux/util"

	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

func main() {
	dataservice.InitDB()
	// go util.RunPoller()
	revProxyServer()
	// server.StartServer(":" + os.Getenv("PORT"))
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

func revProxyServer() {
	logrus.SetFormatter(&logrus.TextFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)
	logger := logrus.WithFields(logrus.Fields{
		"service": "go-reverse-proxy",
	})

	router := httprouter.New()
	//origin, _ := url.Parse("http://localhost:9000/")
	//path := "/*catchall"
	origin, _ := url.Parse("http://52.147.201.40:8080/")
	path := "/*catchall"

	reverseProxy := httputil.NewSingleHostReverseProxy(origin)

	reverseProxy.Director = func(req *http.Request) {
		req.Header.Add("X-Forwarded-Host", req.Host)
		req.Header.Add("X-Origin-Host", origin.Host)
		req.URL.Scheme = origin.Scheme
		req.URL.Host = origin.Host

		wildcardIndex := strings.IndexAny(path, "*")
		proxyPath := singleJoiningSlash(origin.Path, req.URL.Path[wildcardIndex:])
		if strings.HasSuffix(proxyPath, "/") && len(proxyPath) > 1 {
			proxyPath = proxyPath[:len(proxyPath)-1]
		}
		req.URL.Path = proxyPath
	}

	router.Handle("GET", path, func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		reverseProxy.ServeHTTP(w, r)
	})

	logger.Fatal(http.ListenAndServe(":80", router))
}
