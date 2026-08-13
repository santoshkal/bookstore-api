package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
)

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

var (
	db  *sql.DB
	tpl *template.Template
)

type Books struct {
	Isbn   string
	Title  string
	Author string
	Price  float32
}

func main() {
	var err error
	host := strings.TrimSpace(getEnv("POSTGRES_HOST", "localhost"))
	// Handle misreading IPV6 like, ::1 to be interpreted as post:1
	if _, _, err := net.SplitHostPort(host); err != nil {
		host = net.JoinHostPort(host, getEnv("POSTGRES_PORT", "5432"))
	}
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(getEnv("POSTGRES_USER", "santosh"), getEnv("POSTGRES_PASSWORD", "dts123")),
		Host:   host,
		Path:   getEnv("POSTGRES_DB", "bookstore"),
	}
	q := u.Query()
	q.Set("sslmode", getEnv("POSTGRES_SSLMODE", "disable"))
	u.RawQuery = q.Encode()

	db, err = sql.Open("postgres", u.String())
	if err != nil {
		fmt.Printf("Connecting to the Database %s", err)
	}
	if err = db.Ping(); err != nil {
		panic(err)
	}
	fmt.Println("You are connected to the database...")

	tpl = template.Must(template.ParseGlob("templates/*.gohtml"))
	http.HandleFunc("/", index)
	http.HandleFunc("/books", booksIndex)
	http.HandleFunc("/books/show", booksShow)
	http.HandleFunc("/books/create", booksCreateForm)
	http.HandleFunc("/books/create/process", booksCreateProcess)
	http.HandleFunc("/books/update", booksUpdateForm)
	http.HandleFunc("/books/update/process", booksUpdateProcess)
	http.HandleFunc("/books/delete/process", booksDeleteProcess)
	http.Handle("/image/", http.StripPrefix("/image", http.FileServer(http.Dir("./image"))))
	http.ListenAndServe(":8080", nil)
}

// func getCommitSHA() (string, error) {
// 	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
// 	output, err := cmd.Output()
// 	if err != nil {
// 		return "", err
// 	}
// 	// fmt.Printf("OUTPUT: %v", output)

// 	commitSHA := strings.TrimSpace(string(output))
// 	fmt.Printf("COMMITSHA: %v", commitSHA)

// 	return commitSHA, nil
// }

func index(w http.ResponseWriter, r *http.Request) {
	commitSHA := os.Getenv("COMMIT_SHA")
	fmt.Printf("CommitSHA: %v", commitSHA)

	data := struct {
		CommitSHA string
	}{
		CommitSHA: commitSHA,
	}
	tpl.ExecuteTemplate(w, "index.gohtml", data)

	http.Redirect(w, r, "/books", http.StatusSeeOther)
}

func booksIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.Query("SELECT * FROM books")
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}
	defer rows.Close()

	bks := make([]Books, 0)
	for rows.Next() {
		bk := Books{}
		err := rows.Scan(&bk.Isbn, &bk.Title, &bk.Author, &bk.Price)
		if err != nil {
			http.Error(w, http.StatusText(500), 500)
			return
		}
		bks = append(bks, bk)
	}
	if err = rows.Err(); err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	tpl.ExecuteTemplate(w, "books.gohtml", bks)
}

func booksShow(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
		return
	}

	isbn := r.FormValue("isbn")
	if isbn == "" {
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
		return
	}

	row := db.QueryRow("SELECT * FROM books WHERE isbn = $1", isbn)

	bk := Books{}
	err := row.Scan(&bk.Isbn, &bk.Title, &bk.Author, &bk.Price)
	switch {
	case err == sql.ErrNoRows:
		http.NotFound(w, r)
		return
	case err != nil:
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		return
	}

	tpl.ExecuteTemplate(w, "show.gohtml", bk)
}

func booksCreateForm(w http.ResponseWriter, r *http.Request) {
	tpl.ExecuteTemplate(w, "create.gohtml", nil)
}

func booksCreateProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
		return
	}

	// get form values
	bk := Books{}
	bk.Isbn = r.FormValue("isbn")
	bk.Title = r.FormValue("title")
	bk.Author = r.FormValue("author")
	p := r.FormValue("price")

	// validate form values
	if bk.Isbn == "" || bk.Title == "" || bk.Author == "" || p == "" {
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
		return
	}

	// convert form values
	f64, err := strconv.ParseFloat(p, 32)
	if err != nil {
		http.Error(w, http.StatusText(406)+"Please hit back and enter a number for the price", http.StatusNotAcceptable)
		return
	}
	bk.Price = float32(f64)

	// insert values
	_, err = db.Exec("INSERT INTO books (isbn, title, author, price) VALUES ($1, $2, $3, $4)", bk.Isbn, bk.Title, bk.Author, bk.Price)
	if err != nil {
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		return
	}

	// confirm insertion
	tpl.ExecuteTemplate(w, "created.gohtml", bk)
}

func booksUpdateForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
		return
	}

	isbn := r.FormValue("isbn")
	if isbn == "" {
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
		return
	}

	row := db.QueryRow("SELECT * FROM books WHERE isbn = $1", isbn)

	bk := Books{}
	err := row.Scan(&bk.Isbn, &bk.Title, &bk.Author, &bk.Price)
	switch {
	case err == sql.ErrNoRows:
		http.NotFound(w, r)
		return
	case err != nil:
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		return
	}
	tpl.ExecuteTemplate(w, "update.gohtml", bk)
}

func booksUpdateProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
		return
	}

	// get form values
	bk := Books{}
	bk.Isbn = r.FormValue("isbn")
	bk.Title = r.FormValue("title")
	bk.Author = r.FormValue("author")
	p := r.FormValue("price")

	// validate form values
	if bk.Isbn == "" || bk.Title == "" || bk.Author == "" || p == "" {
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
		return
	}

	// convert form values
	f64, err := strconv.ParseFloat(p, 32)
	if err != nil {
		http.Error(w, http.StatusText(406)+"Please hit back and enter a number for the price", http.StatusNotAcceptable)
		return
	}
	bk.Price = float32(f64)

	// insert values
	_, err = db.Exec("UPDATE books SET isbn = $1, title=$2, author=$3, price=$4 WHERE isbn=$1;", bk.Isbn, bk.Title, bk.Author, bk.Price)
	if err != nil {
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		return
	}

	// confirm insertion
	tpl.ExecuteTemplate(w, "updated.gohtml", bk)
}

func booksDeleteProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
		return
	}

	isbn := r.FormValue("isbn")
	if isbn == "" {
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
		return
	}

	// delete book
	_, err := db.Exec("DELETE FROM books WHERE isbn=$1;", isbn)
	if err != nil {
		http.Error(w, http.StatusText(500), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/books", http.StatusSeeOther)
}
