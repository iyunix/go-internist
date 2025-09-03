# Go Internist API

This project is a Go web application that replicates the core logic of the original Next.js/Prisma app. It provides user authentication, chat/message models, and uses SQLite as the database. The app exposes RESTful endpoints for user registration, login, chat creation, and message handling.

## Features
- User registration and login
- JWT-based authentication
- Chat and message management
- SQLite database using GORM
- RESTful API endpoints

## Tech Stack
- Go
- Gorilla Mux (router)
- GORM (ORM)
- SQLite

## Getting Started
1. Install Go (https://golang.org/dl/)
2. Install dependencies: `go mod tidy`
3. Run the app: `go run main.go`

## Endpoints
- `POST /register` - Register a new user
- `POST /login` - Login and receive JWT
- `GET /chats` - List user chats
- `POST /chats` - Create a new chat
- `GET /chats/{id}/messages` - List messages in a chat
- `POST /chats/{id}/messages` - Add a message to a chat

---
This is a starting point. Expand as needed for your requirements.
