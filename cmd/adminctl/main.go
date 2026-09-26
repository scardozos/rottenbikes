// adminctl manages poster roles (admin promotion/demotion) out-of-band.
// Admin identity then lives only in the database — never in git, env files,
// or k8s manifests. The role is read by the API on every token lookup, so
// promote/demote takes effect immediately, without a restart or redeploy.
//
// Usage:
//
//	adminctl promote <email-or-username>   grant the admin role
//	adminctl demote  <email-or-username>   revoke the admin role
//	adminctl list                          list all admins
//
// It uses the same database configuration as the API (DATABASE_URL, or the
// DB_* variables).
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"

	"github.com/scardozos/rottenbikes/internal/dbconfig"
	"github.com/scardozos/rottenbikes/internal/domain"
)

const usage = `usage:
  adminctl promote <email-or-username>   grant the admin role
  adminctl demote  <email-or-username>   revoke the admin role
  adminctl list                          list all admins`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}

	cmd := os.Args[1]

	switch {
	case cmd == "promote" && len(os.Args) == 3:
		if err := runSetRole(os.Args[2], domain.PosterRoleAdmin); err != nil {
			fatal("promote", err)
		}
	case cmd == "demote" && len(os.Args) == 3:
		if err := runSetRole(os.Args[2], domain.PosterRoleUser); err != nil {
			fatal("demote", err)
		}
	case cmd == "list" && len(os.Args) == 2:
		if err := runList(); err != nil {
			fatal("list admins", err)
		}
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
}

func openStore() (*domain.Store, *sql.DB, error) {
	db, err := sql.Open("postgres", dbconfig.DSN())
	if err != nil {
		return nil, nil, fmt.Errorf("open db: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("ping db: %w", err)
	}

	return domain.NewStore(db), db, nil
}

func runSetRole(identifier string, role domain.PosterRole) error {
	store, db, err := openStore()
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p, err := store.SetPosterRole(ctx, identifier, role)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no poster matches %q (email or username)", identifier)
		}
		return err
	}

	fmt.Printf("%s (%s) is now %s\n", p.Username, p.Email, p.Role)
	return nil
}

func runList() error {
	store, db, err := openStore()
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	admins, err := store.ListAdmins(ctx)
	if err != nil {
		return err
	}

	if len(admins) == 0 {
		fmt.Println("no admins — promote one with: adminctl promote <email-or-username>")
		return nil
	}

	for _, a := range admins {
		fmt.Printf("%d\t%s\t%s\t(since %s)\n", a.PosterID, a.Username, a.Email, a.CreatedAt.Format(time.RFC3339))
	}
	return nil
}

func fatal(action string, err error) {
	fmt.Fprintf(os.Stderr, "adminctl: %s: %v\n", action, err)
	os.Exit(1)
}
