package ipam

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/jmoiron/sqlx"

	// import for sqlx to use postgres driver
	_ "github.com/lib/pq"
)

const postgresSchema = `
ALTER TABLE IF EXISTS prefixes RENAME TO prefixes_root;
CREATE TABLE IF NOT EXISTS prefixes_root (
	cidr   text PRIMARY KEY NOT NULL,
	prefix JSONB
);
CREATE INDEX IF NOT EXISTS prefix_idx ON prefixes_root USING GIN(prefix);
`

// SSLMode specifies how to configure ssl encryption to the database
type SSLMode string

func (s SSLMode) String() string {
	return "sslmode=" + string(s)
}

const (
	// SSLModeAllow I don't care about security
	// but I will pay the overhead of encryption if the server insists on it
	SSLModeAllow = SSLMode("allow")
	// SSLModeDisable I don't care about security
	// and I don't want to pay the overhead of encryption.
	SSLModeDisable = SSLMode("disable")
	// SSLModePrefer I don't care about encryption
	// but I wish to pay the overhead of encryption if the server supports it.
	SSLModePrefer = SSLMode("prefer")
	// SSLModeRequire I want my data to be encrypted and I accept the overhead.
	// I trust that the network will make sure I always connect to the server I want.
	SSLModeRequire = SSLMode("require")
	// SSLModeVerifyCA I want my data encrypted and I accept the overhead.
	// I want to be sure that I connect to a server that I trust.
	SSLModeVerifyCA = SSLMode("verify-ca")
	// SSLModeVerifyFull I want my data encrypted and I accept the overhead.
	// I want to be sure that I connect to a server I trust, and that it's the one I specify.
	SSLModeVerifyFull = SSLMode("verify-full")
)

// NewPostgresStorage creates a new Storage which uses postgres.
func NewPostgresStorage(host, port, user, password, dbname string, sslmode SSLMode) (Storage, error) {
	return newPostgres(host, port, user, password, dbname, sslmode)
}

func newPostgres(host, port, user, password, dbname string, sslmode SSLMode) (*sql, error) {
	ds, err := dataSource(host, port, user, password, dbname, sslmode)
	if err != nil {
		// Already wrapped.
		return nil, err
	}

	db, err := sqlx.Connect("postgres", ds)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database:%w", err)
	}
	db.MustExec(postgresSchema)
	sql := &sql{
		db:     db,
		tables: sync.Map{},
	}
	sql.tables.Store(defaultNamespace, struct{}{})
	return sql, nil
}

func dataSource(host, port, user, password, dbname string, sslmode SSLMode) (string, error) {
	baseURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?%s", url.PathEscape(user), url.PathEscape(password), host, port, dbname, sslmode)
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("%w: unable to parse base URL:%s", err, baseURL)
	}

	return parsedURL.String(), nil
}
