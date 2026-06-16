package main

import (
    "fmt"
    "github.com/lookitval/nabu/backend/internal/v1"
)

func main() {
    router := v1.SetupRouter()
    fmt.Println("Initializing API server on port 8080...")
    router.Run(":8080")
}