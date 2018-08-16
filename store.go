package nsrecorder // import "jw4.us/nsrecorder"

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store interface {
	Accept([]Client, []Lookup) error
}

type Client struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
}

type Lookup struct {
	When    time.Time `json:"when"`
	Client  string    `json:"client"`
	Host    string    `json:"host"`
	Type    string    `json:"type"`
	FirstIP string    `json:"first_ip"`
	AllIPs  []string  `json:"all_ips"`
}

func MultiStore(stores ...Store) Store {
	return multiStore(stores)
}

type multiStore []Store

func (s multiStore) Accept(clients []Client, lookups []Lookup) error {
	for _, store := range s {
		if err := store.Accept(clients, lookups); err != nil {
			return err
		}
	}
	return nil
}

func NewLogStore() Store {
	return &logStore{}
}

type logStore struct{}

func (*logStore) Accept(clients []Client, lookups []Lookup) error {
	var b strings.Builder
	fmt.Fprintf(&b, "\nACCEPT %d clients, %d lookups\n", len(clients), len(lookups))
	clientSet := map[string]string{}
	for _, client := range clients {
		clientSet[client.IP] = client.Name
	}
	b.WriteString("clients:\n")
	cx := 0
	for k, v := range clientSet {
		fmt.Fprintf(&b, "%5d %30s %-15s\n", cx, k, v)
		cx++
	}
	b.WriteString("lookups:\n")
	for x, v := range lookups {
		fmt.Fprintf(&b, "%5d %30s %s\n", x, clientSet[v.Client], v.Host)
	}
	log.Println(b.String())

	return nil
}

var (
	ErrInitializationFailed = errors.New("initialization failed")

	dbPatches = []string{
		"CREATE TABLE IF NOT EXISTS lookups (evt TEXT NOT NULL, clientip TEXT NOT NULL, host TEXT NOT NULL, type TEXT NOT NULL, firstip TEXT, PRIMARY KEY(evt, clientip, host) ON CONFLICT REPLACE)",
		"CREATE TABLE IF NOT EXISTS clients (ip  TEXT NOT NULL, name     TEXT NOT NULL, PRIMARY KEY(ip, name) ON CONFLICT REPLACE)",
		"CREATE TABLE IF NOT EXISTS reverse (ip  TEXT NOT NULL, name     TEXT NOT NULL, PRIMARY KEY(ip, name) ON CONFLICT REPLACE)",
	}
)

func NewSQLiteStore(path string) (Store, error) { return &sqliteStore{db: path}, nil }

type sqliteStore struct {
	db    string
	once  sync.Once
	valid bool
}

func (s *sqliteStore) Accept(clients []Client, lookups []Lookup) error {
	db, err := s.conn()
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT OR REPLACE INTO clients (ip, name) VALUES (?, ?)")
	if err != nil {
		return err
	}
	cl2 := map[string]string{}
	for _, client := range clients {
		cl2[client.IP] = client.Name
	}
	for ip, name := range cl2 {
		if _, err = stmt.Exec(ip, name); err != nil {
			_ = stmt.Close()
			return err
		}
	}
	_ = stmt.Close()

	stmt, err = tx.Prepare("INSERT OR REPLACE INTO lookups (evt, clientip, host, type, firstip) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	revStmt, err := tx.Prepare("INSERT OR REPLACE INTO reverse (ip, name) VALUES (?, ?)")
	if err != nil {
		return err
	}
	ips := 0
	for _, lookup := range lookups {
		ips += len(lookup.AllIPs)
		if _, err = stmt.Exec(lookup.When, lookup.Client, lookup.Host, lookup.Type, lookup.FirstIP); err != nil {
			_ = stmt.Close()
			_ = revStmt.Close()
			return err
		}
		for _, lip := range lookup.AllIPs {
			if _, err = revStmt.Exec(lip, lookup.Host); err != nil {
				_ = stmt.Close()
				_ = revStmt.Close()
				return err
			}
		}
	}
	_ = stmt.Close()
	_ = revStmt.Close()

	log.Printf("wrote %d clients, %d reverse ips, and %d lookups", len(cl2), ips, len(lookups))
	return tx.Commit()
}

func (s *sqliteStore) conn() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", s.db)
	if err != nil {
		return nil, err
	}
	s.once.Do(func() { s.initialize(db) })
	if !s.valid {
		return nil, ErrInitializationFailed
	}
	return db, nil
}

func (s *sqliteStore) initialize(db *sql.DB) (err error) {
	for x, dbPatch := range dbPatches {
		if _, err = db.Exec(dbPatch); err != nil {
			log.Printf("error applying database patch %d: %v", x, err)
			return
		}
	}
	s.valid = true
	return
}
