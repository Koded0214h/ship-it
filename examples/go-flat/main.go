package main

import (
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}
