## Query Language
  * **Documentation:** docs/QUERY_LANGUAGE_DESIGN.md ✅
- [ ] Phase 3: Text Syntax Parser (Future)
  * Design text syntax grammar (field:value, operators, boolean logic)
  * Implement parser using participle (Go parser library)
  * Text syntax → JSON query AST translation
  * Field name aliases for user convenience (user → .actor.user.name, src_ip → .src_endpoint.ip)
  * Support wildcards, CIDR notation, comparison operators
  * Grouping and precedence with parentheses
  * API accepts both text and JSON input formats
  * UI "Advanced Query" mode with text input and syntax highlighting
  * Parser error messages with helpful suggestions
- [ ] Phase 4: Saved Searches
  * Database schema for saved searches (PostgreSQL JSONB storage)
  * API endpoints: POST/GET/PUT/DELETE /api/v1/searches
  * Store queries as canonical JSON (version-safe, portable)
  * Save search metadata: name, description, owner, sharing permissions
  * UI: Save/Load buttons in search console
  * Share searches with other users (view/edit/admin permissions)
  * Search templates library (pre-built queries for common use cases)
  * Version history for saved searches (track changes over time)
- [ ] Phase 5: S3 Cold Storage Integration (Long-term)
  * JSON query → S3/Parquet predicate translation
  * Time range → partition pruning logic (year/month/day partitions)
  * Select clause → Parquet column projection
  * Filter conditions → Parquet row group filtering
  * Query router: time-based tier selection (hot/warm/cold)
  * Integration with DuckDB or AWS Athena for S3 queries
  * Result merging across multiple tiers
  * Performance optimization: parallel partition scans