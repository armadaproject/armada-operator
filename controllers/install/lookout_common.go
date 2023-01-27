package install

import "time"

// migrationTimeout is how long we'll wait for the Lookout db migration job
const (
	migrationTimeout   = time.Second * 120
	migrationPollSleep = time.Second * 5
)

type PostgresConfig struct {
	Connection ConnectionConfig
}

type ConnectionConfig struct {
	Host string
	Port string
}
