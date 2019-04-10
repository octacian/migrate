-- @migrate/up

CREATE TABLE IF NOT EXISTS test(
	ID INT PRIMARY KEY,
	first_name VARCHAR(255),
	last_name VARCHAR(255)
);

-- @migrate/down

DROP TABLE IF EXISTS test;
