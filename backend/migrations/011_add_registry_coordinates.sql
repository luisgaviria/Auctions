-- +goose Up
ALTER TABLE auctions ADD COLUMN IF NOT EXISTS registry_book INTEGER;
ALTER TABLE auctions ADD COLUMN IF NOT EXISTS registry_page INTEGER;
