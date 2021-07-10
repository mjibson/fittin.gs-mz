package main

import (
	"database/sql"
	_ "embed"
	"encoding/json"
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
	read: append json from zkillboard into zkillboard.json
	process: process csv from cli args into out.json files`)
	os.Exit(1)
}

func main() {
	flag.Parse()
	if len(os.Args) < 2 {
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
	case "process":
		process(os.Args[2:])
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

type SDEData struct {
	Items  map[int]Item
	Groups map[int]Group
}

func MakeSDEData() SDEData {
	var s SDEData
	if err := json.Unmarshal(GROUPS_JSON, &s.Groups); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(ITEMS_JSON, &s.Items); err != nil {
		panic(err)
	}
	return s
}

func (s *SDEData) NamedItem(id int) NamedItem {
	item := s.Items[id]
	return NamedItem{
		ID:   item.ID,
		Name: item.Name,
	}
}

func (s *SDEData) MaybeNamedItem(id int) (NamedItem, bool) {
	item, ok := s.Items[id]
	return NamedItem{
		ID:   item.ID,
		Name: item.Name,
	}, ok
}

type Item struct {
	ID    int
	Name  string
	Lower string
	Group int
}

type Group struct {
	Name     string
	Lower    string
	Category int
}

func (g Group) IsCharge() bool {
	return g.Category == 8
}

type NamedItem struct {
	ID   int    `json:",omitempty"`
	Name string `json:",omitempty"`
}

type ItemCharge struct {
	NamedItem
	Charge *NamedItem `json:",omitempty"`
}

type FittingsKillmail struct {
	ID     int
	Cost   int
	Ship   Item
	Hi     [8]ItemCharge
	Med    [8]ItemCharge
	Lo     [8]ItemCharge
	Rig    [8]ItemCharge
	Sub    [8]ItemCharge
	Charge []Item
}
