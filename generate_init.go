// +build ignore

package main

// Generates groups.sql and items.sql as static views from an sde extract in the current directory.
// https://developers.eveonline.com/resource/resources

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

func main() {
	groups := map[int32]bool{}
	{
		fmt.Println("reading groupIDs.yaml")
		r, err := os.Open("sde/fsd/groupIDs.yaml")
		if err != nil {
			panic(err)
		}
		var yml map[int32]struct {
			CategoryID int32 `yaml:"categoryID"`
			Name       map[string]string
		}
		if err := yaml.NewDecoder(r).Decode(&yml); err != nil {
			panic(err)
		}
		f, err := os.Create("groups.sql")
		if err != nil {
			panic(err)
		}
		f.WriteString("DROP VIEW IF EXISTS groups CASCADE;\n")
		f.WriteString("CREATE VIEW groups (id, name, category) AS\n\tVALUES")
		sep := ""
		var ids []int32
		for id := range yml {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		type Group struct {
			Name     string
			Lower    string
			Category int32
		}
		asJson := map[int32]Group{}
		for _, id := range ids {
			m := yml[id]
			var cat string
			switch m.CategoryID {
			case 8:
				cat = "charge"
			case 7:
				cat = "module"
			case 6:
				cat = "ship"
			case 32:
				cat = "subsystem"
			default:
				continue
			}
			f.WriteString(sep)
			sep = ","
			fmt.Fprintf(f, "\n\t\t(%d, '%s', '%s')", id, m.Name["en"], cat)
			groups[id] = true
			asJson[id] = Group{
				Name:     m.Name["en"],
				Lower:    strings.ToLower(m.Name["en"]),
				Category: m.CategoryID,
			}
		}
		f.WriteString(";\n")
		r.Close()
		if err := f.Close(); err != nil {
			panic(err)
		}
		f, err = os.Create("groups.json")
		if err != nil {
			panic(err)
		}
		enc := json.NewEncoder(f)
		enc.SetIndent("", "\t")
		if err := enc.Encode(asJson); err != nil {
			panic(err)
		}
		if err := f.Close(); err != nil {
			panic(err)
		}
	}

	{
		fmt.Println("reading types.yaml")
		r, err := os.Open("sde/fsd/typeIDs.yaml")
		if err != nil {
			panic(err)
		}
		var yml map[int32]struct {
			GroupID int32 `yaml:"groupID"`
			Name    map[string]string
		}
		if err := yaml.NewDecoder(r).Decode(&yml); err != nil {
			panic(err)
		}
		f, err := os.Create("items.sql")
		if err != nil {
			panic(err)
		}
		f.WriteString("DROP VIEW IF EXISTS items CASCADE;\n")
		f.WriteString("CREATE VIEW items (id, name, group) AS\n\tVALUES")
		sep := ""
		var ids []int32
		for id := range yml {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		type Item struct {
			ID    int32
			Name  string
			Lower string
			Group int32
		}
		asJson := map[int32]Item{}
		for _, id := range ids {
			m := yml[id]
			if !groups[m.GroupID] {
				continue
			}
			name := m.Name["en"]
			name = strings.ReplaceAll(name, "'", "''")
			f.WriteString(sep)
			sep = ","
			fmt.Fprintf(f, "\n\t\t(%d, '%s', %d)", id, name, m.GroupID)
			asJson[id] = Item{
				ID:    id,
				Name:  m.Name["en"],
				Lower: strings.ToLower(m.Name["en"]),
				Group: m.GroupID,
			}
		}
		f.WriteString(";\n")
		r.Close()
		if err := f.Close(); err != nil {
			panic(err)
		}
		f, err = os.Create("items.json")
		if err != nil {
			panic(err)
		}
		enc := json.NewEncoder(f)
		enc.SetIndent("", "\t")
		if err := enc.Encode(asJson); err != nil {
			panic(err)
		}
		if err := f.Close(); err != nil {
			panic(err)
		}
	}
}
