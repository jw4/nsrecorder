BEGIN TRANSACTION;

CREATE TABLE IF NOT EXISTS hourly (evt TIMESTAMP WITH TIME ZONE, clientip STRING, name STRING, total INT);

INSERT INTO hourly (total, name, clientip, evt)
SELECT 
  COUNT(*) AS Total,
  name as Name,
  clientip as Client,
  date_trunc('hour', evt) AS Time
FROM 
  lookups 
WHERE
  evt < (current_date() - 7)
GROUP BY 
  date_trunc('hour', evt),
  name,
  clientip
ORDER BY
  date_trunc('hour', evt) ASC,
  COUNT(*) ASC
;

DELETE FROM lookups WHERE evt < (SELECT MAX(evt) FROM hourly);

COMMIT;
