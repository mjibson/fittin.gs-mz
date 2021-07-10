-- TODO: ignore anything with no hi slots fitted, which will remove capsules.
-- Can maybe do this by excluding anything without a flag between 27, 34

DROP SOURCE IF EXISTS zk_bytes CASCADE;
DROP SOURCE IF EXISTS zk_bytes_json CASCADE;
DROP SOURCE IF EXISTS zk_bytes_csv CASCADE;
DROP TABLE IF EXISTS queries CASCADE;

CREATE SOURCE zk_bytes_json
	FROM FILE '/home/mjibson/scratch/fit-mz/zkillboard.json'
	WITH (tail=true)
	FORMAT BYTES;

CREATE SOURCE zk_bytes_csv (data)
	FROM FILE '/home/mjibson/scratch/fit-mz/export/n1.0.csv'
--	FROM FILE '/home/mjibson/scratch/fit-mz/export/out.csv'
	WITH (tail=true)
	FORMAT CSV WITH 1 COLUMNS;

CREATE VIEW zk_json AS
	SELECT
		(data->'killID')::INT4 AS killmail,
		(data->'killmail'->'victim'->'items') AS items,
		(data->'killmail'->'victim'->'ship_type_id')::INT4 AS ship,
		(data->'zkb'->'fittedValue')::INT8 cost
	FROM
		(SELECT data::JSONB FROM zk_bytes_csv);
		--(SELECT convert_from(data, 'utf8') AS data FROM zk_bytes_json)k
CREATE INDEX ON zk_json (killmail);

CREATE VIEW zk_item_name AS
	WITH
		itemized
			AS (
				-- This previously used a DISTINCT, but jsonb_object_agg already performs
				-- a per-row de-duplication, so we can skip it. This is good because
				-- https://github.com/MaterializeInc/materialize/issues/7329 might produce
				-- problems (although this specific CTE has not been tested with DISTINCT at
				-- scale, so dunno).
				SELECT
					killmail,
					(items.value->'flag')::INT4 AS flag,
					(items.value->'item_type_id')::INT4 AS item
				FROM
					zk_json,
					jsonb_array_elements(
						items ||
						-- Add in the ship by adding an object to the items array.
						jsonb_build_object(
							'item_type_id',
							ship,
							'flag',
							0
						)
					) items
			),
		named
			AS (
				SELECT
					itemized.killmail,
					items.id item_id,
					jsonb_build_object(
						'id', items.id,
						'name', items.name,
						'group', items."group",
						'group_name', groups.name,
						'category', groups.category,
						'slot',
						CASE
							WHEN flag >= 27 AND flag <= 34 THEN 'hi'
							WHEN flag >= 19 AND flag <= 26 THEN 'med'
							WHEN flag >= 11 AND flag <= 18 THEN 'lo'
							WHEN flag >= 92 AND flag <= 99 THEN 'rig'
							WHEN flag >= 125 AND flag <= 132 THEN 'sub'
							ELSE NULL
						END
					) item
				FROM
					itemized,
					items,
					groups
				WHERE
					itemized.item = items.id
					AND items."group" = groups.id
			)
	SELECT
		killmail,
		jsonb_object_agg(item_id, item) names
	FROM
		named
	WHERE
		-- Remove items that aren't fitted. This also has the effect of removing
		-- killmails with no fitted items (capsules, MTUs).
		item->'slot' != 'null'
		OR item->>'category' = 'ship'
	GROUP BY
		killmail
	HAVING
		-- Also remove any ship (since other killmails with no fitted items were
		-- already removed) that has no hi slots fitted, since we almost certainly
		-- don't care about those.
		count(item->>'slot' = 'hi') > 0;

CREATE VIEW fits AS
	SELECT
		zk_json.killmail,
		zk_json.ship,
		zk_json.cost,
		zk_item_name.names,
		zk_json.items
	FROM
		zk_json,
		zk_item_name
	WHERE
		zk_json.killmail = zk_item_name.killmail
;
CREATE INDEX ON fits(killmail);

CREATE TABLE queries (id INT8 not null, items jsonb NOT NULL);

CREATE VIEW query_items AS
SELECT id, value::INT4
FROM
    queries,
    jsonb_array_elements(queries.items)
WHERE value IS NOT NULL;
CREATE INDEX ON query_items (id);

CREATE  VIEW query_counts AS
SELECT id, jsonb_array_length(items) as total
FROM queries;

-- Use jsonb_object_keys instead of jsonb_array_elements because the latter
-- would require de-duping via DISTINCT, which is currently not handled well
-- for large datasets:
-- https://github.com/MaterializeInc/materialize/issues/7329
CREATE VIEW fits_items AS
SELECT killmail, jsonb_object_keys::INT4 AS value
FROM
    fits,
    jsonb_object_keys(fits.names);
--CREATE INDEX ON fits_items (value);

CREATE VIEW query_fits AS
	SELECT
		query_items.id,
		fits_items.killmail,
		count(*) AS found
	FROM
		query_items, fits_items
	WHERE
		query_items.value = fits_items.value
	GROUP BY
		query_items.id, fits_items.killmail;

CREATE VIEW query_matches AS
	SELECT
		query_fits.id, killmail
	FROM
		query_fits, query_counts
	WHERE
		query_fits.id = query_counts.id
		AND found = total;

CREATE VIEW results AS
	SELECT
		id, killmail
	FROM
		queries,
		LATERAL (
			SELECT
				killmail
			FROM
				query_matches
			WHERE
				query_matches.id = queries.id
			ORDER BY
				killmail DESC
			LIMIT
				100
		);
CREATE INDEX ON results (id);

CREATE MATERIALIZED VIEW killmail_results AS
	SELECT
		results.id AS query_id,
		fits.*
	FROM
		results, fits
	WHERE
		results.killmail = fits.killmail;
CREATE INDEX ON killmail_results (query_id);

CREATE MATERIALIZED VIEW killmail_results_root AS
	SELECT
		*
	FROM
		fits
	ORDER BY
		killmail DESC
	LIMIT
		100;
