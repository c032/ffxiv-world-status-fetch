package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/c032/go-ffxiv"
	"github.com/c032/go-logger"
	_ "github.com/lib/pq"
)

func mustReadEnvironmentVariable(key string) string {
	value := os.Getenv(key)
	if value != strings.TrimSpace(value) {
		panic(fmt.Sprintf("environment variable has leading or trailing whitespace: %#v", key))
	}
	if value == "" {
		panic(fmt.Sprintf("required environment variable is empty: %#v", key))
	}

	return value
}

func readTextFile(file string) (string, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("could not read file: %w", err)
	}

	// TODO Ensure `content` is valid UTF-8.

	return string(content), nil
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}

	return v
}

type Row struct {
	Version int64               `json:"version"`
	Items   []ffxiv.WorldStatus `json:"items"`
}

func mainWithError() error {
	log := logger.Default()

	var (
		err error
		db  *sql.DB
	)

	connStr := must(readTextFile(mustReadEnvironmentVariable("POSTGRESQL_CONNECTION_STRING_FILE")))

	log.Print("Connecting to database.")

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("could not connect to database: %w", err)
	}
	defer db.Close()

	log.Print("Preparing statement.")

	var stmt *sql.Stmt
	stmt, err = db.Prepare(`
		insert into ffxiv.worldstatus_history (
			worldstatus_timestamp,
			worldstatus_data
		) values (
			current_timestamp,
			$1::jsonb
		);
	`)
	if err != nil {
		return fmt.Errorf("could not prepare statement: %w", err)
	}

	log.Print("Creating client.")

	var c ffxiv.Client
	c, err = ffxiv.NewClient()
	if err != nil {
		return fmt.Errorf("could not create client: %w", err)
	}

	var worlds []ffxiv.WorldStatus

	log.Print("Fetching.")

	worlds, err = c.WorldStatus()
	if err != nil {
		return fmt.Errorf("could not fetch worlds status: %w", err)
	}

	row := Row{
		Version: ffxiv.Version,
		Items:   worlds,
	}

	asJSON, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("could not convert to JSON: %w", err)
	}

	log.Print("Saving to database.")

	_, err = stmt.Exec(asJSON)
	if err != nil {
		return fmt.Errorf("could not save to database: %w", err)
	}

	log.Print("Refreshing materialized view.")
	_, err = db.Exec("refresh materialized view ffxiv.worldstatus_v1_materialized;")
	if err != nil {
		return fmt.Errorf("could not refresh materialized view: %w", err)
	}

	return nil
}

func main() {
	err := mainWithError()
	if err != nil {
		panic(err)
	}
}
