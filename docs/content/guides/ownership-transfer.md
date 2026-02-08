---
title: "Ownership Transfer Guide"
weight: 4
---

# Ownership Transfer Guide

This guide explains how to transfer project ownership between users in embapi.

## Overview

Project ownership transfer allows you to reassign full control of a project from one user to another. This is useful when:
- A project maintainer is leaving and wants to hand over control
- Organizational changes require reassigning project ownership
- Consolidating projects under a different user account
- Transferring stewardship of research data to a new PI

## Important Constraints

Before transferring ownership, understand these constraints:

1. **Only the current owner can transfer** - Editors and readers cannot initiate transfers
2. **New owner must exist** - The target user must already be registered in the system
3. **No handle conflicts** - The new owner cannot already have a project with the same handle
4. **Old owner loses access** - After transfer, the original owner has no access to the project
5. **Data remains intact** - All embeddings and metadata are preserved during transfer
6. **Shared users remain** - Existing sharing relationships are maintained

## Transferring Ownership

### Basic Transfer

Transfer a project to another user:

```bash
curl -X POST "https://api.example.com/v1/projects/alice/research-data/transfer-ownership" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{
    "new_owner_handle": "bob"
  }'
```

**Response:**
```json
{
  "message": "Project ownership transferred successfully",
  "project_handle": "research-data",
  "old_owner": "alice",
  "new_owner": "bob"
}
```

After this operation:
- The project is now accessible at `/v1/projects/bob/research-data`
- Bob has full owner privileges
- Alice no longer has any access to the project
- All embeddings remain unchanged

### Complete Transfer Example

Here's a complete workflow showing before and after transfer:

```bash
# Before transfer - Alice is the owner
curl -X GET "https://api.example.com/v1/projects/alice/my-project" \
  -H "Authorization: Bearer alice_api_key"

# Response
{
  "project_handle": "my-project",
  "owner": "alice",
  "description": "Research project",
  "instance_id": 123
}

# Alice transfers to Bob
curl -X POST "https://api.example.com/v1/projects/alice/my-project/transfer-ownership" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"new_owner_handle": "bob"}'

# After transfer - Bob is now the owner
curl -X GET "https://api.example.com/v1/projects/bob/my-project" \
  -H "Authorization: Bearer bob_api_key"

# Response
{
  "project_handle": "my-project",
  "owner": "bob",
  "description": "Research project",
  "instance_id": 123
}

# Alice can no longer access it
curl -X GET "https://api.example.com/v1/projects/alice/my-project" \
  -H "Authorization: Bearer alice_api_key"
# Returns: 404 Not Found
```

## Effects of Transfer

### Project Access Path Changes

The project URL changes to reflect the new owner:

**Before:**
```
/v1/projects/alice/research-data
/v1/embeddings/alice/research-data
/v1/similars/alice/research-data/doc123
```

**After:**
```
/v1/projects/bob/research-data
/v1/embeddings/bob/research-data
/v1/similars/bob/research-data/doc123
```

**Important:** Update all client code and bookmarks to use the new owner's handle.

### New Owner Gains Full Control

Bob (new owner) can now:
- View and modify all embeddings
- Update project settings (description, instance, metadata schema)
- Manage sharing (add/remove shared users)
- Transfer ownership again to someone else
- Delete the project

### Old Owner Loses All Access

Alice (old owner) can no longer:
- Access the project in any way
- View or modify embeddings
- See project metadata
- Manage sharing
- Transfer ownership back

**Note:** If Alice needs continued access, Bob should share the project with her after the transfer.

### Shared Users Remain

If the project was shared with other users, those sharing relationships are preserved:

```bash
# Before transfer - Alice shares with Charlie (editor)
curl -X POST "https://api.example.com/v1/projects/alice/research-data/share" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"share_with_handle": "charlie", "role": "editor"}'

# Transfer to Bob
curl -X POST "https://api.example.com/v1/projects/alice/research-data/transfer-ownership" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"new_owner_handle": "bob"}'

# After transfer - Charlie still has editor access
curl -X GET "https://api.example.com/v1/projects/bob/research-data" \
  -H "Authorization: Bearer charlie_api_key"
# Works! Charlie can still access as editor
```

### Upgrading Shared User to Owner

If the new owner was previously a shared user, their role is automatically upgraded:

```bash
# Alice shares project with Bob (editor)
curl -X POST "https://api.example.com/v1/projects/alice/my-project/share" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"share_with_handle": "bob", "role": "editor"}'

# Alice transfers ownership to Bob
curl -X POST "https://api.example.com/v1/projects/alice/my-project/transfer-ownership" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"new_owner_handle": "bob"}'

# Bob's previous "editor" sharing role is removed
# Bob now has full owner privileges instead
```

## Use Cases

### PI Leaving Institution

A principal investigator leaving an institution transfers project ownership to a colleague:

```bash
curl -X POST "https://api.example.com/v1/projects/prof_jones/lab_data/transfer-ownership" \
  -H "Authorization: Bearer prof_jones_api_key" \
  -H "Content-Type: application/json" \
  -d '{"new_owner_handle": "prof_smith"}'
```

### Account Consolidation

Consolidate multiple projects under a single organizational account:

```bash
# Transfer Alice's personal projects to organization account
curl -X POST "https://api.example.com/v1/projects/alice/project1/transfer-ownership" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"new_owner_handle": "org_datascience"}'

curl -X POST "https://api.example.com/v1/projects/alice/project2/transfer-ownership" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"new_owner_handle": "org_datascience"}'
```

### Graduated Student Handoff

A graduating student transfers their research project to their advisor:

```bash
curl -X POST "https://api.example.com/v1/projects/student_bob/thesis_embeddings/transfer-ownership" \
  -H "Authorization: Bearer student_bob_api_key" \
  -H "Content-Type: application/json" \
  -d '{"new_owner_handle": "advisor_carol"}'

# Advisor can then share it back with the student if needed
curl -X POST "https://api.example.com/v1/projects/advisor_carol/thesis_embeddings/share" \
  -H "Authorization: Bearer advisor_carol_api_key" \
  -H "Content-Type: application/json" \
  -d '{"share_with_handle": "student_bob", "role": "reader"}'
```

### Department Reorganization

Projects move to a new department owner:

```bash
curl -X POST "https://api.example.com/v1/projects/old_dept/resource_library/transfer-ownership" \
  -H "Authorization: Bearer old_dept_api_key" \
  -H "Content-Type: application/json" \
  -d '{"new_owner_handle": "new_dept"}'
```

## Error Conditions

### New Owner Doesn't Exist

```bash
curl -X POST "https://api.example.com/v1/projects/alice/my-project/transfer-ownership" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"new_owner_handle": "nonexistent_user"}'
```

**Error:**
```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "User 'nonexistent_user' does not exist"
}
```

**Solution:** Ensure the target user is registered first. Contact admin to create the user.

### Handle Conflict

```bash
# Bob already has a project called "research-data"
curl -X POST "https://api.example.com/v1/projects/alice/research-data/transfer-ownership" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"new_owner_handle": "bob"}'
```

**Error:**
```json
{
  "title": "Conflict",
  "status": 409,
  "detail": "User 'bob' already has a project with handle 'research-data'"
}
```

**Solution:** Either:
1. Rename Alice's project before transferring:
   ```bash
   curl -X PATCH "https://api.example.com/v1/projects/alice/research-data" \
     -H "Authorization: Bearer alice_api_key" \
     -H "Content-Type: application/json" \
     -d '{"project_handle": "research-data-alice"}'
   ```
2. Ask Bob to rename or delete their existing project
3. Choose a different target user

### Not the Owner

```bash
# Charlie tries to transfer Alice's project
curl -X POST "https://api.example.com/v1/projects/alice/research-data/transfer-ownership" \
  -H "Authorization: Bearer charlie_api_key" \
  -H "Content-Type: application/json" \
  -d '{"new_owner_handle": "bob"}'
```

**Error:**
```json
{
  "title": "Forbidden",
  "status": 403,
  "detail": "Only the project owner can transfer ownership"
}
```

**Solution:** Only the current owner (Alice) can initiate the transfer.

## Best Practices

### Before Transferring

1. **Communicate with New Owner**: Ensure they're willing to accept ownership
2. **Document Current State**: Export or document current embeddings and metadata
3. **Review Shared Users**: Check who has access and whether sharing should continue
4. **Update Client Code**: Identify all systems accessing the project that need updating
5. **Backup Data**: Consider exporting important data before transfer

### During Transfer

1. **Transfer at Low-Activity Time**: Minimize disruption by transferring during quiet periods
2. **Test Access First**: Verify new owner can access their other projects
3. **Use Correct Handle**: Double-check the new owner's handle before submitting

### After Transferring

1. **Verify New Ownership**: Confirm the transfer succeeded
2. **Update Client Applications**: Change all API calls to use new owner handle
3. **Grant Back Access if Needed**: New owner can share project back to old owner
4. **Update Documentation**: Update any documentation referencing the project path
5. **Notify Shared Users**: Inform shared users about the path change

## Maintaining Access After Transfer

If the original owner needs continued access, the new owner should share the project:

```bash
# Step 1: Alice transfers to Bob
curl -X POST "https://api.example.com/v1/projects/alice/research-data/transfer-ownership" \
  -H "Authorization: Bearer alice_api_key" \
  -H "Content-Type: application/json" \
  -d '{"new_owner_handle": "bob"}'

# Step 2: Bob shares back with Alice as editor
curl -X POST "https://api.example.com/v1/projects/bob/research-data/share" \
  -H "Authorization: Bearer bob_api_key" \
  -H "Content-Type: application/json" \
  -d '{"share_with_handle": "alice", "role": "editor"}'

# Now Alice can still access (but as editor, not owner)
curl -X GET "https://api.example.com/v1/embeddings/bob/research-data" \
  -H "Authorization: Bearer alice_api_key"
```

## Checking Current Owner

To verify current project ownership:

```bash
curl -X GET "https://api.example.com/v1/projects/{owner}/{project}" \
  -H "Authorization: Bearer your_api_key"
```

The `owner` field in the response shows the current owner.

## Related Documentation

- [Project Sharing Guide](./project-sharing.md) - Share projects with specific users
- [Public Projects Guide](./public-projects.md) - Make projects publicly accessible

## Troubleshooting

### Cannot Find Project After Transfer

**Problem:** Getting 404 after transfer

**Solution:** Update the owner in your API calls:
- Old: `/v1/projects/alice/my-project`
- New: `/v1/projects/bob/my-project`

### Need to Reverse Transfer

**Problem:** Transferred by mistake, need to reverse

**Solution:** New owner must transfer back:
```bash
curl -X POST "https://api.example.com/v1/projects/bob/my-project/transfer-ownership" \
  -H "Authorization: Bearer bob_api_key" \
  -H "Content-Type: application/json" \
  -d '{"new_owner_handle": "alice"}'
```

### Client Applications Failing

**Problem:** Applications can't access project after transfer

**Solution:** Update all hardcoded owner references in your code to use the new owner's handle.
