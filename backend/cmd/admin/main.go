// Command admin performs local, operator-controlled admin recovery tasks.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ilhamnugraha8944/tauco/backend/internal/auth"
	authapp "github.com/ilhamnugraha8944/tauco/backend/internal/auth/application"
	authdomain "github.com/ilhamnugraha8944/tauco/backend/internal/auth/domain"
	authrepo "github.com/ilhamnugraha8944/tauco/backend/internal/auth/repository"
	"github.com/ilhamnugraha8944/tauco/backend/internal/platform/database"
	"golang.org/x/term"
)

func main() {
	if err := run(os.Args[1:], os.Stdin); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, input io.Reader) error {
	if len(args) == 0 {
		return errors.New("usage: admin <keygen|bootstrap|reset-password|reset-totp|revoke-sessions>")
	}
	if args[0] == "keygen" {
		if len(args) != 3 {
			return errors.New("usage: admin keygen <private-path> <public-path>")
		}
		return keygen(args[1], args[2])
	}
	if len(args) != 2 {
		return errors.New("usage: admin <bootstrap|reset-password|reset-totp|revoke-sessions> <email>")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	databaseConfig, err := database.LoadAdminRuntimeConfig(os.LookupEnv)
	if err != nil {
		return err
	}
	db, err := database.OpenAdminGORM(ctx, databaseConfig)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	runtime, err := auth.LoadRuntime(os.LookupEnv)
	if err != nil {
		return err
	}
	store, err := authrepo.NewPostgres(db)
	if err != nil {
		return err
	}
	service, err := authapp.NewService(store, runtime.Tokens, runtime.Secrets, runtime.RecoverySecret)
	if err != nil {
		return err
	}
	email := strings.ToLower(strings.TrimSpace(args[1]))
	switch args[0] {
	case "bootstrap", "reset-password":
		fmt.Fprint(os.Stderr, "Password (input tidak ditampilkan oleh command): ")
		password, err := readPassword(input)
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		password = strings.TrimRight(password, "\r\n")
		if args[0] == "bootstrap" {
			err = service.BootstrapAdmin(ctx, email, password)
		} else {
			err = service.ResetPassword(ctx, email, password)
		}
		return err
	case "reset-totp":
		return service.ResetTOTP(ctx, email)
	case "revoke-sessions":
		return service.RevokeSessions(ctx, email, "OPERATOR_REVOKE")
	default:
		return errors.New("unknown admin command")
	}
}

func readPassword(input io.Reader) (string, error) {
	if file, ok := input.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		password, err := term.ReadPassword(int(file.Fd()))
		fmt.Fprintln(os.Stderr)
		return string(password), err
	}
	return bufio.NewReader(input).ReadString('\n')
}

func keygen(privatePath, publicPath string) error {
	private, public, err := authdomain.GenerateRSAKeyPair()
	if err != nil {
		return err
	}
	privatePEM, publicPEM, err := authdomain.EncodeRSAKeyPair(private, public)
	if err != nil {
		return err
	}
	for _, path := range []string{privatePath, publicPath} {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
	}
	if err := os.WriteFile(privatePath, privatePEM, 0600); err != nil {
		return err
	}
	return os.WriteFile(publicPath, publicPEM, 0644)
}
