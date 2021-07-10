DROP SOURCE IF EXISTS zk_bytes_json CASCADE;
DROP TABLE IF EXISTS queries CASCADE;

CREATE SOURCE zk_bytes_json
	FROM FILE '/home/mjibson/scratch/fit-mz/out.json'
	WITH (tail=true)
	FORMAT BYTES;

CREATE MATERIALIZED VIEW fits AS
	SELECT
		(data->'ID')::INT AS killmail,
		data
	FROM
		(SELECT CONVERT_FROM(data, 'utf8')::JSONB data FROM zk_bytes_json);
CREATE INDEX ON fits(killmail);

CREATE MATERIALIZED VIEW killmail_results_root AS
	SELECT
		data
	FROM
		fits
	ORDER BY
		killmail DESC
	LIMIT
		100;

CREATE TABLE queries (id INT8 not null, items jsonb NOT NULL);

CREATE VIEW query_items AS
SELECT id, value::INT4
FROM
    queries,
    jsonb_array_elements(queries.items)
WHERE value IS NOT NULL;
--CREATE INDEX ON query_items (id);

CREATE  VIEW query_counts AS
SELECT id, jsonb_array_length(items) as total
FROM queries;

-- Use jsonb_object_keys instead of jsonb_array_elements because the latter
-- would require de-duping via DISTINCT, which is currently not handled well
-- for large datasets:
-- https://github.com/MaterializeInc/materialize/issues/7329
CREATE VIEW fits_items AS
SELECT killmail, value::INT
FROM
    fits,
    jsonb_array_elements(fits.data->'QueryItems');
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
--CREATE INDEX ON results (id);

CREATE MATERIALIZED VIEW killmail_results AS
	SELECT
		results.id AS query_id,
		fits.*
	FROM
		results, fits
	WHERE
		results.killmail = fits.killmail;
CREATE INDEX ON killmail_results (query_id);
