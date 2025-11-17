-- Prevent JSONB columns from containing JSON null value
-- This ensures that model, view, and controller are always valid JSON objects

-- First, delete any existing rows with JSON null values
DELETE FROM detection_schemas WHERE model::text = 'null' OR view::text = 'null' OR controller::text = 'null';

-- Add constraints to prevent future JSON null values
ALTER TABLE detection_schemas ADD CONSTRAINT model_not_json_null CHECK (model::text != 'null');
ALTER TABLE detection_schemas ADD CONSTRAINT view_not_json_null CHECK (view::text != 'null');
ALTER TABLE detection_schemas ADD CONSTRAINT controller_not_json_null CHECK (controller::text != 'null');
