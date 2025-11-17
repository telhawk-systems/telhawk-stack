-- Revert constraints that prevent JSON null values

ALTER TABLE detection_schemas DROP CONSTRAINT IF EXISTS controller_not_json_null;
ALTER TABLE detection_schemas DROP CONSTRAINT IF EXISTS view_not_json_null;
ALTER TABLE detection_schemas DROP CONSTRAINT IF EXISTS model_not_json_null;
