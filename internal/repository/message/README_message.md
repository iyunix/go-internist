# `internal/repository/message/README.md`

# Message Repository Package

This package provides the **MessageRepository** interface and a GORM-based implementation to manage message data operations in the medical AI system.

## Directory Contents

- `interface.go` — Defines the `MessageRepository` interface.
- `message_repository.go` — Implements the GORM-based `gormMessageRepository`.


## Key Responsibilities

- Create new message records with content validation.
- Retrieve messages linked to a particular chat, ordered chronologically.


## Interface Summary

- `Create(ctx context.Context, message *domain.Message) (*domain.Message, error)`
- `FindByChatID(ctx context.Context, chatID uint) ([]domain.Message, error)`


## Implementation Highlights

- Uses GORM with context support for database operations.
- Proper error logging for failure scenarios.
- Validates message content to ensure non-empty messages before creation.