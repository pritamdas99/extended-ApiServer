package main

import (
	"fmt"
	"github.com/PritamDas17021999/extended-ApiServer/lib/certstore"
	"github.com/PritamDas17021999/extended-ApiServer/lib/server"
	"github.com/gorilla/mux"
	"github.com/spf13/afero"
	"k8s.io/client-go/util/cert"
	"log"
	"net"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(w, "Ok")
}

func main() {
	fs := afero.NewOsFs()
	store, err := certstore.NewCertStore(fs, "/tmp/extended-ApiServer")
	if err != nil {
		log.Fatalln(err)
	}
	err = store.NewCA("apiserver")
	if err != nil {
		log.Fatalln(err)
	}
	serverCert, serverKey, err := store.NewServerCertPair(cert.AltNames{
		IPs: []net.IP{net.ParseIP("127.0.1.1")},
	})
	if err != nil {
		log.Fatalln(err)
	}
	err = store.Write("tls", serverCert, serverKey)
	if err != nil {
		log.Fatalln(err)
	}
	clientCert, clientKey, err := store.NewClientCertPair(cert.AltNames{
		DNSNames: []string{"john"},
	})
	if err != nil {
		log.Fatalln(err)
	}

	err = store.Write("john", clientCert, clientKey)
	if err != nil {
		log.Fatalln(err)
	}

	cfg := server.Config{
		Address: "127.0.1.1:8443",
		CACertFiles: []string{
			store.CertFile("ca"),
		},
		CertFile: store.CertFile("tls"),
		KeyFile:  store.KeyFile("tls"),
	}
	srv := server.NewGenericServer(cfg)
	r := mux.NewRouter()
	r.HandleFunc("/core/{resource}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Resource : %v\n", vars["resource"])
	})
	r.HandleFunc("/", handler)
	srv.ListenAndServe(r)
}
