package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	// TODO: Get arguments from os.Environ or os.Args
	port := "8080"
	fileDir := "./files"

	h := service(fileDir)
	fmt.Fprintf(os.Stdout, "Listening :%v...\n", port)
	http.ListenAndServe(":"+port, h)
}
