-- Test data for GetSharedDatasetsForUser (store-level SQL-injection tests).
-- Mirrors api/service/testdata/shared-datasets-test.sql: user 9001 is a Guest
-- (permission_bit = 1) of org 100 with direct access to one dataset, which is the
-- exact shape GetSharedDatasetsForUser looks for. High IDs avoid seed conflicts.
-- Written to be idempotent so the tests can run against a reused DB.

INSERT INTO pennsieve.users (id, email, first_name, last_name, credential, color, url, is_super_admin, created_at, updated_at, node_id) VALUES
  (9001, 'guest.user@example.com', 'Guest', 'User', 'guest123', '#FF0000', 'https://example.com/guest', false, '2023-01-01 00:00:00', '2023-01-01 00:00:00', 'N:user:9001')
ON CONFLICT (id) DO NOTHING;

INSERT INTO pennsieve.organizations (id, name, slug, node_id, created_at, updated_at, storage_bucket, encryption_key_id) VALUES
  (100, 'Shared Org Alpha', 'shared-org-alpha', 'N:organization:100', '2023-01-01 00:00:00', '2023-01-01 00:00:00', 'shared-alpha-storage', 'test-key-100')
ON CONFLICT (id) DO NOTHING;

INSERT INTO pennsieve.organization_user (organization_id, user_id, permission_bit, created_at, updated_at) VALUES
  (100, 9001, 1, '2023-01-01 00:00:00', '2023-01-01 00:00:00')
ON CONFLICT (organization_id, user_id) DO UPDATE SET permission_bit = 1;

-- The "100" schema tables are cloned from "2" via CREATE TABLE AS, which does not
-- copy primary keys, so make the data inserts idempotent by clearing first.
CREATE SCHEMA IF NOT EXISTS "100";
CREATE TABLE IF NOT EXISTS "100".datasets AS TABLE "2".datasets WITH NO DATA;
CREATE TABLE IF NOT EXISTS "100".dataset_user AS TABLE "2".dataset_user WITH NO DATA;
TRUNCATE "100".dataset_user;
TRUNCATE "100".datasets;

INSERT INTO "100".datasets (id, name, description, state, status, node_id, created_at, updated_at, tags, data_use_agreement_id) VALUES
  (1, 'Alpha Dataset 1', 'First dataset in org 100', 'READY', 'AVAILABLE', 'N:dataset:alpha1', '2023-01-01 00:00:00', '2023-01-01 12:00:00', '{"research","medical"}', NULL);

INSERT INTO "100".dataset_user (dataset_id, user_id, role, created_at, updated_at) VALUES
  (1, 9001, 'viewer', '2023-01-01 00:00:00', '2023-01-01 00:00:00');
