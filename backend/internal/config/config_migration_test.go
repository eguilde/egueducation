package config

import (
	"strings"
	"testing"
)

func TestAutoMigrateOnStartupIsDisabledOnlyInProduction(t *testing.T) {
	tests := []struct {
		environment string
		want        bool
	}{
		{environment: "development", want: true},
		{environment: "test", want: true},
		{environment: "production", want: false},
		{environment: "prod", want: false},
	}
	for _, test := range tests {
		t.Run(test.environment, func(t *testing.T) {
			if got := (Config{Environment: test.environment}).AutoMigrateOnStartup(); got != test.want {
				t.Fatalf("AutoMigrateOnStartup() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestMigrationURLFailsClosedInProductionAndFallsBackOnlyOutsideProduction(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
		err  string
	}{
		{
			name: "production requires dedicated URL",
			cfg:  Config{Environment: "production", DatabaseURL: "postgres://runtime"},
			err:  "MIGRATION_DATABASE_URL",
		},
		{
			name: "production uses dedicated URL",
			cfg:  Config{Environment: "production", DatabaseURL: "postgres://runtime", MigrationDatabaseURL: "postgres://migrator"},
			want: "postgres://migrator",
		},
		{
			name: "test falls back to runtime URL",
			cfg:  Config{Environment: "test", DatabaseURL: "postgres://runtime"},
			want: "postgres://runtime",
		},
		{
			name: "non-production still requires a URL",
			cfg:  Config{Environment: "development"},
			err:  "DATABASE_URL",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.cfg.MigrationURL()
			if test.err != "" {
				if err == nil || !strings.Contains(err.Error(), test.err) {
					t.Fatalf("MigrationURL() error = %v, want %q", err, test.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("MigrationURL() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("MigrationURL() = %q, want %q", got, test.want)
			}
		})
	}
}
