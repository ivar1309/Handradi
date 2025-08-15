package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gabriel-vasile/mimetype"
	"github.com/ivar1309/Handradi/internal/db"
)

var (
	storageRoot = "./storage"
)

// Middleware: API Key + CORS
func withAuthAndCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID := sanitizeClient(r.URL.Query().Get("client"))
		apiKey := r.Header.Get("x-api-key")

		if clientID == "" || apiKey == "" {
			http.Error(w, "Missing client or API key", http.StatusUnauthorized)
			return
		}

		allowedOrigin, err := db.CheckAuth(clientID, apiKey)

		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Set dynamic CORS headers
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, x-api-key")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Middleware: Public CORS
func withPublicCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set dynamic CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
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

	fileContent, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "could not read body", http.StatusBadRequest)
		return
	}

	filePath, err := saveFile(dir, filename, fileContent)
	if err != nil {
		http.Error(w, "Cannot create file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Upload: %v to %v", filename, dir)

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"message":"uploaded","path":"%s"}`, filePath)
}

func saveFile(dir string, filename string, content []byte) (string, error) {
	filePath := filepath.Join(dir, filename)
	out, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := out.Write(content); err != nil {
		return "", err
	}

	return filePath, nil
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

		log.Printf("Download: %v in changed dimensions -> %vx%v", filename, width, height)

		return
	}

	log.Printf("Download: %v in original dimensions", filename)

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

	log.Printf("Delete: %v", filename)

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

func presignHandler(w http.ResponseWriter, r *http.Request) {
	client := sanitizeClient(r.URL.Query().Get("client"))
	filename := filepath.Base(r.URL.Query().Get("filename"))

	if client == "" || filename == "" {
		http.Error(w, "client and filename required", http.StatusBadRequest)
		return
	}

	dir := filepath.Join(storageRoot, client)
	filePath := filepath.Join(dir, filename)
	expiresAt := time.Now().Add(5 * time.Minute).Unix()

	// signature: HMAC(secret, path|expires)
	mac := hmac.New(sha256.New, []byte(os.Getenv("PRESIGN_SECRET")))
	mac.Write([]byte(fmt.Sprintf("%s|%d", filePath, expiresAt)))
	sig := mac.Sum(nil)

	encodedPayload := base64.URLEncoding.EncodeToString([]byte(
		fmt.Sprintf("%s|%d|%s", filePath, expiresAt, sig),
	))

	presignedURL := fmt.Sprintf("/presignedupload?q=%s", encodedPayload)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"url": presignedURL,
	})
}

func presignedUploadHandler(w http.ResponseWriter, r *http.Request) {
	q, err := base64.URLEncoding.DecodeString(r.URL.Query().Get("q"))
	if err != nil {
		log.Printf("Could not decode q: %s\n", err.Error())
		http.Error(w, "Could not decode q", http.StatusBadRequest)
	}

	parts := strings.Split(string(q), "|")
	filePath := parts[0]
	expiresStr := parts[1]
	sigString := parts[2]

	expiresAt, _ := strconv.ParseInt(expiresStr, 10, 64)
	if time.Now().Unix() > expiresAt {
		log.Printf("URL expired: %d\n", expiresAt)
		http.Error(w, "URL expired", http.StatusUnauthorized)
		return
	}

	fileContent, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Could not read body: %s\n", err.Error())
		http.Error(w, "Could not read body", http.StatusBadRequest)
		return
	}

	mac := hmac.New(sha256.New, []byte(os.Getenv("PRESIGN_SECRET")))
	mac.Write([]byte(fmt.Sprintf("%s|%d", filePath, expiresAt)))
	expectedSig := mac.Sum(nil)
	sig := []byte(sigString)
	if !hmac.Equal([]byte(expectedSig), []byte(sig)) {
		log.Println("Invalid signature")
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	dir := filepath.Dir(filePath)
	filename := filepath.Base(filePath)
	savedFilePath, err := saveFile(dir, filename, fileContent)
	if err != nil {
		log.Printf("Cannot create file: %s\n", err.Error())
		http.Error(w, "Cannot create file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Upload: %v to %v", filename, dir)

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"message":"uploaded","path":"%s"}`, savedFilePath)
}

func main() {
	db.InitDB()
	defer db.Close()

	mux := http.NewServeMux()
	mux.Handle("/upload", withAuthAndCORS(http.HandlerFunc(uploadHandler)))
	mux.Handle("/delete", withAuthAndCORS(http.HandlerFunc(deleteHandler)))
	mux.Handle("/list", withAuthAndCORS(http.HandlerFunc(listHandler)))
	mux.Handle("/presignurl", withAuthAndCORS(http.HandlerFunc(presignHandler)))

	mux.Handle("/download", withPublicCORS(http.HandlerFunc(downloadHandler)))
	mux.Handle("/presignedupload", withPublicCORS(http.HandlerFunc(presignedUploadHandler)))

	port := "8888"
	if p, exists := os.LookupEnv("HANDRADI_PORT"); exists {
		port = p
	}

	log.Printf("📦 File server running on :%v", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
