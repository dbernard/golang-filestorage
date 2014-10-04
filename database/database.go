package database

import (
	"strconv"
	"database/sql"
	_ "github.com/lib/pq"
)

// Pointer to our database
var database (*sql.DB)

// Initilaze the database that will contain the user JSON file contents
func InitializeDatabase() (error) {
	db, err := sql.Open("postgres", "postgres://csjuhxkfvajwiv:YdQEjG2cD5RTuluw2F6991RlOs@ec2-23-23-80-55.compute-1.amazonaws.com:5432/d3n2d68n0p67j2")
	if err != nil {
		return err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS files (filename TEXT, content TEXT)")
	if err != nil {
		return err
	}

	database = db
	return nil
}

// Check for any duplicate file names and return an altername name if necessary
func GetAvailableFilename(filename string) (string) {
	suffix := 1
	for {
		_, err := DatabaseFetch(filename + strconv.Itoa(suffix))
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
func DatabaseInsert(filename string, data string) (string, error) {
	// Check for existing filename in database.
	_, err := DatabaseFetch(filename)
	if err == nil {
		// We only want to execute this is there IS a duplicate. This is VERY
		// inefficient for large numbers of same-named files.
		filename = GetAvailableFilename(filename)
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
func DatabaseFetch(filename string) (string, error) {
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
func DatabaseRemove(filename string) (error) {
	_, err := database.Exec("DELETE FROM files WHERE filename=$1", filename)
	if err != nil {
		return err
	}

	return nil
}
