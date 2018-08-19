package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFilePathMiddleware(t *testing.T) {
	const notFoundResult = "404"
	testFunc := func(fileDir, pathPrefix, requestURL, expectPath string) {
		h := filePathMiddleware(fileDir, pathPrefix, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if expectPath != notFoundResult {
				fpath := req.Context().Value(keyFileName).(string)
				if fpath != expectPath {
					t.Errorf("Wrong path, fileDir: %s, pathPrefix: %s, requestURL: %s, expectPath: %s, got: %s", fileDir, pathPrefix, requestURL, expectPath, fpath)
				}
			}
		}))

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, requestURL, nil)
		h.ServeHTTP(w, req)
		if expectPath == notFoundResult {
			if w.Code != http.StatusNotFound {
				t.Errorf("Expected not found, but not, pathPrefix: %s, requestURL: %s", pathPrefix, requestURL)
			}
		}
	}

	testFunc("/", "/", "/", "/")
	testFunc("/", "/", "/io", "/io.txt")
	testFunc("/", "/", "/docs/io", "/docs/io.txt")
	testFunc("/", "/", "/docs/io/", "/docs/io/")
	testFunc("/files", "/api", "/api/io", "/files/io.txt")
	testFunc("/files", "/api", "/api/io/", "/files/io/")
	testFunc("/files", "/api", "/io", notFoundResult)
	testFunc("/static", "/", "http://127.0.0.1", "/static/")
	testFunc("/static", "/", "http://127.0.0.1/", "/static/")
	testFunc("/static", "/", "http://127.0.0.1/io", "/static/io.txt")
	testFunc("/static", "/", "http://127.0.0.1/io/", "/static/io/")

	baseDir, _ := os.Getwd()
	testFunc("./files", "/api/v2", "/api/v2/io", filepath.Join(baseDir, "/files/io.txt"))
}

func TestJsonMiddleware(t *testing.T) {
	testFunc := func(contentType string, expectCode int) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("CONTENT-TYPE", contentType)
		w := httptest.NewRecorder()
		h := jsonMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		}))

		h.ServeHTTP(w, req)
		if w.Code != expectCode {
			t.Errorf("Unexpected code, content-Type: %s, code: %d", contentType, w.Code)
		}
	}

	testFunc(jsonContentType, http.StatusOK)
	testFunc("application/json; charset=utf-8", http.StatusOK)
	testFunc("application/json;charset=utf-8     ", http.StatusOK)
	testFunc("application/json; charset=utf-8; code=123", http.StatusOK)
	testFunc("application/json;      charset      =      utf-8", http.StatusOK)
	testFunc("application/json; charset=utf-8; code; yyyy", http.StatusUnsupportedMediaType)
	testFunc("application/json;", http.StatusUnsupportedMediaType)
	testFunc("application/json", http.StatusUnsupportedMediaType)
	testFunc("text/html", http.StatusUnsupportedMediaType)
}

func TestContentMiddleware(t *testing.T) {
	testFunc := func(data interface{}, expectCode int) {
		expectContent := ""
		if c, ok := data.(*contentBody); ok {
			expectContent = c.Content
		}

		b, _ := json.Marshal(data)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
		req.Header.Set("CONTENT-TYPE", "application/json; charset=utf-8")
		w := httptest.NewRecorder()
		h := contentMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			readContent := req.Context().Value(keyContent).(string)
			if len(expectContent) > 0 && readContent != expectContent {
				t.Fatalf("Read failed, content was not same")
			}
		}))

		h.ServeHTTP(w, req)
		if w.Code != expectCode {
			t.Errorf("Unexpected code, data: %v, expect code: %d, but got %d", data, expectCode, w.Code)
		}
	}

	testFunc(&contentBody{"hello world"}, http.StatusOK)
	testFunc(&contentBody{`hello world
	new line
	1234`}, http.StatusOK)
	testFunc(&struct{ Content string }{"hello world"}, http.StatusOK)
	testFunc(&struct {
		Content string
		Code    int
	}{"hello world", 0}, http.StatusBadRequest)
	testFunc(nil, http.StatusBadRequest)
	testFunc(&struct{ ContentX string }{"hello world"}, http.StatusBadRequest)
	testFunc(&struct{}{}, http.StatusBadRequest)
}

func getFileName(fileDir, pathPrefix, pathName string) (string, error) {
	fileName := ""
	h := filePathMiddleware(fileDir, pathPrefix, (http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fileName = req.Context().Value(keyFileName).(string)
	})))
	r := httptest.NewRequest(http.MethodPost, pathName, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		return "", fmt.Errorf("FilePathMiddleware unexpected response, code: %d. (check pathPrefix and pathName)", w.Code)
	}
	return fileName, nil
}

func TestCreateFileHandler(t *testing.T) {
	const fileDir = "./files"
	const pathPrefix = "/"
	const pathName = "/test"

	fileName, err := getFileName(fileDir, pathPrefix, pathName)
	if err != nil {
		t.Fatal(err)
	}

	// Create file
	h := createFileHandler(fileDir, pathPrefix)
	c := `Hello world, A test
text with new line
3456`
	b, _ := json.Marshal(contentBody{c})
	r := httptest.NewRequest(http.MethodPost, pathName, bytes.NewReader(b))
	r.Header.Set("CONTENT-TYPE", jsonContentType)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("Create failed, response body: %s, code: %d", w.Body.String(), w.Code)
	}

	// Remove file in the end
	defer os.Remove(fileName)

	if fileInfo, err := os.Stat(fileName); err == nil {
		if fileInfo.IsDir() {
			t.Fatalf("Should be file, not folder, %s", fileName)
		} else if b, err := ioutil.ReadFile(fileName); err != nil {
			t.Fatalf("Read file failed, %v", err)
		} else if c != string(b) {
			t.Fatalf("Write file failed, content is not same, %s", fileName)
		}
	} else if err != nil {
		t.Fatalf("File stat error, %s, %v", fileName, err)
	}

	// Create file again, but file exists
	r = httptest.NewRequest(http.MethodPost, pathName, bytes.NewReader(b))
	r.Header.Set("CONTENT-TYPE", jsonContentType)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("Unexpected response, response body: %s, code: %d", w.Body.String(), w.Code)
	}
}

func TestModifyFileHandler(t *testing.T) {
	const fileDir = "./files"
	const pathPrefix = "/"
	const pathName = "/test"

	fileName, err := getFileName(fileDir, pathPrefix, pathName)
	if err != nil {
		t.Fatal(err)
	}

	// Modify file if file is not exsits
	{
		h := modifyFileHandler(fileDir, pathPrefix)
		b, _ := json.Marshal(contentBody{`Hello world, A test
		text with new line
		3456`})
		r := httptest.NewRequest(http.MethodPut, pathName, bytes.NewReader(b))
		r.Header.Set("CONTENT-TYPE", jsonContentType)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		if w.Code == http.StatusOK {
			t.Fatalf("Unexpected response, body: %s, code: %d", w.Body.String(), w.Code)
		}
	}

	// Create file
	if err := ioutil.WriteFile(fileName, ([]byte)("hello"), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fileName)

	// Modify file if file exsits
	{
		h := modifyFileHandler(fileDir, pathPrefix)
		s := `Hello world, A test
		text with new line
		3456`
		b, _ := json.Marshal(contentBody{s})
		r := httptest.NewRequest(http.MethodPut, pathName, bytes.NewReader(b))
		r.Header.Set("CONTENT-TYPE", jsonContentType)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("Unexpected response, body: %s, code: %d", w.Body.String(), w.Code)
		}

		if c, err := ioutil.ReadFile(fileName); err != nil {
			t.Fatal(err)
		} else if string(c) != s {
			t.Errorf("File content is not same, want: %s, got: %s", s, string(c))
		}
	}
}

func TestRemoveFileHandler(t *testing.T) {
	const fileDir = "./files"
	const pathPrefix = "/"
	const pathName = "/test"

	fileName, err := getFileName(fileDir, pathPrefix, pathName)
	if err != nil {
		t.Fatal(err)
	}

	// Remove file if file is not exsits
	{
		h := removeFileHandler(fileDir, pathPrefix)
		r := httptest.NewRequest(http.MethodDelete, pathName, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Fatalf("Unexpected response, body: %s, code: %d", w.Body.String(), w.Code)
		}
	}

	// Create file
	if err := ioutil.WriteFile(fileName, ([]byte)("hello"), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fileName)

	// Remove file if file exsits
	{
		h := removeFileHandler(fileDir, pathPrefix)
		r := httptest.NewRequest(http.MethodDelete, pathName, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("Unexpected response, body: %s, code: %d", w.Body.String(), w.Code)
		}

		if _, err := os.Stat(fileName); err == nil || os.IsExist(err) {
			t.Errorf("File remove failed, %s", fileName)
		}
	}
}

func TestRetrieveFileHandler(t *testing.T) {
	const fileDir = "./files"
	const pathPrefix = "/"
	const pathName = "/test"

	fileName, err := getFileName(fileDir, pathPrefix, pathName)
	if err != nil {
		t.Fatal(err)
	}

	// Retrieve file if file is not exsits
	{
		h := retrieveFileHandler(fileDir, pathPrefix)
		r := httptest.NewRequest(http.MethodGet, pathName, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Fatalf("Unexpected response, body: %s, code: %d", w.Body.String(), w.Code)
		}
	}

	// Create file
	data := `helloekmfvcx
	dcmdiew
	mv cx,m ie`
	if err := ioutil.WriteFile(fileName, ([]byte)(data), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fileName)

	// Retrieve file if file exsits
	{
		h := retrieveFileHandler(fileDir, pathPrefix)
		r := httptest.NewRequest(http.MethodGet, pathName, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("Unexpected response, body: %s, code: %d", w.Body.String(), w.Code)
		}

		c := contentBody{}
		if err := json.Unmarshal(w.Body.Bytes(), &c); err != nil {
			t.Fatal(err)
		}

		if c.Content != data {
			t.Errorf("Content is not same, %s, want: %s, got: %s", fileName, data, c.Content)
		}
	}
}
