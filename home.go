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
	"database/sql"
	"encoding/base64"
	_ "github.com/lib/pq"
	"html/template"
	"net/http"
)

// Pre compile templates
var templates = template.Must(template.ParseFiles("resources/upload.html"))

// Global databse variable for manipulation
var database (*sql.DB)

// Load a template to be displayed
func loadTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	templates.ExecuteTemplate(w, tmpl+".html", data)
}

// TODO: Move DB code into its own file!

// Initilaze the database that will contain the user JSON file contents
func initializeDatabase() (*sql.DB, error) {
	db, err := sql.Open("postgres", "YOUR-DATABASE-URL-HERE")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS files (filename TEXT, content TEXT)")
	if err != nil {
		return nil, err
	}

	return db, nil
}

func getAvailableFilename(filename string) (string) {
	suffix := 1
	for {
		_, err := databaseFetch(filename + strconv.Itoa(suffix))
		if err == nil {
			// Already a file with that name, retry with an incrementing suffix
			suffix++
		} else {
			filename = filename + strconv.Itoa(suffix)
			break
		}
	}
	return filename
}

// Handle inserting into a databse
func databaseInsert(filename string, data string) (string, error) {
	// Check for existing filename in database.
	_, err := databaseFetch(filename)
	if err == nil {
		// We only want to execute this is there IS a duplicate. This is VERY
		// inefficient for large numbers of same-named files.
		filename = getAvailableFilename(filename)
	}

	tx, err := database.Begin()
	if err != nil {
		return "", err
	}

	_, err = tx.Exec("INSERT INTO files VALUES ($1, $2)", filename, data)
	if err != nil {
		return "", err
	}

	tx.Commit()

	return filename, nil
}

// Handle retriving info from database
func databaseFetch(filename string) (string, error) {
	stmt, err := database.Prepare("SELECT content FROM files WHERE filename=$1")
	if err != nil {
		return "", err
	}
	
	defer stmt.Close()

	var data string
	err = stmt.QueryRow(filename).Scan(&data)
	if err != nil {
		return "", err
	}

	return data, nil
}

// Handle removal from a database
// (when should this be done? automated?)
func databaseRemove(filename string) (error) {
	_, err := database.Exec("DELETE FROM files WHERE filename=$1", filename)
	if err != nil {
		return err
	}

	return nil
}

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
		fn, err = databaseInsert(name, string(slurp))
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

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	url := r.URL.String()
	file := strings.TrimPrefix(url, "/download/")

	// Fetch the file from the database
	// TODO: Basic HTTP auth
	content, err := databaseFetch(file)
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

func Validate(username, password string) bool {
	if username == "user1" && password == "pass1" {
		return true
	}
	return false
}

func main() {
	db, err := initializeDatabase()
	if err != nil {
		// TODO: How should we handle errors here?
		log.Fatal(err)
	}
	
	// Set the global database variable for later use
	if db == nil {
		// TODO: Again, how can we handle this error
		log.Fatal("No database returned")
	}

	database = db

	http.HandleFunc("/", uploadHandler)

	http.HandleFunc("/download/", downloadHandler)

	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		panic(err)
	}
}

