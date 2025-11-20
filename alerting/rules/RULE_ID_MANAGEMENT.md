# Detection Rule ID Management

## Overview

All detection rules in TelHawk use **deterministic UUIDs** based on the rule name. This ensures:
- **Consistency across environments**: Same rule name = same UUID everywhere (dev, staging, prod)
- **Git-trackable**: IDs are committed to version control in `.id` files
- **Validation enforced**: Alerting service **WILL HALT** if .id files don't match generated UUIDs

## UUID Generation Algorithm

UUIDs are generated using **UUID v5 (SHA-1 based)**:

```go
namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // DNS namespace
ruleID := uuid.NewSHA1(namespace, []byte("telhawk:builtin:"+ruleName))
```

**Example:**
- Rule name: `failed_logins`
- Generated ID: `263aedcb-9f25-5798-a316-38e039d1d3fb`

This is **deterministic** - the same rule name always produces the same UUID.

## File Structure

Each rule requires **TWO files**:

1. **`rule_name.json`** - The rule definition
2. **`rule_name.json.id`** - The deterministic UUID (one line, UUID only)

**Example:**
```
alerting/rules/
├── failed_logins.json
├── failed_logins.json.id         # Contains: 263aedcb-9f25-5798-a316-38e039d1d3fb
├── port_scanning.json
├── port_scanning.json.id         # Contains: 02628c90-d915-5da9-af84-21e714b68cc6
└── ...
```

## Creating New Rules

### Step 1: Create the Rule JSON

Create `alerting/rules/your_rule_name.json`:

```json
{
  "name": "your_rule_name",
  "description": "What this rule detects",
  "model": { ... },
  "view": { ... },
  "controller": { ... }
}
```

**CRITICAL:** The `"name"` field must match the filename (without `.json` extension).

### Step 2: Generate the .id File

Run the ID generator tool:

```bash
./bin/generate-rule-ids alerting/rules
```

This will:
- ✓ Generate deterministic UUIDs for all rules
- ✓ Create/update .id files
- ✓ Skip rules that already have correct IDs
- ✓ Warn if updating an existing ID

### Step 3: Commit Both Files to Git

```bash
git add alerting/rules/your_rule_name.json
git add alerting/rules/your_rule_name.json.id
git commit -m "Add detection rule: your_rule_name"
```

**BOTH files MUST be committed** - the alerting service validates .id files on startup.

## Validation & Safety

### Startup Validation

When the alerting service starts, it:

1. Reads each `.json` rule file
2. Generates the expected UUID from the rule name
3. Reads the corresponding `.json.id` file
4. **Compares the IDs**

**If IDs don't match → Service HALTS with error:**

```
CRITICAL: Rule ID validation failed for 'failed_logins':
ID MISMATCH: .id file contains '00000000-0000-0000-0000-000000000000'
but rule name 'failed_logins' generates '263aedcb-9f25-5798-a316-38e039d1d3fb'.
This indicates the .id file is out of sync.
Run tools/generate_rule_ids.go to regenerate .id files, then commit to git.
```

### What This Prevents

- ❌ Accidental ID changes when rules are edited
- ❌ Different UUIDs in different environments
- ❌ Rules being created with wrong IDs
- ❌ ID drift over time
- ❌ Duplicate rules with different IDs

## ID Generator Tool

Location: `/tools/generate_rule_ids/`

**Build:**
```bash
cd tools/generate_rule_ids
go build -o ../../bin/generate-rule-ids .
```

**Usage:**
```bash
# Generate/validate IDs for all rules in directory
./bin/generate-rule-ids alerting/rules

# Output example:
# ✓ failed_logins.json (ID already correct)
# ✓ Created new_rule.json.id with ID: abc123...
# ⚠ old_rule.json (updating ID from xxx to yyy)
#
# Summary: 1 created/updated, 43 already correct, 0 errors
```

**When to run:**
- After creating a new rule
- After renaming a rule (will generate new ID)
- If validation fails on service startup
- Before committing rule changes

## Troubleshooting

### Service won't start - ID validation failed

**Cause:** .id file doesn't match the deterministic UUID for the rule name

**Fix:**
```bash
# Regenerate all .id files
./bin/generate-rule-ids alerting/rules

# Review changes
git diff alerting/rules/*.id

# If changes look correct, commit them
git add alerting/rules/*.id
git commit -m "Fix rule IDs to match deterministic generation"
```

### Rule was renamed - now validation fails

**Expected behavior!** Renaming a rule changes its deterministic UUID.

**Fix:**
```bash
# Update the "name" field in the JSON to match new filename
vim alerting/rules/new_name.json  # Update "name": "new_name"

# Regenerate the .id file
./bin/generate-rule-ids alerting/rules

# The .id will show as changed because the UUID changed
git add alerting/rules/new_name.json alerting/rules/new_name.json.id
git rm alerting/rules/old_name.json alerting/rules/old_name.json.id
git commit -m "Rename rule: old_name -> new_name (UUID changed)"
```

**WARNING:** This creates a NEW rule with a new ID. The old rule will be orphaned in the database.

### Merging branches - ID conflicts

If two branches create rules with the same name:

1. Both will generate the **same UUID** (deterministic!)
2. Git may show conflict on `.id` file
3. Simply run `./bin/generate-rule-ids alerting/rules`
4. The tool will confirm both have the same correct ID

### .id file is missing

**Service will HALT** with error:

```
.id file not found at alerting/rules/rule.json.id -
all rules MUST have committed .id files for deterministic UUIDs across environments
```

**Fix:**
```bash
./bin/generate-rule-ids alerting/rules
git add alerting/rules/*.id
git commit -m "Add missing .id files"
```

## Migration Notes

### Migrating from UUID v7 (time-based) to UUID v5 (name-based)

Old rules used UUID v7 (time-based, non-deterministic). We migrated to UUID v5 for determinism.

**All .id files have been regenerated** with deterministic UUIDs as of commit XXXX.

**Impact:**
- Rule IDs changed (one-time change)
- Existing alerts/cases reference old IDs (orphaned)
- Database cleanup may be needed for old IDs
- **Future rule updates**: IDs will remain stable across environments

## Best Practices

### DO ✅

- Commit both `.json` and `.json.id` files together
- Run `generate-rule-ids` after creating new rules
- Run `generate-rule-ids` before committing rule changes
- Keep rule `"name"` field matching the filename
- Review `.id` changes in git diffs

### DON'T ❌

- Manually edit `.id` files
- Use UUID v7 (time-based) generators
- Create rules without running `generate-rule-ids`
- Rename rules without understanding UUID will change
- Delete `.id` files

## Technical Details

### Why UUID v5 (not v4 or v7)?

| UUID Version | Type | Deterministic? | Use Case |
|--------------|------|----------------|----------|
| v4 | Random | ❌ No | General purpose unique IDs |
| v7 | Time-based | ❌ No (includes timestamp) | Sortable IDs |
| **v5** | **Name-based (SHA-1)** | **✅ Yes** | **Same input = same UUID** |

**We need deterministic UUIDs** so rule IDs are identical across:
- Developer machines
- CI/CD pipelines
- Staging environment
- Production environment

### Validation Code Location

**Importer:** `alerting/internal/importer/importer.go`

```go
func validateRuleID(jsonFilePath, ruleName, expectedID string) error
func generateDeterministicUUID(ruleName string) string
```

**Called during import:**
```go
ruleID := generateDeterministicUUID(rule.Name)
if err := validateRuleID(filePath, rule.Name, ruleID); err != nil {
    return fmt.Errorf("CRITICAL: Rule ID validation failed: %w", err)
}
```

This runs on **every service startup** for **every rule**.

## Summary

1. **Every rule needs a `.json` AND `.json.id` file**
2. **UUIDs are deterministic** (same name = same UUID)
3. **Service validates on startup** and halts on mismatch
4. **Use `generate-rule-ids` tool** for all ID operations
5. **Commit .id files to git** alongside rule JSON files

This system ensures **zero ID drift** across environments and **immediate detection** of ID inconsistencies.
