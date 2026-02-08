-- Generate go code with: sqlc generate

-- sqlc creates Go functions from the SQL commands below, using annotations
-- (beginning with "-- name:") to derive function names and result types.
-- In the end of the annotation, :one means single result, :many means multiple results, :exec means no results.

-- The conventions for function names used in this project are as follows:
-- - "Get" functions return lists of objects as identifiers or minimal metadata,
-- - "Retrieve" functions return single objects with full object data.
-- - "Upsert" functions insert or update objects and return only identifiers or minimal metadata.
-- - "Delete" functions delete objects and return no data.
-- - "Link..." and "Unlink..." functions create or remove associations between objects.
-- - "Is..." functions return boolean values.
-- - "Count..." functions return counts of objects.
-- - "...All..." functions return all objects of a type without filtering (or perform an action on all records).
-- - "...By..." functions return objects filtered by some association.


-- === USERS ===


-- name: UpsertUser :one
INSERT
INTO users (
  "user_handle", "name", "email", "vdb_key", "created_at", "updated_at"
) VALUES (
  $1, $2, $3, $4, NOW(), NOW()
)
ON CONFLICT ("user_handle") DO UPDATE SET
  "name" = EXCLUDED."name",
  "email" = EXCLUDED."email",
  "vdb_key" = EXCLUDED."vdb_key",
  "updated_at" = NOW()
RETURNING users."user_handle";

-- name: DeleteUser :exec
DELETE
FROM users
WHERE "user_handle" = $1;

-- name: RetrieveUser :one
SELECT *
FROM users
WHERE "user_handle" = $1 LIMIT 1;

-- name: GetAllUsers :many
SELECT "user_handle"
FROM users
ORDER BY "user_handle" ASC LIMIT $1 OFFSET $2;

-- name: GetUsersByProject :many
SELECT users."user_handle", users_projects."role"
FROM users JOIN users_projects
ON users."user_handle" = users_projects."user_handle"
JOIN projects ON users_projects."project_id" = projects."project_id"
WHERE projects."owner" = $1 AND projects."project_handle" = $2
ORDER BY users."user_handle" ASC LIMIT $3 OFFSET $4;

-- name: GetKeyByUser :one
SELECT "vdb_key"
FROM users
WHERE "user_handle" = $1 LIMIT 1;

-- name: GetUserByVDBKey :one
SELECT "user_handle"
FROM users
WHERE "vdb_key" = $1 LIMIT 1;

-- name: GetKeysByProject :many
SELECT users."user_handle", users_projects."role", users."vdb_key"
FROM users
JOIN users_projects
ON users."user_handle" = users_projects."user_handle"
JOIN projects
ON users_projects."project_id" = projects."project_id"
WHERE projects."owner" = $1
AND projects."project_handle" = $2
ORDER BY users."user_handle" ASC LIMIT $3 OFFSET $4;

-- name: GetKeysByDefinition :many
SELECT users."user_handle", users."vdb_key"
FROM users
JOIN definitions_shared_with
ON users."user_handle" = definitions_shared_with."user_handle"
JOIN definitions
ON definitions_shared_with."definition_id" = definitions."definition_id"
WHERE definitions."owner" = $1
AND definitions."definition_handle" = $2
ORDER BY users."user_handle" ASC LIMIT $3 OFFSET $4;

-- name: GetKeysByInstance :many
SELECT users."user_handle", instances_shared_with."role", users."vdb_key"
FROM users
JOIN instances_shared_with
ON users."user_handle" = instances_shared_with."user_handle"
JOIN instances
ON instances_shared_with."instance_id" = instances."instance_id"
WHERE instances."owner" = $1
AND instances."instance_handle" = $2
ORDER BY users."user_handle" ASC LIMIT $3 OFFSET $4;


-- === PROJECTS ===


-- name: UpsertProject :one
INSERT
INTO projects (
  "project_handle", "owner", "description", "metadata_scheme", "public_read", "instance_id", "created_at", "updated_at"
) VALUES (
  $1, $2, $3, $4, $5, $6, NOW(), NOW()
)
ON CONFLICT ("owner", "project_handle") DO UPDATE SET
  "description" = EXCLUDED."description",
  "metadata_scheme" = EXCLUDED."metadata_scheme",
  "public_read" = EXCLUDED."public_read",
  "instance_id" = EXCLUDED."instance_id",
  "updated_at" = NOW()
RETURNING "project_id", "owner", "project_handle";

-- name: DeleteProject :exec
DELETE
FROM projects
WHERE "owner" = $1
AND "project_handle" = $2;

-- name: GetProjectsByUser :many
SELECT projects."owner",
  projects."project_handle",
  projects."project_id",
  projects."public_read",
  users_projects."role"
FROM projects
JOIN users_projects
ON projects."project_id" = users_projects."project_id"
WHERE users_projects."user_handle" = $1
ORDER BY projects."project_handle" ASC LIMIT $2 OFFSET $3;

-- name: GetAccessibleProjectsByUser :many
SELECT projects."owner",
  projects."project_handle",
  projects."project_id",
  projects."public_read",
  CASE
    WHEN projects."owner" = $1 THEN 'owner'
    ELSE users_projects."role"
  END AS "role"
FROM projects
LEFT JOIN users_projects
ON projects."project_id" = users_projects."project_id"
WHERE users_projects."user_handle" = $1
OR projects."owner" = $1
ORDER BY projects."owner" ASC 
LIMIT $2 OFFSET $3;

-- name: RetrieveProject :one
SELECT *
FROM projects
WHERE "owner" = $1
AND "project_handle" = $2
LIMIT 1;

-- name: RetrieveProjectForUser :one
SELECT projects.*, users_projects."role"
FROM projects
LEFT JOIN users_projects
ON projects."project_id" = users_projects."project_id"
WHERE projects."owner" = $1
AND projects."project_handle" = $2
AND (users_projects."user_handle" = $3 OR projects."public_read" = TRUE)
LIMIT 1;

-- name: IsProjectPubliclyReadable :one
SELECT "public_read"
FROM projects
WHERE "owner" = $1
AND "project_handle" = $2
LIMIT 1;

-- name: GetAllProjects :many
SELECT projects."owner", projects."project_handle"
FROM projects
ORDER BY "owner" ASC, "project_handle" ASC LIMIT $1 OFFSET $2;

-- name: CountAllProjects :one
SELECT COUNT(*)
FROM projects;

-- name: LinkProjectToUser :one
INSERT
INTO users_projects (
  "user_handle", "project_id", "role", "created_at", "updated_at"
) VALUES (
  $1, $2, $3, NOW(), NOW()
)
ON CONFLICT ("user_handle", "project_id") DO UPDATE SET
  "role" = EXCLUDED."role",
  "updated_at" = NOW()
RETURNING users_projects."user_handle", users_projects."project_id";

-- name: UnlinkProjectFromUser :exec
DELETE
FROM users_projects
WHERE "user_handle" = $1
AND "project_id" = $2;


-- === LLM Service Definitions (user-shared templates) ===


-- name: UpsertDefinition :one
INSERT
INTO definitions (
  "owner", "definition_handle", "endpoint", "description", "api_standard", "model", "dimensions", "context_limit", "is_public", "created_at", "updated_at"
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW()
)
ON CONFLICT ("owner", "definition_handle") DO UPDATE SET
  "endpoint" = EXCLUDED."endpoint",
  "description" = EXCLUDED."description",
  "api_standard" = EXCLUDED."api_standard",
  "model" = EXCLUDED."model",
  "dimensions" = EXCLUDED."dimensions",
  "context_limit" = EXCLUDED."context_limit",
  "is_public" = EXCLUDED."is_public",
  "updated_at" = NOW()
RETURNING "owner", "definition_handle", "definition_id", "is_public";

-- name: DeleteDefinition :exec
DELETE
FROM definitions
WHERE "owner" = $1
AND "definition_handle" = $2;

-- name: RetrieveDefinition :one
SELECT *
FROM definitions
WHERE "owner" = $1
AND "definition_handle" = $2
LIMIT 1;

-- name: GetDefinitionsByUser :many
SELECT definitions."definition_handle", definitions."definition_id"
FROM definitions
WHERE "owner" = $1
ORDER BY "definition_handle" ASC LIMIT $2 OFFSET $3;

-- name: GetAllDefinitions :many
SELECT definitions."owner", definitions."definition_handle", definitions."definition_id"
FROM definitions
ORDER BY "owner" ASC, "definition_handle" ASC LIMIT $1 OFFSET $2;

-- name: GetSystemDefinitions :many
SELECT definitions."definition_handle", definitions."definition_id"
FROM definitions
WHERE "owner" = '_system'
ORDER BY "definition_handle" ASC LIMIT $1 OFFSET $2;

-- name: LinkDefinitionToUser :exec
INSERT
INTO definitions_shared_with (
  "user_handle", "definition_id", "created_at", "updated_at"
) VALUES (
  $1, $2, NOW(), NOW()
)
ON CONFLICT ("user_handle", "definition_id") DO NOTHING;

-- name: UnlinkDefinition :exec
DELETE
FROM definitions_shared_with
WHERE "user_handle" = $1
AND "definition_id" = $2;

-- name: GetSharedUsersForDefinition :many
SELECT  definitions_shared_with."user_handle"
FROM definitions_shared_with
JOIN definitions
ON definitions."definition_id" = definitions_shared_with."definition_id"
WHERE definitions."owner" = $1
  AND definitions."definition_handle" = $2
  AND definitions_shared_with."user_handle" != '*'
ORDER BY "user_handle" ASC LIMIT $3 OFFSET $4;

-- name: GetAccessibleDefinitionsByUser :many
-- Get all definitions accessible to a user (owned + shared + _system)
-- Returns definitions with metadata indicating ownership
SELECT  definitions."owner",
        definitions."definition_handle",
        definitions."definition_id",
        definitions."is_public"
FROM definitions
LEFT JOIN definitions_shared_with
  ON definitions."definition_id" = definitions_shared_with."definition_id"
WHERE definitions."owner" = $1
   OR definitions."owner" = '_system'
   OR definitions_shared_with."user_handle" = $1
ORDER BY definitions."owner" ASC, definitions."definition_handle" ASC 
LIMIT $2 OFFSET $3;


-- === LLM Service Instances (user-specific instances with optional API keys) ===


-- name: UpsertInstance :one
INSERT
INTO instances (
  "owner", "instance_handle", "definition_id", "endpoint", "description", "api_key_encrypted", "api_standard", "model", "dimensions", "context_limit", "created_at", "updated_at"
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW()
)
ON CONFLICT ("owner", "instance_handle") DO UPDATE SET
  "definition_id" = EXCLUDED."definition_id",
  "endpoint" = EXCLUDED."endpoint",
  "description" =  EXCLUDED."description",
  "api_key_encrypted" = EXCLUDED."api_key_encrypted",
  "api_standard" = EXCLUDED."api_standard",
  "model" = EXCLUDED."model",
  "dimensions" = EXCLUDED."dimensions",
  "context_limit" = EXCLUDED."context_limit",
  "updated_at" = NOW()
RETURNING "owner", "instance_handle", "instance_id";

-- name: UpsertInstanceFromDefinition :one
INSERT
INTO instances (
  "owner", "instance_handle", "definition_id", "endpoint", "description", "api_key_encrypted", "api_standard", "model", "dimensions", "context_limit", "created_at", "updated_at"
)
SELECT
  $1 as "owner",
  $2 as "instance_handle", 
  def."definition_id",
  COALESCE($3, def."endpoint") as "endpoint",
  COALESCE($4, def."description") as "description",
  $5 as "api_key_encrypted",
  COALESCE($6, def."api_standard") as "api_standard",
  COALESCE($7, def."model") as "model",
  COALESCE($8::INTEGER, def."dimensions") as "dimensions",
  COALESCE($9::INTEGER, def."context_limit") as "context_limit",
  NOW() as "created_at",
  NOW() as "updated_at"
FROM definitions def
WHERE def."owner" = $9 AND def."definition_handle" = $10
ON CONFLICT ("owner", "instance_handle") DO UPDATE SET
  "definition_id" = EXCLUDED."definition_id",
  "endpoint" = EXCLUDED."endpoint",
  "description" = EXCLUDED."description",
  "api_key_encrypted" = EXCLUDED."api_key_encrypted",
  "api_standard" = EXCLUDED."api_standard",
  "model" = EXCLUDED."model",
  "dimensions" = EXCLUDED."dimensions",
  "context_limit" = EXCLUDED."context_limit",
  "updated_at" = NOW()
RETURNING "owner", "instance_handle", "instance_id";

-- name: DeleteInstance :exec
DELETE
FROM instances
WHERE "owner" = $1
AND "instance_handle" = $2;

-- name: RetrieveInstance :one
SELECT  instances."owner",
        instances."instance_handle",
        instances."instance_id",
        instances."definition_id",
        definitions."owner" AS "definition_owner",
        definitions."definition_handle" AS "definition_handle",
        instances."endpoint",
        instances."description",
        -- when api_key_encrypted is not null, return true
        CASE
          WHEN instances."api_key_encrypted" IS NULL THEN TRUE
          ELSE FALSE
        END AS "has_api_key",
        instances."api_standard",
        instances."model",
        instances."dimensions",
        instances."context_limit",
        instances."created_at",
        instances."updated_at"
FROM instances
LEFT JOIN definitions
ON instances."definition_id" = definitions."definition_id"
WHERE instances."owner" = $1
AND instances."instance_handle" = $2
LIMIT 1;

-- name: RetrieveInstanceByID :one
SELECT  instances."owner",
        instances."instance_handle",
        instances."instance_id",
        instances."definition_id",
        instances."endpoint",
        instances."description",
        instances."api_standard",
        instances."model",
        instances."dimensions",
        instances."context_limit",
        instances."created_at",
        instances."updated_at"
FROM instances
WHERE "instance_id" = $1
LIMIT 1;

-- name: RetrieveInstanceByProject :one
SELECT  instances."owner",
        instances."instance_handle",
        instances."instance_id",
        instances."definition_id",
        instances."endpoint",
        instances."description",
        instances."api_standard",
        instances."model",
        instances."dimensions",
        instances."context_limit",
        instances."created_at",
        instances."updated_at"
FROM instances
JOIN projects
ON projects."instance_id" = instances."instance_id"
WHERE projects."owner" = $1
  AND projects."project_handle" = $2
LIMIT 1;

-- name: RetrieveInstanceByProjectForUser :one
SELECT  instances."owner",
        instances."instance_handle",
        instances."instance_id",
        instances."definition_id",
        instances."endpoint",
        instances."description",
        instances."api_standard",
        instances."model",
        instances."dimensions",
        instances."context_limit",
        instances."created_at",
        instances."updated_at",
        instances_shared_with."role" AS "access_role"
FROM instances
JOIN projects
ON projects."instance_id" = instances."instance_id"
LEFT JOIN instances_shared_with
ON instances_shared_with."instance_id" = instances."instance_id"
WHERE projects."owner" = $1
  AND projects."project_handle" = $2
  AND instances_shared_with."user_handle" = $3
LIMIT 1;

-- name: RetrieveInstanceByProjectID :one
SELECT  instances."owner",
        instances."instance_handle",
        instances."instance_id",
        instances."definition_id",
        instances."endpoint",
        instances."description",
        instances."api_standard",
        instances."model",
        instances."dimensions",
        instances."context_limit",
        instances."created_at",
        instances."updated_at"
FROM instances
JOIN projects
ON projects."instance_id" = instances."instance_id"
WHERE projects."project_id" = $1
LIMIT 1;

-- name: LinkInstanceToUser :exec
INSERT
INTO instances_shared_with (
  "user_handle", "instance_id", "role", "created_at", "updated_at"
) VALUES (
  $1, $2, $3, NOW(), NOW()
)
ON CONFLICT ("user_handle", "instance_id") DO UPDATE SET
  "role" = EXCLUDED."role",
  "updated_at" = NOW();

-- name: UnlinkInstance :exec
DELETE
FROM instances_shared_with
WHERE "user_handle" = $1
AND "instance_id" = $2;

-- name: GetSharedUsersForInstance :many
SELECT  instances_shared_with."user_handle",
        instances_shared_with."role"
FROM instances_shared_with
JOIN instances
ON instances."instance_id" = instances_shared_with."instance_id"
WHERE instances."owner" = $1
  AND instances."instance_handle" = $2
ORDER BY "user_handle" ASC LIMIT $3 OFFSET $4;

-- name: RetrieveSharedInstance :one
-- Get single instance, but only if it is shared with requesting user
SELECT  instances."owner",
        instances."instance_handle",
        instances."definition_id",
        instances."endpoint",
        instances."description",
        instances."api_standard",
        instances."model",
        instances."dimensions",
        instances."context_limit",
        instances."created_at",
        instances."updated_at"
FROM instances
JOIN instances_shared_with
ON instances."instance_id" = instances_shared_with."instance_id"
WHERE (instances_shared_with."user_handle" = $1 AND instances."owner" = $2 AND instances."instance_handle" = $3)
LIMIT 1;

-- Get all instances owned by a user
-- name: GetInstancesByUser :many
SELECT  instances."owner",
        instances."instance_handle",
        instances."instance_id"
FROM instances
WHERE instances."owner" = $1
ORDER BY instances."instance_handle" ASC LIMIT $2 OFFSET $3;

-- name: GetSharedInstancesByUser :many
SELECT  instances."owner",
        instances."instance_handle",
        instances."instance_id",
        instances_shared_with."role"
FROM instances
JOIN instances_shared_with
ON instances."instance_id" = instances_shared_with."instance_id"
WHERE instances_shared_with."user_handle" = $1
ORDER BY instances_shared_with."role" ASC, instances."owner" ASC, instances."instance_handle" ASC 
LIMIT $2 OFFSET $3;

-- name: GetAccessibleInstancesByUser :many
-- Get all instances accessible to a user (owned + shared)
-- Returns instances with metadata indicating ownership
SELECT  instances."owner",
        instances."instance_handle",
        instances."instance_id",
        CASE 
          WHEN instances."owner" = $1 THEN 'owner'
          ELSE instances_shared_with."role"
        END as "role",
        instances."owner" = $1 as "is_owner"
FROM instances
LEFT JOIN instances_shared_with
  ON instances."instance_id" = instances_shared_with."instance_id"
WHERE instances."owner" = $1
   OR instances_shared_with."user_handle" = $1
ORDER BY instances."owner" ASC, instances."instance_handle" ASC 
LIMIT $2 OFFSET $3;

-- name: CountInstancesByUser :one
SELECT COUNT(*)
FROM instances
WHERE "owner" = $1;


-- === EMBEDDINGS ===


-- name: UpsertEmbeddings :one
INSERT
INTO embeddings (
  "text_id", "owner", "project_id", "instance_id", "text", "vector", "vector_dim", "metadata", "created_at", "updated_at"
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()
)
ON CONFLICT ("text_id", "owner", "project_id", "instance_id") DO UPDATE SET
  "text" = $5,
  "vector" = $6,
  "vector_dim" = $7,
  "metadata" = $8,
  "updated_at" = NOW()
RETURNING "embeddings_id", "text_id", "owner", "project_id", "instance_id";

-- name: DeleteEmbeddingsByID :exec
DELETE
FROM embeddings
WHERE "embeddings_id" = $1;

-- name: DeleteEmbeddingsByProject :exec
DELETE
FROM embeddings
USING embeddings AS e
JOIN projects AS p
ON e."project_id" = p."project_id"
WHERE embeddings."owner" = $1
AND embeddings."project_id" = e."project_id"
AND p."project_handle" = $2;

-- name: DeleteEmbeddingsByDocID :exec
DELETE FROM embeddings e
USING projects p
WHERE e."owner" = $1
  AND e."project_id" = p."project_id"
  AND p."project_handle" = $2
  AND e."text_id" = $3;

-- name: RetrieveEmbeddings :one
SELECT embeddings.*, projects."project_handle", instances."instance_handle"
FROM embeddings
JOIN instances
ON embeddings."instance_id" = instances."instance_id"
JOIN projects
ON embeddings."project_id" = projects."project_id"
WHERE embeddings."owner" = $1
AND projects."project_handle" = $2
AND embeddings."text_id" = $3
LIMIT 1;

-- name: RetrieveEmbeddingsByID :one
SELECT embeddings.*, projects."project_handle", instances."instance_handle"
FROM embeddings
JOIN instances
ON embeddings."instance_id" = instances."instance_id"
JOIN projects
ON embeddings."project_id" = projects."project_id"
WHERE embeddings."embeddings_id" = $1
LIMIT 1;

-- name: GetEmbeddingsByProject :many
SELECT embeddings."embeddings_id", embeddings."text_id", projects."owner", projects."project_handle", instances."instance_handle"
FROM embeddings
JOIN instances
ON instances."instance_id" = embeddings."instance_id"
JOIN projects
ON projects."project_id" = embeddings."project_id"
WHERE embeddings."owner" = $1
AND projects."project_handle" = $2
ORDER BY embeddings."text_id" ASC LIMIT $3 OFFSET $4;

-- name: CountEmbeddingsByProject :one
SELECT COUNT(*)
FROM embeddings
JOIN projects
ON embeddings."project_id" = projects."project_id"
WHERE embeddings."owner" = $1
AND projects."project_handle" = $2;

-- name: CountAllEmbeddings :one
SELECT COUNT(*)
FROM embeddings;


-- === SIMILARITY SEARCH ===


-- name: GetSimilarsByVector :many
SELECT embeddings."embeddings_id", embeddings."text_id", instances."owner", instances."instance_handle"
FROM embeddings
JOIN instances
ON embeddings."instance_id" = instances."instance_id"
ORDER BY "vector" <=> $1
LIMIT $2 OFFSET $3;

-- name: GetSimilarsByID :many
SELECT e2."text_id", (1 - (e1.vector <=> e2.vector))::float8 AS similarity
FROM embeddings e1
CROSS JOIN embeddings e2
JOIN projects
ON e1."project_id" = projects."project_id"
WHERE e2."embeddings_id" != e1."embeddings_id"
  AND e1."text_id" = $1
  AND e1."owner" = $2
  AND projects."project_handle" = $3
  AND e1."vector_dim" = e2."vector_dim"
  AND e1."project_id" = e2."project_id"
  AND 1 - (e1.vector <=> e2.vector) >= $4::double precision
ORDER BY e1.vector <=> e2.vector
LIMIT $5 OFFSET $6;

-- name: GetSimilarsByIDWithFilter :many
SELECT e2."text_id", (1 - (e1.vector <=> e2.vector))::float8 AS similarity
FROM embeddings e1
CROSS JOIN embeddings e2
JOIN projects
ON e1."project_id" = projects."project_id"
WHERE e2."embeddings_id" != e1."embeddings_id"
  AND e1."text_id" = $1
  AND e1."owner" = $2
  AND projects."project_handle" = $3
  AND e1."vector_dim" = e2."vector_dim"
  AND e1."project_id" = e2."project_id"
  AND 1 - (e1.vector <=> e2.vector) >= $4::double precision
  AND (e2."metadata" ->> $5::text IS NULL OR trim(e2."metadata" ->> $5::text) <> trim($6::text))
ORDER BY e1.vector <=> e2.vector
LIMIT $7 OFFSET $8;

-- name: GetSimilarsByVectorWithProject :many
SELECT e."text_id", (1 - (e.vector <=> $3::halfvec))::float8 AS similarity
FROM embeddings e
JOIN projects p
ON e."project_id" = p."project_id"
WHERE p."owner" = $1
  AND p."project_handle" = $2
  AND 1 - (e.vector <=> $3::halfvec) >= $4::double precision
ORDER BY e.vector <=> $3::halfvec
LIMIT $5 OFFSET $6;

-- name: GetSimilarsByVectorWithProjectAndFilter :many
SELECT e."text_id", (1 - (e.vector <=> $3::halfvec))::float8 AS similarity
FROM embeddings e
JOIN projects p
ON e."project_id" = p."project_id"
WHERE p."owner" = $1
  AND p."project_handle" = $2
  AND 1 - (e.vector <=> $3::halfvec) >= $4::double precision
  AND (e."metadata" ->> $5::text IS NULL OR trim(e."metadata" ->> $5::text) <> trim($6::text))
ORDER BY e.vector <=> $3::halfvec
LIMIT $7 OFFSET $8;


-- === API STANDARDS ===


-- name: UpsertAPIStandard :one
INSERT
INTO api_standards (
  "api_standard_handle", "description", "key_method", "key_field", "created_at", "updated_at"
) VALUES (
  $1, $2, $3, $4, NOW(), NOW()
)
ON CONFLICT ("api_standard_handle") DO UPDATE SET
  "description" = $2,
  "key_method" = $3,
  "key_field" = $4,
  "updated_at" = NOW()
RETURNING "api_standard_handle";

-- name: DeleteAPIStandard :exec
DELETE
FROM api_standards
WHERE "api_standard_handle" = $1;

-- name: RetrieveAPIStandard :one
SELECT *
FROM api_standards
WHERE "api_standard_handle" = $1 LIMIT 1;

-- name: GetAPIStandards :many
SELECT api_standards."api_standard_handle"
FROM api_standards
ORDER BY "api_standard_handle" ASC LIMIT $1 OFFSET $2;

-- name: CheckIfAPIStandardInUse :one
SELECT EXISTS (
  SELECT 1
  FROM definitions
  WHERE "api_standard" = $1
  LIMIT 1
);

-- === ADMIN QUERIES ===


-- name: ResetAllSerials :exec
DO $$
DECLARE
    seq_name text;
    max_id bigint;
BEGIN
    FOR seq_name IN
      SELECT sequence_name
      FROM information_schema.sequences
      WHERE sequence_schema = 'public' AND sequence_name LIKE '%_seq'
    LOOP
        -- For definitions table, set sequence to max preserved definition_id + 1
        IF seq_name = 'definitions_definition_id_seq' THEN
            SELECT COALESCE(MAX("definition_id"), 0) INTO max_id
            FROM definitions
            WHERE "owner" = '_system';
            EXECUTE format('ALTER SEQUENCE public.%I RESTART WITH %s', seq_name, max_id + 1);
        -- For all other sequences, reset to 1
        ELSE
            EXECUTE format('ALTER SEQUENCE public.%I RESTART WITH 1', seq_name);
        END IF;
    END LOOP;
END $$;

-- name: DeleteAllRecords :exec
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT table_name 
        FROM information_schema.tables 
        WHERE table_schema = 'public'
          AND table_name NOT IN ('key_methods', 'vdb_roles', 'api_standards') -- preserve static reference data
    LOOP
        -- Preserve _system user and its definitions
        IF r.table_name = 'users' THEN
            EXECUTE format('DELETE FROM %I WHERE "user_handle" != ''_system'';', r.table_name);
        ELSIF r.table_name = 'definitions' THEN
            EXECUTE format('DELETE FROM %I WHERE "owner" != ''_system'';', r.table_name);
        ELSE
            EXECUTE format('DELETE FROM %I;', r.table_name);
        END IF;
    END LOOP;
END $$;
