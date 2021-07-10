package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func read_json() {
	for {
		if err := read_json_record(); err != nil {
			fmt.Println(err)
			time.Sleep(time.Second * 10)
		}
	}
}

func read_json_record() error {
	resp, err := http.Get("https://redisq.zkillboard.com/listen.php?queueID=fittings-mz")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		io.Copy(os.Stderr, resp.Body)
		return errors.New(resp.Status)
	}

	var pkg struct {
		Package *json.RawMessage `json:"package"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pkg); err != nil {
		return err
	}
	if pkg.Package == nil {
		return nil
	}
	f, err := os.OpenFile("zkillboard.json", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if _, err := f.Write(*pkg.Package); err != nil {
		return err
	}
	if _, err := f.WriteString("\n"); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return nil
}

/*
CREATE SOURCE bytea_data
FROM FILE '/home/mjibson/scratch/fit-mz/zkillboard.json'
WITH (tail=true)
FORMAT BYTES;

CREATE MATERIALIZED VIEW json AS
SELECT CAST(data AS JSONB) AS data
FROM (
    SELECT CONVERT_FROM(data, 'utf8') AS data
    FROM bytea_data
);

select json_build_object('killID', id, 'killmail', km, 'zkb', zkb) from killmails where id=93743656;

export into csv
's3://mjibson-fittings/export1/?AWS_ACCESS_KEY_ID=ASIAV2KIV5LPVN4KPXW6&AWS_SECRET_ACCESS_KEY=cDxh1AX9hXAVQ/EQx4szuU5Y3UL9KsHa5kRh6QCK&AWS_SESSION_TOKEN=IQoJb3JpZ2luX2VjEDwaCXVzLWVhc3QtMSJGMEQCIE01T6fYQnwoUItwls%2FRyFYx12gzNEEEG%2BnHNS5A%2Fnu8AiANud4O6I%2FVnzqkAtDMN4%2BffePtPJP0swku4NSVx2D%2F3yqSAwglEAEaDDQwMDEyMTI2MDc2NyIM4Mp5Se8lWt5N7njkKu8CCT6bZVXPtbXF74S0zjX1v6wvoTDINdjdUrRA5ibyWLfUHvuRkhBFg37wDsLrkMqEpVo3SDrsuFkpp9Pca16223%2BEftD7dVzXvTd%2Fr9KJ6VGgTt4ZtzEYrpXM%2B9IdvSuefjYBryBd4wD2G5t5hyDzB8WFoJby4AybuvxnqnWXTn0AFo41HZ5WT3qBpjP%2F5p%2B7CVe%2FOZU1zJJpW9znSskyXUDss1zckBZT8R3lpZ15oMy9K1aRuC49D2JgYEMsHoeKWAM1s8cwgRy7rslt7vUab3%2FILcUQ%2BgGt4%2BQk%2BTQFEo1jYf%2BMo%2BykuyWVVeuLsaDv%2FqU%2FR1zfi0Qkw%2BP8gKaNfH6JTLj7Z0RK7f6gIYIe0XKduNNqB9Ka65Zv13bVaTYU5qPCPe0hjcK3FUJfK%2Bk8Z0b48q9%2FqxwyruvF0j4yfUfM955rasClaKpuipsF0gc%2F%2BTDHy1A%2BSOfpIMKH4nFv5upHPreGFvssRLwW6JOWTDCJgYqHBjqnAZZyatTav86k%2Bru9rkMpk4UysgAQ9v%2BZ8zk2a6fqsPf2AP8r0d0r6KySzoYUtx0soto6A5uQ3%2FNLxMyFwhIZ%2FyIpKU81aC%2B%2BFYXSYu1AQGwu4tirUDaeW1yv1PHRLjQwymJXg9vTywkNgGkVX0UUtWfr2pG21EGKvGAWe%2B52ftaU6DzbDD57l1zaBetoFOV%2Fb5k80a8MkG1hN8eXbZbjBRRYcxBinAtn'
from select json_build_object('killID', id, 'killmail', km, 'zkb', zkb) from killmails;


{
    "killID": 93743656,
    "killmail":
        {
            "attackers":
                [
                    {
                        "alliance_id": 99010894,
                        "character_id": 92404330,
                        "corporation_id": 98607967,
                        "damage_done": 7786,
                        "final_blow": true,
                        "security_status": -3.6,
                        "ship_type_id": 643,
                        "weapon_type_id": 2446
                    }
                ],
            "killmail_id": 93743656,
            "killmail_time": "2021-07-04T01:42:00Z",
            "solar_system_id": 30005234,
            "victim":
                {
                    "character_id": 2118810323,
                    "corporation_id": 1000169,
                    "damage_taken": 7786,
                    "items":
                        [
                            {"flag": 11, "item_type_id": 10998, "quantity_destroyed": 1, "singleton": 0},
                            {"flag": 13, "item_type_id": 5631, "quantity_destroyed": 1, "singleton": 0},
                            {"flag": 20, "item_type_id": 10872, "quantity_dropped": 1, "singleton": 0},
                            {"flag": 21, "item_type_id": 434, "quantity_destroyed": 1, "singleton": 0},
                            {"flag": 19, "item_type_id": 8419, "quantity_dropped": 1, "singleton": 0},
                            {"flag": 12, "item_type_id": 5631, "quantity_destroyed": 1, "singleton": 0},
                            {"flag": 14, "item_type_id": 1403, "quantity_destroyed": 1, "singleton": 0},
                            {"flag": 5, "item_type_id": 2395, "quantity_dropped": 8728, "singleton": 0},
                            {"flag": 24, "item_type_id": 580, "quantity_dropped": 1, "singleton": 0},
                            {"flag": 94, "item_type_id": 31790, "quantity_destroyed": 1, "singleton": 0}
                        ],
                    "position": {"x": -2538870788931.712, "y": 514405606324.65356, "z": -1080168908850.236},
                    "ship_type_id": 648
                }
        },
    "zkb":
        {
            "awox": false,
            "destroyedValue": 2465925.46,
            "droppedValue": 11565533.01,
            "fittedValue": 2577267.67,
            "hash": "40f54c3f2773377cad3a2649625c646b2b0d799a",
            "href": "https://esi.evetech.net/v1/killmails/93743656/40f54c3f2773377cad3a2649625c646b2b0d799a/",
            "labels": ["cat:6", "solo", "pvp", "lowsec"],
            "locationID": 40330966,
            "npc": false,
            "points": 10,
            "solo": true,
            "totalValue": 14031458.47
        }
}

*/
