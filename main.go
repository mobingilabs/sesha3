package main

import (
	"fmt"
	"net/http"
	"time"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Version 5: "+time.Now().String())
}

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":80", nil)
}
