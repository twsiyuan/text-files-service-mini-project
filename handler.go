package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/unrolled/render"
)

type key int

const (
	keyFileName key = iota
	keyContent
)

const (
	jsonContentType = "application/json; charset=utf-8"
)

var ren = render.New()

type responseError struct {
	Error string
}

type contentBody struct {
	Content string
}

// recoveryHandler is a handler that handles and logs panics
func recoveryHandler(outputErr bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				stack := make([]byte, 5012)
				stack = stack[:runtime.Stack(stack, false)]
				displayErr := "Internal server error"
				if outputErr {
					displayErr = fmt.Sprintf("Unexpected error: %v, in %s", err, stack)
				}

				fmt.Fprintf(os.Stderr, "Unexpected error: %v, in %s\n", err, stack)
				ren.JSON(w, http.StatusInternalServerError, responseError{displayErr})
			}
		}()

		next.ServeHTTP(w, req)
	})
}

// filePathMiddleware is a middleware that converts URL path to physical file path, then stores the file path into context
func filePathMiddleware(fileDir, pathPrefix string, next http.Handler) http.Handler {
	if len(fileDir) <= 0 {
		panic("fileDir should not be empty")
	}
	return http.StripPrefix(pathPrefix, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(fileDir, "./") {
			baseDir, err := os.Getwd()
			if err != nil {
				panic(err)
			}
			fileDir = filepath.Join(baseDir, strings.TrimPrefix(fileDir, "./"))
		}
		fileName := filepath.Join(fileDir, "/", req.URL.Path)
		if u := req.URL.Path; !strings.HasSuffix(u, "/") && len(u) > 0 {
			fileName += ".txt"
		} else if !strings.HasSuffix(fileName, "/") {
			fileName += "/"
		}

		ctx := context.WithValue(req.Context(), keyFileName, fileName)
		req = req.WithContext(ctx)
		next.ServeHTTP(w, req)
	}))
}

// jsonMiddleware is a middleware that tests request content-type should be application/json; charset=utf-8. If test failed, it will return http.StatusUnsupportedMediaType
func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		pass := true
		contentType := req.Header.Get("CONTENT-TYPE")
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			pass = false
		} else if mediaType != "application/json" {
			pass = false
		} else if charset, exist := params["charset"]; !exist {
			pass = false
		} else if charset != "utf-8" {
			pass = false
		}

		if !pass {
			ren.JSON(w, http.StatusUnsupportedMediaType, responseError{"Bad request, invalid content-type"})
			return
		}

		next.ServeHTTP(w, req)
	})
}

// contentMiddleware is a middleware that reads and parses body, then stores the content into context
func contentMiddleware(next http.Handler) http.Handler {
	return jsonMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.ContentLength <= 0 {
			ren.JSON(w, http.StatusBadRequest, responseError{"Bad request, no content"})
			return
		}

		decoder := json.NewDecoder(req.Body)
		decoder.DisallowUnknownFields()
		defer func() {
			io.Copy(ioutil.Discard, req.Body)
			req.Body.Close()
		}()

		c := contentBody{}
		if err := decoder.Decode(&c); err != nil || len(c.Content) <= 0 {
			ren.JSON(w, http.StatusBadRequest, responseError{"Bad request, json parse failed"})
			return
		}

		ctx := context.WithValue(req.Context(), keyContent, c.Content)
		req = req.WithContext(ctx)
		next.ServeHTTP(w, req)
	}))
}

// fileExistsMiddleware is a middleware that check file exists at certain path, the path is from Context()
//
// If file does not exist, it will response http.StatusNotFound
//
// Note: Must pass filePathMiddleware
func fileExistsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fileName := req.Context().Value(keyFileName).(string)

		ok := false
		if fileInfo, err := os.Stat(fileName); err == nil {
			ok = !fileInfo.IsDir()
		}

		if !ok {
			ren.JSON(w, http.StatusNotFound, responseError{"File does not exist"})
			return
		}

		next.ServeHTTP(w, req)
	})
}

// fileNotExistsMiddleware is a middleware that check file not exists at certain path, the path is from Context()
//
// If file does exist, it will response http.StatusForbidden
//
// Note: Must pass filePathMiddleware
func fileNotExistsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fileName := req.Context().Value(keyFileName).(string)

		ok := false
		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			ok = true
		}

		if !ok {
			ren.JSON(w, http.StatusForbidden, responseError{"File does exist"})
			return
		}

		next.ServeHTTP(w, req)
	})
}

// folderExistsMiddleware is a middleware that check folder exists at certain path, the path is from Context()
//
// If file does not exist, it will response http.StatusNotFound
//
// Note: Must pass filePathMiddleware
func folderExistsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fileName := req.Context().Value(keyFileName).(string)

		ok := false
		if fileInfo, err := os.Stat(fileName); err == nil {
			ok = fileInfo.IsDir()
		}

		if !ok {
			print(fileName)
			ren.JSON(w, http.StatusNotFound, responseError{"Folder does not exist"})
			return
		}

		next.ServeHTTP(w, req)
	})
}

// createFileHandler is a handler that create a file from request
func createFileHandler(fileDir, pathPrefix string) http.Handler {
	return filePathMiddleware(fileDir, pathPrefix, fileNotExistsMiddleware(contentMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		fileName := ctx.Value(keyFileName).(string)
		dirName := filepath.Dir(fileName)
		content := ctx.Value(keyContent).(string)

		if err := os.MkdirAll(dirName, os.ModePerm); err != nil {
			panic(err)
		}

		file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		if _, err := file.Write(([]byte)(content)); err != nil {
			panic(err)
		}

		// TODO: Log and send operator ID
		ren.JSON(w, http.StatusOK, "Done")
	}))))
}

// modifyFileHandler is a handler that update the file from request
func modifyFileHandler(fileDir, pathPrefix string) http.Handler {
	return filePathMiddleware(fileDir, pathPrefix, fileExistsMiddleware(contentMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		fileName := ctx.Value(keyFileName).(string)
		content := ctx.Value(keyContent).(string)

		file, err := os.OpenFile(fileName, os.O_TRUNC|os.O_WRONLY, os.ModePerm)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		if _, err := file.Write(([]byte)(content)); err != nil {
			panic(err)
		}

		// TODO: Log and send operator ID
		ren.JSON(w, http.StatusOK, "Done")
	}))))
}

// removeFileHandler is a handler that remove the file
func removeFileHandler(fileDir, pathPrefix string) http.Handler {
	return filePathMiddleware(fileDir, pathPrefix, fileExistsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fileName := req.Context().Value(keyFileName).(string)

		if err := os.Remove(fileName); err != nil {
			panic(err)
		}

		// TODO: Log and send operator ID
		ren.JSON(w, http.StatusOK, "Done")
	})))
}

// retrieveFileHandler is a handler that inspect the file content
func retrieveFileHandler(fileDir, pathPrefix string) http.Handler {
	return filePathMiddleware(fileDir, pathPrefix, fileExistsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fileName := req.Context().Value(keyFileName).(string)

		b, err := ioutil.ReadFile(fileName)
		if err != nil {
			panic(err)
		}

		ren.JSON(w, http.StatusOK, contentBody{
			string(b),
		})
	})))
}

// dirHandler is a handler that get some statistics per folder
func dirHandler(fileDir, pathPrefix string) http.Handler {
	return filePathMiddleware(fileDir, pathPrefix, folderExistsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		dirname := req.Context().Value(keyFileName).(string)
		stat, err := dirStatistics(dirname)
		if err != nil {
			panic(err)
		}
		ren.JSON(w, http.StatusOK, stat)
	})))
}
