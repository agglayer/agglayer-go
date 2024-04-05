-- +migrate Up
ALTER TABLE state.monitored_txs
ADD COLUMN num_retries DECIMAL(78, 0) NOT NULL DEFAULT 0;

-- +migrate Down
ALTER TABLE state.monitored_txs DROP COLUMN num_retries;