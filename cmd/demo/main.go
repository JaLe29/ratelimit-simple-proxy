package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
)

func main() {
	var port = 8881

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("/")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("/a")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK a"))
	})

	http.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("/b")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK b"))
	})

	log.Println("Server started on http://localhost:" + strconv.Itoa(port))

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), nil))
}
