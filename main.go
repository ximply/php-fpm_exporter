package main

import (
	"flag"
	"net"
	"os"
	"net/http"
	"strings"
	"io"
	"github.com/parnurzeal/gorequest"
	"time"
	"fmt"
)

var (
	Name           = "php-fpm_exporter"
	listenAddress  = flag.String("unix-sock", "/dev/shm/php-fpm_exporter.sock", "Address to listen on for unix sock access and telemetry.")
	metricsPath    = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	statusUrl      = flag.String("url", "", "Destination php-fpm status url.")
)

func metrics(w http.ResponseWriter, r *http.Request) {
	req := gorequest.New()
	_, body, errs := req.Retry(1, 2 * time.Second,
		http.StatusBadRequest, http.StatusInternalServerError).Get(*statusUrl).End()
	if errs != nil {
		io.WriteString(w, "")
		return
	}

	ret := ""
	l := strings.Split(body, "\n")
	if len(l) > 2 {
		for _, i := range l {
			if strings.HasPrefix(i,"active processes:") {
				s := strings.TrimLeft(i,"active processes:")
				ret = ret + fmt.Sprintf("php_fpm_active_processes %s\n",
					strings.Replace(s, " ", "", -1))
			} else if strings.HasPrefix(i,"total processes:") {
				s := strings.TrimLeft(i,"total processes:")
				ret = ret + fmt.Sprintf("php_fpm_total_processes %s\n",
					strings.Replace(s, " ", "", -1))
			}
		}
	}
	io.WriteString(w, ret)
}

func main() {
	flag.Parse()

	addr := "/dev/shm/php-fpm_exporter.sock"
	if listenAddress != nil {
		addr = *listenAddress
	}

	if statusUrl == nil {
		panic("error status url")
	}

	mux := http.NewServeMux()
	mux.HandleFunc(*metricsPath, metrics)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Php-fpm Exporter</title></head>
             <body>
             <h1>Php-fpm Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})
	server := http.Server{
		Handler: mux, // http.DefaultServeMux,
	}
	os.Remove(addr)

	listener, err := net.Listen("unix", addr)
	if err != nil {
		panic(err)
	}
	server.Serve(listener)
}