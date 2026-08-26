// scripts/debug-load/main.go — quick loader smoke test.
package main

import (
	"encoding/json"
	"fmt"

	"github.com/nuteo/nuteo-web/internal/storage"
)

func main() {
	store := storage.New()
	if err := store.LoadAll("/root/projects/nuteo-web/content"); err != nil {
		fmt.Println("ERR:", err)
		return
	}
	fmt.Println("services loaded:", len(store.Services))
	for _, s := range store.Services {
		j, _ := json.Marshal(s)
		fmt.Printf("  - %s\n", j)
	}
	fmt.Println()
	s := store.ServiceBySlug("cloud-migration")
	if s == nil {
		fmt.Println("LOOKUP: nil")
	} else {
		fmt.Printf("LOOKUP: %s\n", s.Title)
	}
}
