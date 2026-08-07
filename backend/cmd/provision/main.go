package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/database"
	"golang.org/x/term"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if os.Getenv("DATABASE_DEPLOYMENT_PROFILE") != string(database.ProfileSupabase) {
		return errors.New("DATABASE_DEPLOYMENT_PROFILE must be supabase")
	}
	rawURL := os.Getenv("PROVISION_DATABASE_URL")
	if rawURL == "" {
		return errors.New("PROVISION_DATABASE_URL is required")
	}
	passwords := make([]string, 3)
	labels := []string{"migration", "public runtime", "admin runtime"}
	for index, label := range labels {
		value, err := readPassword(label)
		if err != nil {
			return err
		}
		passwords[index] = value
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return database.ProvisionManagedRoles(ctx, rawURL, database.ManagedPasswords{
		Migration: passwords[0], Runtime: passwords[1], Admin: passwords[2],
	})
}

func readPassword(label string) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", errors.New("managed provisioning requires an interactive terminal")
	}
	_, _ = fmt.Fprintf(os.Stderr, "%s password (input hidden): ", label)
	value, err := term.ReadPassword(fd)
	_, _ = fmt.Fprintln(os.Stderr)
	return string(value), err
}
