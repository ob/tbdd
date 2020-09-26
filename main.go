package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/ob/tbdd/disk"
)

const dataDir = "storeDir/dataDir"
const indexDir = "storeDir/indexDir"
const tempDir = "storeDir/tempDir"

func main() {
	var port string
	var bindAddr string
	var dir string
	flag.StringVar(&bindAddr, "host", "0.0.0.0", "address to listen on")
	flag.StringVar(&port, "port", "12625", "port on which to listen to")
	flag.StringVar(&dir, "dir", "", "dir on which to store the data")
	flag.Parse()

	if dir == "" {
		fmt.Println("--dir <arg> required")
		os.Exit(1)
	}

	storage := disk.New(dir)
	mux := http.NewServeMux()
	httpServer := &http.Server{
		Addr:         bindAddr + ":" + port,
		Handler:      mux,
		ReadTimeout:  0,
		WriteTimeout: 0,
	}
	mux.HandleFunc("/", storage.RequestHandler)
	log.Printf("Server listening on http://%s\n", httpServer.Addr)
	httpServer.ListenAndServe()
}
