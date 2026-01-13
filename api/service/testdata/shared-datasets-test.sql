-- Test data for shared datasets functionality
-- This sets up cross-organization scenarios where users have guest access (permission_bit = 1)
-- and should see shared datasets from other organizations

-- Insert test users (using very high IDs to avoid conflicts)
INSERT INTO pennsieve.users (id, email, first_name, last_name, credential, color, url, authy_id, is_super_admin, preferred_org_id, created_at, updated_at, node_id) VALUES
(9001, 'guest.user@example.com', 'Guest', 'User', 'guest123', '#FF0000', 'https://example.com/guest', NULL, false, NULL, '2023-01-01 00:00:00', '2023-01-01 00:00:00', 'N:user:9001'),
(9002, 'full.contributor@example.com', 'Full', 'Contributor', 'full123', '#00FF00', 'https://example.com/full', NULL, false, NULL, '2023-01-01 00:00:00', '2023-01-01 00:00:00', 'N:user:9002'),
(9003, 'another.guest@example.com', 'Another', 'Guest', 'guest456', '#0000FF', 'https://example.com/another', NULL, false, NULL, '2023-01-01 00:00:00', '2023-01-01 00:00:00', 'N:user:9003');

-- Create schemas for test organizations
CREATE SCHEMA IF NOT EXISTS "100";
CREATE SCHEMA IF NOT EXISTS "101";

-- Create datasets table in test schemas
CREATE TABLE IF NOT EXISTS "100".datasets AS TABLE "2".datasets WITH NO DATA;
CREATE TABLE IF NOT EXISTS "100".dataset_user AS TABLE "2".dataset_user WITH NO DATA;
CREATE TABLE IF NOT EXISTS "101".datasets AS TABLE "2".datasets WITH NO DATA;
CREATE TABLE IF NOT EXISTS "101".dataset_user AS TABLE "2".dataset_user WITH NO DATA;

-- Insert test organizations (using high IDs to avoid conflicts)
INSERT INTO pennsieve.organizations (id, name, slug, node_id, created_at, updated_at, storage_bucket, encryption_key_id) VALUES
(100, 'Shared Org Alpha', 'shared-org-alpha', 'N:organization:100', '2023-01-01 00:00:00', '2023-01-01 00:00:00', 'shared-alpha-storage', 'test-key-100'),
(101, 'Shared Org Beta', 'shared-org-beta', 'N:organization:101', '2023-01-01 00:00:00', '2023-01-01 00:00:00', 'shared-beta-storage', 'test-key-101');

-- Set up organization membership
-- User 9001 (guest.user@example.com) has guest access (permission_bit = 1) to org 100 and 101
-- User 9002 (full.contributor@example.com) has full access (permission_bit = 32) to org 100
-- User 9003 (another.guest@example.com) has guest access (permission_bit = 1) to org 101 only
INSERT INTO pennsieve.organization_user (organization_id, user_id, permission_bit, created_at, updated_at) VALUES
(100, 9001, 1, '2023-01-01 00:00:00', '2023-01-01 00:00:00'),  -- Guest in org 100
(101, 9001, 1, '2023-01-01 00:00:00', '2023-01-01 00:00:00'),  -- Guest in org 101
(100, 9002, 32, '2023-01-01 00:00:00', '2023-01-01 00:00:00'), -- Full contributor in org 100
(101, 9003, 1, '2023-01-01 00:00:00', '2023-01-01 00:00:00');  -- Guest in org 101

-- Create datasets in org 100 schema
INSERT INTO "100".datasets (id, name, description, state, status, node_id, created_at, updated_at, tags, data_use_agreement_id) VALUES
(1, 'Alpha Dataset 1', 'First dataset in org 100', 'READY', 'AVAILABLE', 'N:dataset:alpha1', '2023-01-01 00:00:00', '2023-01-01 12:00:00', '{"research","medical"}', NULL),
(2, 'Alpha Dataset 2', 'Second dataset in org 100', 'READY', 'AVAILABLE', 'N:dataset:alpha2', '2023-01-01 00:00:00', '2023-01-01 13:00:00', '{"public"}', NULL),
(3, 'Alpha Dataset Deleted', 'Deleted dataset should not appear', 'DELETED', 'UNAVAILABLE', 'N:dataset:alpha3', '2023-01-01 00:00:00', '2023-01-01 14:00:00', '{}', NULL);

-- Create datasets in org 101 schema  
INSERT INTO "101".datasets (id, name, description, state, status, node_id, created_at, updated_at, tags, data_use_agreement_id) VALUES
(1, 'Beta Dataset 1', 'First dataset in org 101', 'READY', 'AVAILABLE', 'N:dataset:beta1', '2023-01-02 00:00:00', '2023-01-02 12:00:00', '{"collaboration"}', NULL),
(2, 'Beta Dataset 2', 'Second dataset in org 101', 'PROCESSING', 'AVAILABLE', 'N:dataset:beta2', '2023-01-02 00:00:00', '2023-01-02 13:00:00', '{}', 1),
(3, 'Beta Dataset Private', 'No access granted', 'READY', 'AVAILABLE', 'N:dataset:beta3', '2023-01-02 00:00:00', '2023-01-02 14:00:00', '{"private"}', NULL);

-- Grant dataset access to users
-- User 9001 (guest) should have access to alpha1, alpha2, beta1
-- User 9002 (full contributor) should not appear in shared datasets (has permission_bit 32)
-- User 9003 (guest) should have access to beta1 only

-- Org 100 dataset access
INSERT INTO "100".dataset_user (dataset_id, user_id, role, created_at, updated_at) VALUES
(1, 9001, 'viewer', '2023-01-01 00:00:00', '2023-01-01 00:00:00'), -- Guest access to alpha1
(2, 9001, 'viewer', '2023-01-01 00:00:00', '2023-01-01 00:00:00'), -- Guest access to alpha2
(1, 9002, 'owner', '2023-01-01 00:00:00', '2023-01-01 00:00:00'),  -- Full contributor (should not show in shared datasets)
(2, 9002, 'editor', '2023-01-01 00:00:00', '2023-01-01 00:00:00'); -- Full contributor (should not show in shared datasets)

-- Org 101 dataset access  
INSERT INTO "101".dataset_user (dataset_id, user_id, role, created_at, updated_at) VALUES
(1, 9001, 'viewer', '2023-01-02 00:00:00', '2023-01-02 00:00:00'), -- Guest access to beta1
(2, 9001, 'viewer', '2023-01-02 00:00:00', '2023-01-02 00:00:00'), -- Guest access to beta2
(1, 9003, 'viewer', '2023-01-02 00:00:00', '2023-01-02 00:00:00'); -- Guest access to beta1
-- Note: beta3 has no user access granted, so it won't appear in results