# 📦 Handraði, a Simple File Storage Server & CLI

A lightweight Go-based file storage service with per-client isolation, API key authentication, and an accompanying CLI for user and file management.  
Designed to integrate easily with [MedusaJS](https://medusajs.com) or be used as a standalone file backend.

---

## ✨ Features

### Server
- **HTTP-based file storage API**
  - `POST /upload?client=<ID>&filename=<NAME>` — Upload a file for an authenticated client
  - `GET /download?client=<ID>&filename=<NAME>&width=<WIDTH>` — Serve stored files directly
    - optional parameters `width` & `height` trigger a server side resize of an image fetched.
    - if only one of them is set, then the image keeps the original aspect ratio.
  - `DELETE /upload?client=<ID>&filename=<NAME>` — Delete a file
  - all requests except for `/download` require `x-api-key` to be set. 
- **Client isolation** — Each client has its own directory
- **API key authentication** — Verify requests using `client_id` and `api_key`
- **Allowed Origin control** — Per-client CORS restriction
- **SQLite database** — Stores `client_id`, `api_key`, and `allowed_origin`
- **Server Port** — Set the environment variable `HANDRADI_PORT` to your desired port, default is `8888`.
---

### CLI
Manage the server’s SQLite user database from the command line:
- `cli add <client_id> <api_key> <allowed_origin>` — Create a new client
- `cli list` — View all registered clients
- `cli delete <client_id>` — Delete a client (clients files are not deleted)

---

## 🚀 Getting Started

### Prerequisites
- Go 1.24+
- (Optional) Docker

---

### Running Locally
1. Clone the repo:
   ```bash
   git clone https://github.com/ivar1309/Handradi.git
   cd Handradi
   ```
2. Run the server
   ```bash
   go run ./cmd/server/server.go
   ```
3. Run the cli
   ```bash
   go run ./cmd/cli/cli.go add user1 secureKEY http://example.com
   ```

### Running with Docker
```bash
wget https://github.com/ivar1309/Handradi/blob/main/compose.yaml
docker compose up -d
```