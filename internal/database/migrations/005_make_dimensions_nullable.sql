-- Make dimensions nullable in instances table for consistency with definitions
-- This allows migration from older databases that may not have dimensions set

ALTER TABLE instances ALTER COLUMN "dimensions" DROP NOT NULL;

---- create above / drop below ----

-- Rollback: Make dimensions NOT NULL again
ALTER TABLE instances ALTER COLUMN "dimensions" SET NOT NULL;
