# `internal/repository/README.md`

# Repository Package

This directory contains repository packages that define interfaces and GORM implementations for interacting with the database for core domain entities in the Go Internist medical AI system.

## Package Structure

```
repository/
├── chat/       # Chat repository interface + implementation
├── message/    # Message repository interface + implementation
├── user/       # User repository interface + implementation
```


## Design Principles

- **Domain-Centric:** Separate repository packages by domain (chat, message, user) ensure clear boundaries.
- **Interface Segregation:** Each package defines a focused interface for its domain operations.
- **Implementation Encapsulation:** GORM implementations are kept close to their interfaces promoting maintainability.
- **Production Ready:** Uses context-aware DB operations, structured error logging, and appropriate validation.
- **Scalable \& Modular:** Easy to extend with new repositories or swap underlying DB implementations if required.


## Usage

Import the relevant domain repository package and instantiate it with a GORM DB instance for use in service and handler layers.
