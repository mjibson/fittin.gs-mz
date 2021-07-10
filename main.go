package main

import (
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/lib/pq"

	// Needed so `go mod tidy` won't remove it for generate_init.go, which has a build ignore.
	_ "gopkg.in/yaml.v2"
)

//go:generate go run generate_init.go

//go:embed groups.sql
var GROUPS_SQL string

//go:embed items.sql
var ITEMS_SQL string

//go:embed init.sql
var INIT_SQL string

type Specification struct {
	Port    string `default:"4001"`
	DB_Addr string `default:"postgres://materialize@localhost:6875/?sslmode=disable"`
}

func usage() {
	fmt.Println(`run with argument:
	web: start webserver on $PORT at $DB_ADDR
	init: initialize views at $DB_ADDR
	read: append json from zkillboard into zkillboard.json`)
	os.Exit(1)
}

func main() {
	flag.Parse()
	if len(os.Args) != 2 {
		usage()
	}

	var spec Specification
	err := envconfig.Process("", &spec)
	if err != nil {
		log.Fatal(err.Error())
	}
	if !strings.Contains(spec.Port, ":") {
		spec.Port = fmt.Sprintf(":%s", spec.Port)
	}

	switch os.Args[1] {
	case "web":
		web(spec.Port, spec.DB_Addr)
	case "init":
		init_db(spec.DB_Addr)
	case "read":
		read_json()
	default:
		usage()
	}
}

func init_sql(dbURL string) *sql.DB {
	connector, err := pq.NewConnector(dbURL)
	if err != nil {
		log.Fatal(err)
	}
	db := sql.OpenDB(connector)
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("inited", dbURL)
	return db
}

func init_db(dbURL string) {
	db := init_sql(dbURL)
	defer db.Close()

	for _, sql := range []string{
		GROUPS_SQL,
		ITEMS_SQL,
		INIT_SQL,
	} {
		for _, sql := range strings.Split(sql, ";\n") {
			if _, err := db.Exec(sql); err != nil {
				fmt.Println(sql)
				panic(err)
			}
		}
	}
}
