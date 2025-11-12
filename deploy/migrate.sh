#!/bin/bash
# Database migration script for DigitalOcean App Platform
# Run this after the database is created to initialize the schema

set -e

echo "Running database migrations..."

# Check if DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    echo "Error: DATABASE_URL environment variable is not set"
    echo "Please set it to your PostgreSQL connection string"
    echo "Example: postgres://user:password@host:port/database"
    exit 1
fi

# Run the migration SQL file
psql "$DATABASE_URL" -f migrations/0001_init.sql

echo "Migration completed successfully!"

