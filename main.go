package main

import (
	"encoding/base64"
	"encoding/json"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

// This is our database "connection"
var (
	database *gorm.DB
)

// These are our templates
var (
	displayTmpl = template.Must(template.ParseFiles("templates/display.html"))
)

// Paste represents a paste made by a user (Anonymous or by an authenticated user)
type Paste struct {
	ID      uint   `json:"id" gorm:"primary_key;auto_increment"`
	PasteID string `json:"pasteId"`
	// CreatedAt is the time that the paste was created at
	CreatedAt time.Time `json:"createdAt"`
	// Content is the content of the paste
	Content string `json:"content"`
	// Language is the programming language the paste is written in
	Language string `json:"language,omitempty"`
}

// NewPaste creates a new paste
func NewPaste(content, language string) *Paste {
	return &Paste{
		PasteID:  generateID(),
		Content:  content,
		Language: language,
	}
}

func init() {
	db, err := gorm.Open("sqlite3", "pastes.db")

	if err != nil {
		log.Fatalln(err)
	}

	database = db
	// Create the table for the pase
	db.AutoMigrate(&Paste{})
}

func main() {
	// Close the database when we are done
	defer database.Close()
	router := mux.NewRouter()

	// Our routes
	router.Path("/{pid:[\\w\\-]+}").
		Methods("GET").
		HandlerFunc(displayHandler)

	router.Path("/api/paste").
		Methods("POST").
		HandlerFunc(pasteHandler)

	log.Fatal(http.ListenAndServe(":8080", router))
}

func displayHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	if exists(vars["pid"]) {
		var res Paste
		database.Model(&Paste{}).Where("paste_id = ?", vars["pid"]).Scan(&res)
		displayTmpl.Execute(w, &res)
	} else {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}
}

func pasteHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(int64(1 << 21)) // 2MB

	if err != nil {
		log.Panicln(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	text := r.FormValue("text")
	lang := r.FormValue("language")

	p := NewPaste(text, lang)
	database.Save(p)

	d, err := json.Marshal(p)

	if err != nil {
		log.Panicln(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(d)
}

func generateID() string {
	rnd := [10]byte{}
	rand.Seed(time.Now().UnixNano())
	rand.Read(rnd[:])
	id := base64.RawURLEncoding.EncodeToString(rnd[:])

	// attempt to recreate if it exists
	for exists(id) {
		rnd = [10]byte{}
		rand.Seed(time.Now().UnixNano())
		rand.Read(rnd[:])
		id = base64.RawURLEncoding.EncodeToString(rnd[:])
	}

	return id
}

// checks if a paste id exists
func exists(id string) bool {
	var count uint
	database.Model(&Paste{}).Where("paste_id = ?", id).Count(&count)
	return count > 0
}
