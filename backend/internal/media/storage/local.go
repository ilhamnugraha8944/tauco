package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrObjectConflict = errors.New("object key already contains different data")

type Local struct {
	root string
}

func NewLocal(root string) (*Local, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("local media root is required")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve local media root: %w", err)
	}
	if err := os.MkdirAll(absolute, 0o700); err != nil {
		return nil, fmt.Errorf("create local media root: %w", err)
	}
	return &Local{root: absolute}, nil
}

func (store *Local) PutIfAbsent(ctx context.Context, key, _ string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := store.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create object directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		existing, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read existing object: %w", readErr)
		}
		if !bytes.Equal(existing, data) {
			return ErrObjectConflict
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("create object: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return fmt.Errorf("write object: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("close object: %w", err)
	}
	return nil
}

func (store *Local) Get(ctx context.Context, key string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path, err := store.path(key)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read object: %w", err)
	}
	return data, nil
}

func (store *Local) path(key string) (string, error) {
	if strings.HasPrefix(key, "/") || strings.HasPrefix(key, "\\") {
		return "", errors.New("invalid object key")
	}
	clean := filepath.Clean(filepath.FromSlash(key))
	if key == "" || filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("invalid object key")
	}
	path := filepath.Join(store.root, clean)
	relative, err := filepath.Rel(store.root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("object key escapes storage root")
	}
	return path, nil
}
