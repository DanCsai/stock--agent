package store

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

func newID(prefix string) string {
	return prefix + "_" + time.Now().UTC().Format("20060102150405.000000")
}

func nowString(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func randomID(prefix string) (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(buf), nil
}

func RandomUploadName(prefix, ext string) (string, error) {
	id, err := randomID(prefix)
	if err != nil {
		return "", err
	}
	return id + ext, nil
}
