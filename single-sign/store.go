package main

import (
	"context"
	"log"
)

// Store store
type Store struct{}

// ReadProperty read property
func (s Store) ReadProperty(ctx context.Context, key string) (string, error) {
	log.Println("ReadProperty", key)
	return "", nil
}

// WriteProperty write property
func (s Store) WriteProperty(ctx context.Context, key, value string) error {
	log.Println("WriteProperty", key, value)
	return nil
}
