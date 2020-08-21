package main

import (
	"fmt"
	"io"
	"net/http"
)

func handle(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Hello world")
}

func main() {
	portNumber := "6515"
	http.HandleFunc("/", handle)
	fmt.Printf("Server listening on http://localhost:%s\n", portNumber)
	http.ListenAndServe(":"+portNumber, nil)
}
