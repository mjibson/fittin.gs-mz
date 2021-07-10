package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	servertiming "github.com/mitchellh/go-server-timing"
)

//go:embed groups.json
var GROUPS_JSON []byte

//go:embed items.json
var ITEMS_JSON []byte

type WebContext struct {
	DB           *sql.DB
	X            *sqlx.DB
	Items        map[int32]Item
	Groups       map[int32]Group
	enableTiming bool

	lock        sync.RWMutex
	lastQueryID int64
	queries     map[string]int64
}

type Item struct {
	ID    int32
	Name  string
	Lower string
	Group int32
}

type Group struct {
	Name     string
	Lower    string
	Category int32
}

func (g Group) IsCharge() bool {
	return g.Category == 8
}

func web(port string, dbURL string) {
	db := init_sql(dbURL)
	defer db.Close()

	s := &WebContext{
		DB:           db,
		X:            sqlx.NewDb(db, "postgres"),
		enableTiming: true,
		queries:      make(map[string]int64),
	}

	if err := json.Unmarshal(GROUPS_JSON, &s.Groups); err != nil {
		panic(err)
	}
	if err := json.Unmarshal(ITEMS_JSON, &s.Items); err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/Fit", s.Wrap(s.Fit))
	mux.Handle("/api/Fits", s.Wrap(s.Fits))
	mux.Handle("/api/Search", s.Wrap(s.Search))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := s.DB.Ping(); err != nil {
			http.Error(w, err.Error(), 500)
		}
	})

	fmt.Println("HTTP listen on addr:", port)
	log.Fatal(http.ListenAndServe(port, mux))
}

func (s *WebContext) Wrap(
	f func(context.Context, *http.Request, *servertiming.Header) (interface{}, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Max-Age", "3600")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*60)
		defer cancel()
		var sh servertiming.Header
		ctx = servertiming.NewContext(ctx, &sh)
		if v, err := url.ParseQuery(r.URL.RawQuery); err == nil {
			r.URL.RawQuery = v.Encode()
		}
		url := r.URL.String()
		start := time.Now()
		defer func() { fmt.Printf("%s: %s\n", url, time.Since(start)) }()
		tm := servertiming.FromContext(ctx).NewMetric("req").Start()
		res, err := f(ctx, r, &sh)
		tm.Stop()
		if len(sh.Metrics) > 0 {
			w.Header().Add(servertiming.HeaderKey, sh.String())
			if s.enableTiming {
				for _, m := range sh.Metrics {
					fmt.Printf("timing: %s: %s\n", m.Name, m.Duration)
				}
			}
		}
		if err != nil {
			log.Printf("%s: %+v", url, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data, gzip, err := resultToBytes(res)
		if err != nil {
			log.Printf("%s: %v", url, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeDataGzip(w, r, data, gzip)
	}
}

func (s *WebContext) Fit(
	ctx context.Context, r *http.Request, timing *servertiming.Header,
) (interface{}, error) {
	id := r.FormValue("id")
	if id == "" {
		return nil, errors.New("missing fit id")
	}
	var ret struct {
		Killmail int64
		Ship     int32
		Cost     int64
		Names    json.RawMessage
		Items    json.RawMessage
	}
	if err := s.DB.QueryRowContext(ctx, `SELECT killmail, ship, cost, names, items from fits where killmail = $1`, id).Scan(&ret.Killmail, &ret.Ship, &ret.Cost, &ret.Names, &ret.Items); err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *WebContext) Fits(
	ctx context.Context, r *http.Request, timing *servertiming.Header,
) (interface{}, error) {
	var ret struct {
		Filter map[string][]Item
		Fits   []struct {
			Killmail int64
			Ship     int32
			Cost     int64
			Names    json.RawMessage
			Items    json.RawMessage
		}
	}
	ret.Filter = map[string][]Item{}
	r.ParseForm()

	items := map[int]struct{}{}
	if ship, _ := strconv.Atoi(r.Form.Get("ship")); ship > 0 {
		items[ship] = struct{}{}
		ret.Filter["ship"] = append(ret.Filter["ship"], s.Items[int32(ship)])
	}
	for _, item := range r.Form["item"] {
		itemid, _ := strconv.Atoi(item)
		if itemid <= 0 {
			continue
		}
		items[itemid] = struct{}{}
		ret.Filter["item"] = append(ret.Filter["item"], s.Items[int32(itemid)])
	}

	var query strings.Builder
	query.WriteString(`
		SELECT
			killmail,
			ship,
			cost,
			names,
			items
		FROM
			killmail_results`)
	var args []interface{}
	if len(items) == 0 {
		query.WriteString("_root")
	} else {
		queryID, err := s.QueryID(ctx, items)
		if err != nil {
			return nil, err
		}
		query.WriteString(` WHERE query_id = $1`)
		args = append(args, queryID)
	}
	fmt.Println(query.String())

	selectT := timing.NewMetric("select").Start()
	// TODO: handle case if mz restarts and there's no queryid. Maybe detectable if
	// no rows?
	err := s.X.SelectContext(ctx, &ret.Fits, query.String(), args...)
	selectT.Stop()
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *WebContext) QueryID(ctx context.Context, itemMap map[int]struct{}) (int64, error) {
	// Dedup and sort to make canonical.
	items := make([]int, 0, len(itemMap))
	for item, _ := range itemMap {
		items = append(items, item)
	}
	sort.Ints(items)
	marshaled, err := json.Marshal(items)
	if err != nil {
		return 0, err
	}
	name := string(marshaled)

	s.lock.RLock()
	query_id := s.queries[name]
	s.lock.RUnlock()
	if query_id != 0 {
		return query_id, nil
	}

	// If there's no query then:
	// - Check if it already exists.
	// - Increment the query counter. We use that instead of len(s.queries) if we need to skip one.
	// - Check if the query id already exists.
	// - Insert the new query.
	// - Wait for results.
	s.lock.Lock()
	defer s.lock.Unlock()

	if err := s.DB.QueryRowContext(ctx, "SELECT id FROM queries WHERE items = $1", name).Scan(&query_id); err == nil {
		// This query already has an id, use it.
		s.queries[name] = query_id
		fmt.Println("ALREADY", query_id, name)
		return query_id, nil
	}

	for {
		s.lastQueryID++
		fmt.Println("MAYBE", s.lastQueryID, "FOR", name)
		if err := s.DB.QueryRowContext(ctx, "SELECT id FROM queries WHERE id = $1", s.lastQueryID).Scan(&query_id); err == sql.ErrNoRows {
			// Ok.
		} else if err == nil {
			// This id already exists somehow; increment and try again. This isn't a
			// perfect solution since the SELECT and INSERT aren't in a transaction, but
			// it's ok.
			continue
		} else {
			return 0, err
		}
		// TODO: add timing
		if _, err := s.DB.ExecContext(ctx, "INSERT INTO queries VALUES ($1, $2)", s.lastQueryID, name); err != nil {
			return 0, err
		}
		// We don't need to listen for updates because, since we used a table, any
		// subsequent select is guaranteed to have a higher timestamp than the insert
		// due to the linearizability guarantee provided by materialize.
		return s.lastQueryID, nil
	}
}

var searchCategories = map[int32]string{
	6:  "ship",
	7:  "item", // module
	8:  "item", // charge
	32: "item", // subsystem
}

func (s *WebContext) Search(
	ctx context.Context, r *http.Request, timing *servertiming.Header,
) (interface{}, error) {
	type Result struct {
		Type string
		Name string
		ID   int32
	}
	var ret struct {
		Search  string
		Results []Result
	}
	ret.Search = strings.ToLower(strings.TrimSpace(r.FormValue("term")))
	if len(ret.Search) < 3 {
		return nil, nil
	}
	fields := strings.Fields(ret.Search)
	match := func(s string) bool {
		if strings.Contains(s, ret.Search) {
			return true
		}
		containsAll := true
		for _, term := range fields {
			if !strings.Contains(s, term) {
				containsAll = false
				break
			}
		}
		return containsAll
	}
	for id, group := range s.Groups {
		if !match(group.Lower) {
			continue
		}
		ret.Results = append(ret.Results, Result{
			Type: "group",
			Name: group.Name,
			ID:   id,
		})
	}
	for id, item := range s.Items {
		if !match(item.Lower) {
			continue
		}
		if typ := searchCategories[s.Groups[item.Group].Category]; typ != "" {
			ret.Results = append(ret.Results, Result{
				Type: typ,
				Name: item.Name,
				ID:   id,
			})
		}
		if len(ret.Results) > 50 {
			break
		}
	}
	return ret, nil
}

func resultToBytes(res interface{}) (data, gzipped []byte, err error) {
	data, err = json.Marshal(res)
	if err != nil {
		return nil, nil, fmt.Errorf("json marshal: %w", err)
	}
	var gz bytes.Buffer
	gzw, _ := gzip.NewWriterLevel(&gz, gzip.BestCompression)
	if _, err := gzw.Write(data); err != nil {
		return nil, nil, fmt.Errorf("gzip: %w", err)
	}
	if err := gzw.Close(); err != nil {
		return nil, nil, fmt.Errorf("gzip close: %w", err)

	}
	return data, gz.Bytes(), nil
}

func writeDataGzip(w http.ResponseWriter, r *http.Request, data, gzip []byte) {
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Cache-Control", "max-age=3600")
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Add("Content-Encoding", "gzip")
		w.Write(gzip)
	} else {
		w.Write(data)
	}
}
