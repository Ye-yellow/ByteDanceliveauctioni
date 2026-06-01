ALTER TABLE auction_runtime_projection_shard_offsets
  MODIFY COLUMN shard_id INT NOT NULL;

DELETE FROM auction_runtime_projection_shard_offsets
WHERE shard_id < 0 OR shard_id > 15;
