package main

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"fmt"
	"log"
	"os"
	"strings"
	"strconv"
	"encoding/base64"
	_ "github.com/lib/pq"
	"home/database"
	"html/template"
	"net/http"
)

// Pre compile templates
var templates = template.Must(template.ParseFiles("resources/upload.html"))

// Load a template to be displayed
func loadTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	templates.ExecuteTemplate(w, tmpl+".html", data)
}

// Execute the upload to database
func executeUpload(r *http.Request) (string, error) {
	// Execute this only if we have valid credentials
	success := BasicAuth(r)
	if !success {
		return "", errors.New("authorization failure")
	}

	// Filename used in download url later
	fn := ""

	// Processes request as a stream
	m := r.MultipartForm

	files := m.File["myfiles"]

	for i, _ := range files {
		file, err := files[i].Open()
		defer file.Close()
		if err != nil {
			return "", err
		}

		if !strings.HasSuffix(files[i].Filename, ".json") {
			return "", errors.New("invalid file type")
		}

		slurp, err := ioutil.ReadAll(file)
		if err != nil {
			return "", err
		}

		// Strip the .json off of the end of the filename
		name := strings.TrimSuffix(files[i].Filename, ".json")
		fn, err = database.DatabaseInsert(name, string(slurp))
		if err != nil {
			return "", err
		}
	}
	return fn, nil
}

// Handle upload requests
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	// GET request loads the upload form
	case "GET":
		

		loadTemplate(w, "upload", nil)

	// POST will upload the file
	case "POST":
		username := r.FormValue("username")
		password := r.FormValue("password")
		r.SetBasicAuth(username, password)
		fn, err := executeUpload(r)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fail_str := fmt.Sprintf("Failed: %s", err.Error())
			loadTemplate(w, "upload", fail_str)
			return
		}
		
		dl_url := fmt.Sprintf("/download/%s", fn)
		success_msg := fmt.Sprintf("Successfully uploaded! Visit (this URL)%s later to download your file.", dl_url)
		loadTemplate(w, "upload", success_msg)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// Handle a download request
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	url := r.URL.String()
	file := strings.TrimPrefix(url, "/download/")

	// Fetch the file from the database
	content, err := database.DatabaseFetch(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Initialize the download
	cont_disp := fmt.Sprintf("attachment; filename=%s", file)
	w.Header().Set("Content-Disposition", cont_disp)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))

	// Write the content to a buffer to prepare it for downloading
	buff := bytes.NewBufferString(content)
	io.Copy(w, buff)
}

// Parse basic authentication
func BasicAuth(r *http.Request) (bool) {
	if r.Header.Get("Authorization") == "" {
		return false
	}
	auth := strings.SplitN(r.Header["Authorization"][0], " ", 2)

	if len(auth) != 2 || auth[0] != "Basic" {
		return false
	}

	payload, _ := base64.StdEncoding.DecodeString(auth[1])
	pair := strings.SplitN(string(payload), ":", 2)

	if len(pair) != 2 || !Validate(pair[0], pair[1]) {
		return false
	}

	return true
}

// Validate the basic auth username and password
func Validate(username, password string) bool {
	if username == "user1" && password == "pass1" {
		return true
	}
	return false
}

func main() {
	err := database.InitializeDatabase()
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", uploadHandler)

	http.HandleFunc("/download/", downloadHandler)

	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		panic(err)
	}
}

