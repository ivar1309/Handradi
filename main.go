package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/gabriel-vasile/mimetype"
)

var (
	storageRoot = "./storage"
	validAPIKey = "secret123" // change in production!
)

// Middleware: API Key + CORS
func withAuthAndCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, x-api-key")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// API key check
		if r.Header.Get("x-api-key") != validAPIKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Sanitization for client IDs
func sanitizeClient(client string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' || r == '-' {
			return r
		}
		return -1
	}, client)
}

// Upload
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	client := sanitizeClient(r.URL.Query().Get("client"))
	filename := filepath.Base(r.URL.Query().Get("filename"))

	if client == "" || filename == "" {
		http.Error(w, "client and filename required", http.StatusBadRequest)
		return
	}

	dir := filepath.Join(storageRoot, client)
	if err := os.MkdirAll(dir, 0755); err != nil {
		http.Error(w, "Cannot create storage dir: "+err.Error(), http.StatusInternalServerError)
		return
	}

	filePath := filepath.Join(dir, filename)
	out, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Cannot create file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer out.Close()

	if _, err := out.ReadFrom(r.Body); err != nil {
		http.Error(w, "Write failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"message":"uploaded","path":"%s"}`, filePath)
}

// Download + optional resize
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	client := sanitizeClient(r.URL.Query().Get("client"))
	filename := filepath.Base(r.URL.Query().Get("filename"))

	if client == "" || filename == "" {
		http.Error(w, "client and filename required", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(storageRoot, client, filename)

	// Detect mime
	mime, _ := mimetype.DetectFile(filePath)
	w.Header().Set("Content-Type", mime.String())

	// Optional resizing: /download?...&width=300&height=200
	widthStr := r.URL.Query().Get("width")
	heightStr := r.URL.Query().Get("height")

	if widthStr != "" || heightStr != "" {
		img, err := imaging.Open(filePath)
		if err != nil {
			http.Error(w, "Cannot open image: "+err.Error(), http.StatusInternalServerError)
			return
		}

		width := 0
		height := 0
		if widthStr != "" {
			fmt.Sscanf(widthStr, "%d", &width)
		}
		if heightStr != "" {
			fmt.Sscanf(heightStr, "%d", &height)
		}

		resized := imaging.Resize(img, width, height, imaging.Lanczos)
		w.Header().Set("Content-Type", "image/png")
		imaging.Encode(w, resized, imaging.PNG)
		return
	}

	http.ServeFile(w, r, filePath)
}

// Delete
func deleteHandler(w http.ResponseWriter, r *http.Request) {
	client := sanitizeClient(r.URL.Query().Get("client"))
	filename := filepath.Base(r.URL.Query().Get("filename"))

	if client == "" || filename == "" {
		http.Error(w, "client and filename required", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(storageRoot, client, filename)
	if err := os.Remove(filePath); err != nil {
		http.Error(w, "Delete failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, `{"message":"deleted"}`)
}

// List
func listHandler(w http.ResponseWriter, r *http.Request) {
	client := sanitizeClient(r.URL.Query().Get("client"))
	if client == "" {
		http.Error(w, "client required", http.StatusBadRequest)
		return
	}

	dir := filepath.Join(storageRoot, client)
	files, err := os.ReadDir(dir)
	if err != nil {
		http.Error(w, "Cannot read dir: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var out []string
	for _, f := range files {
		if !f.IsDir() {
			out = append(out, f.Name())
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("/upload", withAuthAndCORS(http.HandlerFunc(uploadHandler)))
	mux.Handle("/download", withAuthAndCORS(http.HandlerFunc(downloadHandler)))
	mux.Handle("/delete", withAuthAndCORS(http.HandlerFunc(deleteHandler)))
	mux.Handle("/list", withAuthAndCORS(http.HandlerFunc(listHandler)))

	log.Println("📦 File server running on :8888")
	log.Fatal(http.ListenAndServe(":8888", mux))
}
