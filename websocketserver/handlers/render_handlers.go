package handlers

import (
	"html/template"
	"log"
	"net/http"
)

// ServeHome renders the main index page
func ServeHome(w http.ResponseWriter, r *http.Request) {
	log.Println("Parsing Files")
	tmpl := template.Must(template.ParseFiles("templates/index.html"))

	tmpl.Execute(w, nil)
}

// ServeDownload renders the download page
func ServeDownload(w http.ResponseWriter, r *http.Request) {
	log.Println("Parsing Files")
	tmpl := template.Must(template.ParseFiles("templates/download.html"))

	tmpl.Execute(w, nil)
}
