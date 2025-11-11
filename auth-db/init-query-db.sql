-- Create database for query service (saved searches, alerts, dashboards)
-- This script runs automatically on first startup of the PostgreSQL container

-- Check if database already exists before creating
SELECT 'CREATE DATABASE telhawk_query'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'telhawk_query')\gexec
