#!/bin/bash

# Cleanup Script: Remove obsolete migration files
# Date: 2026-04-09
# Purpose: Clean up migrations that are no longer needed

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "════════════════════════════════════════════════════════════════"
echo "  InnerLamp Migration Cleanup"
echo "════════════════════════════════════════════════════════════════"
echo ""
echo "This script will DELETE the following files:"
echo ""
echo "  PostgreSQL Migrations (obsolete):"
echo "    - migrations/010_create_onboarding_tables.up.sql"
echo "    - migrations/010_create_onboarding_tables.down.sql"
echo "    - migrations/011_update_onboarding_schema.up.sql"
echo "    - migrations/011_update_onboarding_schema.down.sql"
echo "    - migrations/016_update_devices_table_comprehensive.up.sql"
echo "    - migrations/016_update_devices_table_comprehensive.down.sql"
echo ""
echo "  MongoDB Seed Scripts (obsolete):"
echo "    - migrations/mongodb/002_seed_declaration_questions.js"
echo ""
echo "  Reason:"
echo "    - 010/011: Used MongoDB for onboarding_sessions (not PostgreSQL)"
echo "    - 016: Superseded by updated 013 (comprehensive devices table)"
echo "    - 002: Declaration questions now stored in PostgreSQL only"
echo ""
echo "════════════════════════════════════════════════════════════════"
echo ""

# Prompt confirmation
read -p "⚠️  Are you SURE you want to DELETE these files? (yes/no): " -r
echo

if [[ ! $REPLY =~ ^yes$ ]]; then
    echo "❌ Aborted by user. No files were deleted."
    exit 0
fi

echo ""
echo "🗑️  Deleting obsolete migration files..."
echo ""

# Delete PostgreSQL migrations 010, 011
if [ -f "$PROJECT_ROOT/migrations/010_create_onboarding_tables.up.sql" ]; then
    rm "$PROJECT_ROOT/migrations/010_create_onboarding_tables.up.sql"
    echo "✅ Deleted: 010_create_onboarding_tables.up.sql"
fi

if [ -f "$PROJECT_ROOT/migrations/010_create_onboarding_tables.down.sql" ]; then
    rm "$PROJECT_ROOT/migrations/010_create_onboarding_tables.down.sql"
    echo "✅ Deleted: 010_create_onboarding_tables.down.sql"
fi

if [ -f "$PROJECT_ROOT/migrations/011_update_onboarding_schema.up.sql" ]; then
    rm "$PROJECT_ROOT/migrations/011_update_onboarding_schema.up.sql"
    echo "✅ Deleted: 011_update_onboarding_schema.up.sql"
fi

if [ -f "$PROJECT_ROOT/migrations/011_update_onboarding_schema.down.sql" ]; then
    rm "$PROJECT_ROOT/migrations/011_update_onboarding_schema.down.sql"
    echo "✅ Deleted: 011_update_onboarding_schema.down.sql"
fi

# Delete PostgreSQL migration 016
if [ -f "$PROJECT_ROOT/migrations/016_update_devices_table_comprehensive.up.sql" ]; then
    rm "$PROJECT_ROOT/migrations/016_update_devices_table_comprehensive.up.sql"
    echo "✅ Deleted: 016_update_devices_table_comprehensive.up.sql"
fi

if [ -f "$PROJECT_ROOT/migrations/016_update_devices_table_comprehensive.down.sql" ]; then
    rm "$PROJECT_ROOT/migrations/016_update_devices_table_comprehensive.down.sql"
    echo "✅ Deleted: 016_update_devices_table_comprehensive.down.sql"
fi

# Delete MongoDB seed script
if [ -f "$PROJECT_ROOT/migrations/mongodb/002_seed_declaration_questions.js" ]; then
    rm "$PROJECT_ROOT/migrations/mongodb/002_seed_declaration_questions.js"
    echo "✅ Deleted: 002_seed_declaration_questions.js"
fi

echo ""
echo "════════════════════════════════════════════════════════════════"
echo "  ✅ Cleanup Complete"
echo "════════════════════════════════════════════════════════════════"
echo ""
echo "📋 Summary of changes:"
echo ""
echo "  Deleted: 7 files total"
echo "    - 6 PostgreSQL migration files (010, 011, 016)"
echo "    - 1 MongoDB seed script (002)"
echo ""
echo "  Remaining migrations:"
echo "    - 012: Clean Architecture schema (updated - no DROP statements)"
echo "    - 013: Comprehensive devices table (updated)"
echo "    - 014: Login audit logs"
echo "    - 015: Login columns for users"
echo "    - 017: Soft delete for users"
echo ""
echo "  Next steps:"
echo "    1. Review updated migration 012 (no DROP statements)"
echo "    2. Review updated migration 013 (comprehensive devices)"
echo "    3. Commit changes:"
echo "       git add migrations/"
echo "       git commit -m 'refactor: cleanup obsolete migrations'"
echo ""
echo "════════════════════════════════════════════════════════════════"
