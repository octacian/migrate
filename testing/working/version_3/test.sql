-- @migrate/up

ALTER TABLE test ADD COLUMN Email VARCHAR(255);

-- @migrate/down

BEGIN TRANSACTION;

ALTER TABLE test RENAME TO temp_test;

CREATE TABLE test(
	ID INT PRIMARY KEY,
	FirstName VARCHAR(255),
	LastName VARCHAR(255)
);

INSERT INTO test SELECT ID, FirstName, LastName FROM temp_test;

DROP TABLE temp_test;

COMMIT;
