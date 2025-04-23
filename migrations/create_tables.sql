-- Table for orders
CREATE TABLE orders (
    order_uid VARCHAR(255) PRIMARY KEY,
    track_number VARCHAR(255),
    entry VARCHAR(50),
    locale VARCHAR(10),
    internal_signature VARCHAR(255),
    customer_id VARCHAR(255),
    delivery_service VARCHAR(50),
    shardkey VARCHAR(10),
    sm_id INT,
    date_created TIMESTAMP,
    oof_shard VARCHAR(10)
);

-- Table for delivery details
CREATE TABLE delivery (
    order_uid VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255),
    phone VARCHAR(20),
    zip VARCHAR(20),
    city VARCHAR(100),
    address VARCHAR(255),
    region VARCHAR(100),
    email VARCHAR(255),
    FOREIGN KEY (order_uid) REFERENCES orders(order_uid)
);

-- Table for payment details
CREATE TABLE payment (
    transaction_id VARCHAR(255) PRIMARY KEY,
    order_uid VARCHAR(255),
    request_id VARCHAR(255),
    currency VARCHAR(10),
    provider VARCHAR(50),
    amount DECIMAL(10, 2),
    payment_dt BIGINT,
    bank VARCHAR(100),
    delivery_cost DECIMAL(10, 2),
    goods_total DECIMAL(10, 2),
    custom_fee DECIMAL(10, 2),
    FOREIGN KEY (order_uid) REFERENCES orders(order_uid)
);

-- Table for items
CREATE TABLE items (
    chrt_id BIGINT PRIMARY KEY,
    order_uid VARCHAR(255),
    track_number VARCHAR(255),
    price DECIMAL(10, 2),
    rid VARCHAR(255),
    name VARCHAR(255),
    sale INT,
    size VARCHAR(10),
    total_price DECIMAL(10, 2),
    nm_id BIGINT,
    brand VARCHAR(255),
    status INT,
    FOREIGN KEY (order_uid) REFERENCES orders(order_uid)
);

-- Insert data into orders table
INSERT INTO orders (
    order_uid, track_number, entry, locale, internal_signature, customer_id, 
    delivery_service, shardkey, sm_id, date_created, oof_shard
) VALUES (
    'b563feb7b2b84b6test', 'WBILMTESTTRACK', 'WBIL', 'en', '', 'test',
    'meest', '9', 99, '2021-11-26T06:22:19Z', '1'
);

-- Insert data into delivery table
INSERT INTO delivery (
    order_uid, name, phone, zip, city, address, region, email
) VALUES (
    'b563feb7b2b84b6test', 'Test Testov', '+9720000000', '2639809', 'Kiryat Mozkin',
    'Ploshad Mira 15', 'Kraiot', 'test@gmail.com'
);

-- Insert data into payment table
INSERT INTO payment (
    transaction_id, order_uid, request_id, currency, provider, amount, 
    payment_dt, bank, delivery_cost, goods_total, custom_fee
) VALUES (
    'b563feb7b2b84b6test', 'b563feb7b2b84b6test', '', 'USD', 'wbpay', 1817, 
    1637907727, 'alpha', 1500, 317, 0
);

-- Insert data into items table
INSERT INTO items (
    chrt_id, order_uid, track_number, price, rid, name, sale, size, 
    total_price, nm_id, brand, status
) VALUES (
    9934930, 'b563feb7b2b84b6test', 'WBILMTESTTRACK', 453, 'ab4219087a764ae0btest',
    'Mascaras', 30, '0', 317, 2389212, 'Vivienne Sabo', 202
);
