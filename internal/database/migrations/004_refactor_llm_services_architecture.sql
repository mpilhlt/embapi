-- Refactor LLM services architecture into Definitions and Instances
-- This migration separates service templates (definitions) from user-specific instances


-- I. API Standards


-- Step 1: Ensure required API standards exist before creating definitions
-- These API standards are needed for the default LLM Service Definitions

INSERT INTO api_standards ("api_standard_handle", "description", "key_method", "key_field", "created_at", "updated_at")
VALUES ('openai',
        'OpenAI Embeddings API, Version 1, as documented in https://platform.openai.com/docs/api-reference/embeddings',
        'auth_bearer',
        'Authorization',
        NOW(),
        NOW())
ON CONFLICT ("api_standard_handle") DO NOTHING;

INSERT INTO api_standards ("api_standard_handle", "description", "key_method", "key_field", "created_at", "updated_at")
VALUES ('cohere',
        'Cohere Embed API, Version 2, as documented in https://docs.cohere.com/reference/embed',
        'auth_bearer',
        'Authorization',
        NOW(),
        NOW())
ON CONFLICT ("api_standard_handle") DO NOTHING;

INSERT INTO api_standards ("api_standard_handle", "description", "key_method", "key_field", "created_at", "updated_at")
VALUES ('gemini',
        'Gemini Embeddings API, as documented in https://ai.google.dev/gemini-api/docs/embeddings',
        'auth_bearer',
        'x-goog-api-key',
        NOW(),
        NOW())
ON CONFLICT ("api_standard_handle") DO NOTHING;


-- II. Definitions


-- Step 2: Create the _system user for global definitions
INSERT INTO users ("user_handle", "name", "email", "vdb_key", "created_at", "updated_at")
VALUES ('_system', 'System User', 'system@embapi.internal',
        -- WARNING! There is a function that changes this after migrations. Make sure it does not stay like this!!
        '0000000000000000000000000000000000000000000000000000000000000000',
        NOW(), NOW())
ON CONFLICT ("user_handle") DO NOTHING;

-- Step 3: Create LLM Service Definitions table (templates that can be shared)
CREATE TABLE IF NOT EXISTS definitions(
  "definition_id" SERIAL PRIMARY KEY,
  "definition_handle" VARCHAR(20) NOT NULL,
  "owner" VARCHAR(20) NOT NULL REFERENCES "users"("user_handle") ON DELETE CASCADE,
  "endpoint" TEXT NOT NULL,
  "description" TEXT,
  "api_standard" VARCHAR(20) NOT NULL REFERENCES "api_standards"("api_standard_handle"),
  "model" TEXT NOT NULL,
  "dimensions" INTEGER,
  "context_limit" INTEGER,
  "is_public" BOOLEAN NOT NULL DEFAULT FALSE,  -- If true, shared with all users
  "created_at" TIMESTAMP NOT NULL,
  "updated_at" TIMESTAMP NOT NULL,
  UNIQUE ("owner", "definition_handle")
);

-- Step 4: Seed default LLM Service Definitions from _system user and share them with all users
-- These serve as templates that all users can reference

-- 4. (a) OpenAI text-embedding-3-large
INSERT INTO definitions
  ("definition_handle", "owner", "endpoint", "description", "api_standard", "model", "dimensions", "context_limit", "is_public", "created_at", "updated_at")
VALUES 
  ('openai-large',
   '_system',
   'https://api.openai.com/v1/embeddings', 
   'OpenAI text-embedding-3-large service (3072 dimensions)', 
   'openai',
   'text-embedding-3-large',
   3072,
   8192,
   TRUE,
   NOW(),
   NOW())
ON CONFLICT ("owner", "definition_handle") DO NOTHING;

-- 4. (b) OpenAI text-embedding-3-small
INSERT INTO definitions
  ("definition_handle", "owner", "endpoint", "description", "api_standard", "model", "dimensions", "context_limit", "is_public", "created_at", "updated_at")
VALUES 
  ('openai-small',
   '_system',
   'https://api.openai.com/v1/embeddings', 
   'OpenAI text-embedding-3-small service (1536 dimensions)', 
   'openai',
   'text-embedding-3-small',
   1536,
   8191,
   TRUE,
   NOW(),
   NOW())
ON CONFLICT ("owner", "definition_handle") DO NOTHING;

-- 4. (c) Cohere embed-v4.0
INSERT INTO definitions
  ("definition_handle", "owner", "endpoint", "description", "api_standard", "model", "dimensions", "context_limit", "is_public", "created_at", "updated_at")
VALUES 
  ('cohere-v4',
   '_system',
   'https://api.cohere.com/v2/embed',
   'Cohere embed-v4.0 service (1536 dimensions)',
   'cohere',
   'embed-v4.0',
   1536,
   128000,
   TRUE,
   NOW(),
   NOW())
ON CONFLICT ("owner", "definition_handle") DO NOTHING;

-- 4. (d) Google Gemini embedding-001
INSERT INTO definitions
  ("definition_handle", "owner", "endpoint", "description", "api_standard", "model", "dimensions", "context_limit", "is_public", "created_at", "updated_at")
VALUES 
  ('gemini-embedding-001',
   '_system', 
   'https://generativelanguage.googleapis.com/v1beta/models/gemini-embedding-001:embedContent', 
   'Google Gemini embedding-001 service (3072 dimensions)', 
   'gemini',
   'gemini-embedding-001',
   3072,
   2048,
   TRUE,
   NOW(),
   NOW())
ON CONFLICT ("owner", "definition_handle") DO NOTHING;

-- Step 5: Create table for definitions sharing (n:m relationship between instances and users)
CREATE TABLE IF NOT EXISTS definitions_shared_with(
  "user_handle" VARCHAR(20) NOT NULL REFERENCES "users"("user_handle") ON DELETE CASCADE,
  "definition_id" INTEGER NOT NULL REFERENCES "definitions"("definition_id") ON DELETE CASCADE,
  "created_at" TIMESTAMP NOT NULL,
  "updated_at" TIMESTAMP NOT NULL,
  PRIMARY KEY ("user_handle", "definition_id")
);

-- Step 6: Create indexes on definitions_shared_with for efficient lookups
CREATE INDEX IF NOT EXISTS definitions_shared_with_user_idx ON "definitions_shared_with"("user_handle");
CREATE INDEX IF NOT EXISTS definitions_shared_with_definition_idx ON "definitions_shared_with"("definition_id");


-- III. Instances


-- Step 7: Rename existing instances table to instances
ALTER TABLE llm_services RENAME TO instances;

-- Step 8: Fix columns in instances table (rename id and handle, add definition_id, context_limit, and api_key_encrypted, drop api_key)
ALTER TABLE instances RENAME COLUMN llm_service_id TO instance_id;
ALTER TABLE instances RENAME COLUMN llm_service_handle TO instance_handle;
ALTER TABLE instances DROP COLUMN IF EXISTS api_key;
ALTER TABLE instances ADD COLUMN "context_limit" INTEGER;
ALTER TABLE instances ADD COLUMN "definition_id" INTEGER REFERENCES "definitions"("definition_id") ON DELETE SET NULL;
ALTER TABLE instances ADD COLUMN "api_key_encrypted" BYTEA;

-- Step 9: Update the instances index
DROP INDEX IF EXISTS llm_services_handle;
CREATE INDEX IF NOT EXISTS instances_owner_handle ON "instances"("owner", "instance_handle");

-- Step 10: Create table for instance sharing (n:m relationship between instances and users)
CREATE TABLE IF NOT EXISTS instances_shared_with(
  "user_handle" VARCHAR(20) NOT NULL REFERENCES "users"("user_handle") ON DELETE CASCADE,
  "instance_id" INTEGER NOT NULL REFERENCES "instances"("instance_id") ON DELETE CASCADE,
  "role" VARCHAR(20) NOT NULL REFERENCES "vdb_roles"("vdb_role"),
  "created_at" TIMESTAMP NOT NULL,
  "updated_at" TIMESTAMP NOT NULL,
  PRIMARY KEY ("user_handle", "instance_id")
);

-- Step 11: Drop redundant users_llm_services table
-- Ownership is tracked in instances.owner, sharing is tracked in instances_shared_with, no other table needed
DROP TABLE IF EXISTS users_llm_services;

-- Step 12: Migrate data - Add the new column (nullable initially)
ALTER TABLE projects ADD COLUMN "instance_id" INTEGER REFERENCES "instances"("instance_id") ON DELETE RESTRICT;

-- Step 13: Migrate data - for each project, pick the first linked LLM service instance
-- This is a best-effort migration; admins should verify manually if multiple services were used
UPDATE projects p
SET instance_id = (
    SELECT pls.llm_service_id
    FROM projects_llm_services pls
    WHERE pls.project_id = p.project_id
    ORDER BY pls.created_at
    LIMIT 1
)
WHERE EXISTS (
    SELECT 1 FROM projects_llm_services pls WHERE pls.project_id = p.project_id
);

-- Step 14: Update embeddings table to reference instance_id
-- and Update foreign key constraint
ALTER TABLE embeddings RENAME COLUMN llm_service_id TO instance_id;
ALTER TABLE embeddings DROP CONSTRAINT IF EXISTS embeddings_llm_service_id_fkey;
ALTER TABLE embeddings ADD CONSTRAINT embeddings_instance_id_fkey FOREIGN KEY (instance_id) REFERENCES instances(instance_id);

-- Step 15: Drop the old projects_llm_services table (many-to-many, no longer needed)
-- Projects now have exactly one instance via the instance_id column
DROP TABLE IF EXISTS projects_llm_services;


---- create above / drop below ----


-- Rollback instructions (reverse order)

-- Step 15: Restore projects_llm_services table
CREATE TABLE IF NOT EXISTS projects_llm_services(
  "project_id" SERIAL NOT NULL REFERENCES "projects"("project_id") ON DELETE CASCADE,
  "instance_id" SERIAL NOT NULL REFERENCES "instances"("instance_id") ON DELETE CASCADE,
  "created_at" TIMESTAMP NOT NULL,
  "updated_at" TIMESTAMP NOT NULL,
  PRIMARY KEY ("project_id", "instance_id")
);

-- Step 14: Rename embeddings column back
ALTER TABLE embeddings RENAME COLUMN instance_id TO llm_service_id;

-- Step 13: ??

-- Step 12: Remove the single instance reference from projects
ALTER TABLE projects DROP COLUMN IF EXISTS instance_id;

-- Step 11: Restore users_llm_services table (rollback)
CREATE TABLE IF NOT EXISTS users_llm_services(
  "user_handle" VARCHAR(20) NOT NULL REFERENCES "users"("user_handle") ON DELETE CASCADE,
  "llm_service_id" SERIAL NOT NULL REFERENCES "instances"("llm_service_id") ON DELETE CASCADE,
  "role" VARCHAR(20) NOT NULL REFERENCES "vdb_roles"("vdb_role"),
  "created_at" TIMESTAMP NOT NULL,
  "updated_at" TIMESTAMP NOT NULL,
  PRIMARY KEY ("user_handle", "llm_service_id")
);

-- Step 10: Drop instance sharing table
DROP TABLE IF EXISTS instances_shared_with;

-- Step 9: Restore index name
DROP INDEX IF EXISTS instances_handle;
CREATE INDEX IF NOT EXISTS llm_services_handle ON "instances"("llm_service_handle");

-- Step 8: Remove new columns from instances
ALTER TABLE instances DROP COLUMN IF EXISTS api_key_encrypted;
ALTER TABLE instances DROP COLUMN IF EXISTS definition_id;
ALTER TABLE instances DROP COLUMN IF EXISTS context_limit;
ALTER TABLE instaces ADD COLUMN "api_key" TEXT;

-- Step 7: Rename instances table back to instances
ALTER TABLE instances RENAME COLUMN instance_handle TO llm_service_handle;
ALTER TABLE instances RENAME COLUMN instance_id TO llm_service_id;
ALTER TABLE instances RENAME TO instances;

-- Step 6: Drop indexes on definitions_shared_with
DROP INDEX IF EXISTS definitions_shared_with_user_idx;
DROP INDEX IF EXISTS definitions_shared_with_definition_idx;

-- Step 5: Drop definitions sharing table
DROP TABLE IF EXISTS definitions_shared_with;

-- Step 4: Drop seeded definitions
DELETE FROM definitions WHERE owner = '_system';

-- Step 3: Drop definitions table
DROP TABLE IF EXISTS definitions;

-- Step 2: Remove _system user
DELETE FROM users WHERE user_handle = '_system';

-- Step 1: Ensure required API standards exist before creating definitions
DELETE FROM api_standards WHERE api_standard_handle IN ('openai', 'cohere', 'gemini');
