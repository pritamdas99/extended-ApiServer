package main

import (
	"crypto/x509"
	"flag"
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
	var proxy = false
	flag.BoolVar(&proxy, "receive-proxy-request", proxy, "receive forwarded requests from apiserver")
	flag.Parse()
	fs := afero.NewOsFs()
	store, err := certstore.NewCertStore(fs, "/tmp/extended-ApiServer")
	if err != nil {
		log.Fatalln(err)
	}
	err = store.InitCA("database")
	if err != nil {
		log.Fatalln(err)
	}
	serverCert, serverKey, err := store.NewServerCertPair(cert.AltNames{
		IPs: []net.IP{net.ParseIP("127.0.0.2")},
	})
	if err != nil {
		log.Fatalln(err)
	}
	err = store.Write("tls", serverCert, serverKey)
	if err != nil {
		log.Fatalln(err)
	}
	clientCert, clientKey, err := store.NewClientCertPair(cert.AltNames{
		DNSNames: []string{"jane"},
	})
	if err != nil {
		log.Fatalln(err)
	}

	err = store.Write("jane", clientCert, clientKey)
	if err != nil {
		log.Fatalln(err)
	}

	//---------------------------------------------
	apiserverStore, err := certstore.NewCertStore(fs, "/tmp/extended-ApiServer")
	if err != nil {
		log.Fatalln(err)
	}
	if proxy {
		err = apiserverStore.LoadCA("apiserver")
		if err != nil {
			log.Fatalln(err)
		}
	}
	//-----------------------------------------------

	//-----------------------------------------------
	rhCACertPool := x509.NewCertPool()
	rhStore, err := certstore.NewCertStore(fs, "/tmp/extended-ApiServer")
	if err != nil {
		log.Fatalln(err)
	}
	if proxy {
		err = rhStore.LoadCA("requestheader")
		if err != nil {
			log.Fatalln(err)
		}
		rhCACertPool.AppendCertsFromPEM(rhStore.CACertBytes())
	}
	//---------------------------------------------------

	cfg := server.Config{
		Address: "127.0.0.2:8444",
		CACertFiles: []string{
			store.CertFile("ca"),
		},
		CertFile: store.CertFile("tls"),
		KeyFile:  store.KeyFile("tls"),
	}
	if proxy {
		cfg.CACertFiles = append(cfg.CACertFiles, apiserverStore.CertFile("ca"))
		cfg.CACertFiles = append(cfg.CACertFiles, rhStore.CertFile("ca"))
	}
	srv := server.NewGenericServer(cfg)
	r := mux.NewRouter()
	r.HandleFunc("/database/{resource}", func(w http.ResponseWriter, r *http.Request) {
		user := "system:anonymous"
		src := "-"
		if len(r.TLS.PeerCertificates) > 0 {
			opts := x509.VerifyOptions{
				Roots:     rhCACertPool,
				KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			}

			if _, err := r.TLS.PeerCertificates[0].Verify(opts); err != nil {
				user = r.TLS.PeerCertificates[0].Subject.CommonName
				src = "Client-Cert-CN"
			} else {
				user = r.Header.Get("X-Remote-User")
				src = "X-Remote-User"
			}
		}
		vars := mux.Vars(r)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Resource : %v requested by user [%s]=%s\n", vars["resource"], src, user)
	})
	r.HandleFunc("/", handler)
	srv.ListenAndServe(r)
}
