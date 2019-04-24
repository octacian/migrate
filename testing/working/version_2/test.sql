-- @migrate/up

ALTER TABLE test RENAME first_name TO FirstName;
ALTER TABLE test RENAME last_name TO LastName;

-- @migrate/down

ALTER TABLE test RENAME FirstName TO first_name;
ALTER TABLE test RENAME LastName TO last_name;
