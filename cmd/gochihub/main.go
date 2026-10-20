// Command gochihub runs a Gochi cloud hub on its own, without the game:
// players' Gochi servers connect to it to back up and sync pets, see pets
// nearby, share achievements and meet at AR playdates (see the hub
// package). A Gochi server started with -hub serves the same hub alongside
// the game; this program is for running the hub as a separate service.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Michael-W-Ellison/gochi/internal/hub"
)

func main() {
	addr := flag.String("addr", ":8090", "address to listen on")
	dir := flag.String("dir", "", "directory for the hub's data (default ~/.gochi-hub)")
	joinCode := flag.String("join-code", "", "the code players need to register; empty closes registration")
	tlsCert := flag.String("tls-cert", "", "serve HTTPS with this certificate (PEM; needs -tls-key)")
	tlsKey := flag.String("tls-key", "", "the HTTPS certificate's private key (PEM)")
	flag.Parse()

	if err := run(*addr, *dir, *joinCode, *tlsCert, *tlsKey); err != nil {
		log.Fatal(err)
	}
}

// run serves the hub until interrupted
func run(addr, dir, joinCode, tlsCert, tlsKey string) error {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dir = filepath.Join(home, ".gochi-hub")
	}
	if (tlsCert == "") != (tlsKey == "") {
		return errors.New("HTTPS needs both -tls-cert and -tls-key")
	}
	h, err := hub.New(dir, joinCode)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/hub/v1/", h.Handler())
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second,
		TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12}}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverErr := make(chan error, 1)
	go func() {
		if tlsCert != "" {
			serverErr <- server.ListenAndServeTLS(tlsCert, tlsKey)
		} else {
			serverErr <- server.ListenAndServe()
		}
	}()
	scheme := "http"
	if tlsCert != "" {
		scheme = "https"
	}
	log.Printf("Gochi hub at %s://localhost%s/hub/v1, keeping its data in %s", scheme, addr, dir)
	if !h.RegistrationOpen() {
		log.Printf("Registration is closed (no -join-code); existing players can still connect")
	}
	if tlsCert == "" {
		log.Printf("Serving plain HTTP: use -tls-cert and -tls-key, or a reverse proxy, before opening it to the internet")
	}

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
	}
	log.Printf("Shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}
