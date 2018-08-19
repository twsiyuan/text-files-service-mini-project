package main

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func service(fileDir string) http.Handler {
	const pathPrefix = "/"

	r := mux.NewRouter()
	r.PathPrefix(pathPrefix).Handler(func() http.Handler {
		dir := dirHandler(fileDir, pathPrefix)
		file := retrieveFileHandler(fileDir, pathPrefix)
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if strings.HasSuffix(req.URL.Path, "/") || len(req.URL.Path) <= 0 {
				dir.ServeHTTP(w, req)
				return
			}
			file.ServeHTTP(w, req)
		})
	}()).Methods(http.MethodGet)
	r.PathPrefix(pathPrefix).Handler(modifyFileHandler(fileDir, pathPrefix)).Methods(http.MethodPut)
	r.PathPrefix(pathPrefix).Handler(createFileHandler(fileDir, pathPrefix)).Methods(http.MethodPost)
	r.PathPrefix(pathPrefix).Handler(removeFileHandler(fileDir, pathPrefix)).Methods(http.MethodDelete)

	// TODO: GZIP, CORS (if need)

	return recoveryHandler(true, r)
}
