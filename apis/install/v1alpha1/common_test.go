package v1alpha1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePostgresConfig(t *testing.T) {
	type testCase struct {
		Name     string
		Config   *CommonAppConfig
		Expected error
	}

	testCases := []*testCase{
		{
			"All Good",
			&CommonAppConfig{
				Postgres: PostgresConfig{
					Connection: &PostgresConnection{
						Host:     "postgres",
						Port:     1337,
						User:     "postgres",
						Password: "password",
						DBName:   "postgres",
					},
				},
			},
			nil,
		},
		{
			"Postgres Config Absent",
			&CommonAppConfig{
				Postgres: PostgresConfig{},
			},
			fmt.Errorf("Postgres.Connection cannot be nil"),
		},
		{
			"Postgres Config Bad Port",
			&CommonAppConfig{
				Postgres: PostgresConfig{
					Connection: &PostgresConnection{
						Host:     "postgres",
						Port:     0,
						User:     "postgres",
						Password: "password",
						DBName:   "postgres",
					},
				},
			},
			fmt.Errorf("Invalid port"),
		},
		{
			"No Hostname",
			&CommonAppConfig{
				Postgres: PostgresConfig{
					Connection: &PostgresConnection{
						Host:     "",
						Port:     1337,
						User:     "postgres",
						Password: "password",
						DBName:   "postgres",
					},
				},
			},
			fmt.Errorf("No host specified for postgres"),
		},
		{
			"No Username",
			&CommonAppConfig{
				Postgres: PostgresConfig{
					Connection: &PostgresConnection{
						Host:     "postgres",
						Port:     1337,
						Password: "password",
						DBName:   "postgres",
					},
				},
			},
			fmt.Errorf("No user specified for postgres"),
		},
		{
			"No Password",
			&CommonAppConfig{
				Postgres: PostgresConfig{
					Connection: &PostgresConnection{
						Host:   "postgres",
						Port:   1337,
						User:   "postgres",
						DBName: "postgres",
					},
				},
			},
			fmt.Errorf("No password specified for postgres"),
		},
		{
			"No database name",
			&CommonAppConfig{
				Postgres: PostgresConfig{
					Connection: &PostgresConnection{
						Host:     "postgres",
						Port:     1337,
						User:     "postgres",
						Password: "password",
					},
				},
			},
			fmt.Errorf("No database name specified for postgres"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := validatePostgresConfig(tc.Config)
			if tc.Expected != nil {
				assert.Contains(t, result.Error(), tc.Expected.Error())
			}
		})
	}
}
