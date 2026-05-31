-- Durable Redis runtime projection offsets.

CREATE TABLE IF NOT EXISTS auction_runtime_projection_offsets (
  lot_id VARCHAR(64) PRIMARY KEY,
  room_id VARCHAR(64) NOT NULL,
  last_projected_version BIGINT NOT NULL DEFAULT 0,
  last_stream_id VARCHAR(64) NOT NULL DEFAULT '',
  updated_at_unix_ms BIGINT NOT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  INDEX idx_runtime_projection_room (room_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
