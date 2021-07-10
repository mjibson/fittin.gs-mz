package main

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"os"
	"sort"
	"time"
)

func process(paths []string) {
	if len(paths) == 0 {
		panic("empty paths")
	}
	s := MakeSDEData()
	out, err := os.Create("out.json")
	if err != nil {
		panic(err)
	}
	j := json.NewEncoder(out)
	var z ZkillboardKillmail
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			panic(err)
		}
		csv := csv.NewReader(file)
		csv.FieldsPerRecord = 1
		for {
			record, err := csv.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				panic(err)
			}
			if err := json.Unmarshal([]byte(record[0]), &z); err != nil {
				panic(err)
			}
			dbkm, ok := processKillmail(&s, z)
			if !ok {
				continue
			}
			if err := j.Encode(dbkm); err != nil {
				panic(err)
			}
		}
	}
	if err := out.Close(); err != nil {
		panic(err)
	}
}

type DBKillmail struct {
	ID         int
	Cost       int
	Ship       int
	Hi         [8]DBItem
	Med        [8]DBItem
	Lo         [8]DBItem
	Rig        [8]DBItem
	Sub        [8]DBItem
	QueryItems []int
}

type DBItem struct {
	ID     int `json:",omitempty"`
	Charge int `json:",omitempty"`
}

func processKillmail(s *SDEData, z ZkillboardKillmail) (km DBKillmail, ok bool) {
	km = DBKillmail{
		ID:   z.KillID,
		Cost: int(z.Zkb.FittedValue),
	}
	queryItems := map[int]struct{}{}
	queryItems[z.Killmail.Victim.ShipTypeID] = struct{}{}
	hasHi := false
	if _, ok = s.MaybeNamedItem(z.Killmail.Victim.ShipTypeID); !ok {
		return km, false
	}
	km.Ship = z.Killmail.Victim.ShipTypeID
	for _, item := range z.Killmail.Victim.Items {
		var offset int
		var slot *[8]DBItem
		switch {
		case item.Flag >= 11 && item.Flag <= 18:
			offset = 11
			slot = &km.Lo
			hasHi = true
		case item.Flag >= 19 && item.Flag <= 26:
			offset = 19
			slot = &km.Med
		case item.Flag >= 27 && item.Flag <= 34:
			offset = 27
			slot = &km.Hi
		case item.Flag >= 92 && item.Flag <= 99:
			offset = 92
			slot = &km.Rig
		case item.Flag >= 125 && item.Flag <= 132:
			offset = 125
			slot = &km.Sub
		default:
			continue
		}
		idx := item.Flag - offset
		sdeItem, ok := s.Items[item.ItemTypeID]
		if !ok {
			continue
		}
		sdeGroup, ok := s.Groups[sdeItem.Group]
		if !ok {
			continue
		}
		if sdeGroup.IsCharge() {
			if _, ok := s.Items[sdeItem.ID]; !ok {
				return km, false
			}
			slot[idx].Charge = sdeItem.ID
		} else {
			slot[idx].ID = sdeItem.ID
		}
		queryItems[item.ItemTypeID] = struct{}{}
	}
	if !hasHi {
		return km, false
	}
	items := make([]int, 0, len(queryItems))
	for item := range queryItems {
		items = append(items, item)
	}
	sort.Ints(items)
	km.QueryItems = items
	return km, hasHi
}

type ZkillboardKillmail struct {
	KillID   int `json:"killID"`
	Killmail struct {
		KillmailID    int       `json:"killmail_id"`
		KillmailTime  time.Time `json:"killmail_time"`
		SolarSystemID int       `json:"solar_system_id"`
		Victim        struct {
			Items []struct {
				Flag       int `json:"flag"`
				ItemTypeID int `json:"item_type_id"`
			} `json:"items"`
			ShipTypeID int `json:"ship_type_id"`
		} `json:"victim"`
	} `json:"killmail"`
	Zkb struct {
		FittedValue float64 `json:"fittedValue"`
		Hash        string  `json:"hash"`
		Href        string  `json:"href"`
		LocationID  int     `json:"locationID"`
	} `json:"zkb"`
}
