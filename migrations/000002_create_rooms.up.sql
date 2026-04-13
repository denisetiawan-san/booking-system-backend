
CREATE TABLE IF NOT EXISTS rooms (
    id          VARCHAR(36)   NOT NULL,
    name        VARCHAR(255)  NOT NULL,  
    description TEXT,                   
    capacity    INT           NOT NULL DEFAULT 1,  

    price_per_night DECIMAL(15,2) NOT NULL,

    is_active   TINYINT(1)    NOT NULL DEFAULT 1,
    created_at  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    
    KEY idx_rooms_is_active (is_active),
    
    CONSTRAINT chk_rooms_price    CHECK (price_per_night > 0),
    CONSTRAINT chk_rooms_capacity CHECK (capacity > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
