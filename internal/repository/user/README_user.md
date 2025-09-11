# `internal/repository/user/README.md`

# User Repository Package

This package handles all user-related data operations through the **UserRepository** interface and a GORM-backed implementation.

## Directory Contents

- `interface.go` — Defines the `UserRepository` interface.
- `gorm_user_repository.go` — GORM implementation of the user repository.


## Core Responsibilities

- User lifecycle management including creation, update, retrieval, and deletion.
- User authentication-related operations like failed attempts reset.
- Character balance tracking with get and update methods.
- Bulk retrieval of all users for administrative purposes.


## Interface Summary

- User creation, read by ID, username, phone.
- Updates, deletes, and special queries by username or phone with status.
- Management of account lockouts and character balance.


## Implementation Highlights

- Uses structured error handling with detailed logging.
- Enforces domain-specific logic such as balance updates and security reset.
- Optimized queries with GORM context and efficient scanning for balance retrieval.