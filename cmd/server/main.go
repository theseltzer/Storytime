package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"mime"
	"net/http"
	"os"

	// Blank import: we never call this package by name, we only need its
	// init() to register itself with database/sql under the name "pgx".
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/theseltzer/storytime/internal/spot"
)

const addr = ":8080"

// cvPage is everything templates/cv.html needs. A template can only reach the
// one value you hand it, so the page-wide language and the rows travel together
// in a single struct.
type cvPage struct {
	Lang  string // "en" or "tr"
	Spots []spot.Spot
}

func main() {
	// The browser refuses to stream-compile a .wasm file that isn't served as
	// application/wasm. Go's table already knows this extension, but the mime
	// package also reads system files that can override it, so pin it.
	if err := mime.AddExtensionType(".wasm", "application/wasm"); err != nil {
		log.Fatal(err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// sql.Open does not connect, it only parses the URL and prepares a pool.
	// Ping forces a real connection, so a bad URL or a dead server fails here
	// at startup instead of inside a request handler later.
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	log.Print("connected to the database")

	tmpl := template.Must(template.ParseFiles("templates/cv.html"))

	mux := http.NewServeMux()
	mux.Handle("GET /api/spots", handleSpots(db))

	// One route per language rather than sniffing Accept-Language: a crawler
	// indexes a URL, not a header, and a shared link has to open the same page
	// for the person receiving it.
	mux.Handle("GET /cv", handleCV(db, tmpl, "en"))
	mux.Handle("GET /cv/tr", handleCV(db, tmpl, "tr"))

	// Least specific pattern, so it only catches what the routes above did not.
	mux.Handle("/", noCache(http.FileServer(http.Dir("web"))))

	log.Printf("Story_time listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// getSpots reads every spot from the database and knows nothing about HTTP.
// Two different pages need the same rows — the JSON API and /cv — and a
// function that only speaks data can serve both without either one copying
// the query. ORDER BY id keeps the order stable across calls.
func getSpots(db *sql.DB) ([]spot.Spot, error) {
	rows, err := db.Query("SELECT id, x, y, radius, title_en, body_en, title_tr, body_tr FROM spots ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	// Not nil: a nil slice marshals to null, an empty one to [].
	spots := []spot.Spot{}
	for rows.Next() {
		var s spot.Spot
		if err := rows.Scan(&s.ID, &s.X, &s.Y, &s.Radius, &s.TitleEN, &s.BodyEN, &s.TitleTR, &s.BodyTR); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		spots = append(spots, s)
	}

	// rows.Next() returns false both when the rows ran out and when the
	// connection broke halfway. Only rows.Err() tells the two apart.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return spots, nil
}

func handleSpots(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		spots, err := getSpots(db)
		if err != nil {
			// The detail goes to the log; the client gets a vague sentence,
			// because a database error message can leak table and column names.
			log.Printf("api spots: %v", err)
			http.Error(w, "something went wrong", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		// The status code is already on the wire by now, so a failure here
		// cannot become a 500 for the client. Logging is all that is left.
		if err := json.NewEncoder(w).Encode(spots); err != nil {
			log.Printf("json encode error: %v", err)
		}
	}
}

// handleCV renders the same rows as /api/spots into plain HTML, because a
// <canvas> is invisible to crawlers, ATS parsers, and anyone skimming on a
// phone. Same store, second renderer.
func handleCV(db *sql.DB, tmpl *template.Template, lang string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		spots, err := getSpots(db)
		if err != nil {
			log.Printf("cv page: %v", err)
			http.Error(w, "something went wrong", http.StatusInternalServerError)
			return
		}

		// Render into memory first. Execute writes straight to w, so a template
		// failure halfway through would leave a 200 and half a page already on
		// the wire, with no way left to turn it into a 500.
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, cvPage{Lang: lang, Spots: spots}); err != nil {
			log.Printf("cv render: %v", err)
			http.Error(w, "something went wrong", http.StatusInternalServerError)
			return
		}

		// Go would sniff this from the bytes anyway, but sniffing guesses and
		// an explicit header does not.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := buf.WriteTo(w); err != nil {
			log.Printf("cv write: %v", err)
		}
	}
}

// noCache stops the browser from serving a stale game.wasm after a rebuild.
// Drop this once the site is deployed for real.
func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		h.ServeHTTP(w, r)
	})
}
