-- @migrate/up

ALTER TABLE test RENAME TO new_test;

-- @migrate/down

ALTER TABLE new_test RENAME TO test;
