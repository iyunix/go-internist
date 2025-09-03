package main

import (
    "embed"
    "net/http"
)

//go:embed web/static/*
var staticFS embed.FS
