package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func sayHello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello World！")
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", sayHello)

	srv := &http.Server{
		Addr:           ":8080",
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		ErrorLog:       log.Default(),
		MaxHeaderBytes: 1 << 20,
	}
	serverConnClosed := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer func() {
			cancel()
			close(serverConnClosed)
		}()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("http server shutdown failed, err:%v\n", err)
			return
		}

		log.Printf("http server will shutdown after 5 seconds\n")
		<-ctx.Done()
		log.Printf("http server shutdown successfully\n")
	}()

	srv.Addr = ":8080"
	srv.Handler = http.HandlerFunc(sayHello)
	log.Printf("http server start at %s\n", srv.Addr)
	err := srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Printf("http server failed, err:%v\n", err)
		panic(err)
	}

	<-serverConnClosed
}
