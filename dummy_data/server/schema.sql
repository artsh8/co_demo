CREATE TABLE IF NOT EXISTS customer (
    id int8 GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    first_name varchar(50) NOT NULL,
    last_name varchar(50) NOT NULL,
    email varchar(320)
);

CREATE TABLE IF NOT EXISTS merchant (
    id int8 GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    name varchar(250) NOT NULL
);

CREATE TABLE IF NOT EXISTS product (
    id int8 GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    merchant_id int8 NOT NULL REFERENCES merchant (id),
    price int8 NOT NULL,
    amount int4 NOT NULL,
    name varchar(250) NOT NULL
);

CREATE INDEX IF NOT EXISTS i_product_merchant_id ON product (merchant_id);

CREATE TABLE IF NOT EXISTS customer_order (
    id int8 GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    customer_id int8 NOT NULL REFERENCES customer (id)
);

CREATE INDEX IF NOT EXISTS i_customer_order_customer_id ON customer_order (customer_id);

CREATE TABLE IF NOT EXISTS product_order (
    id int8 GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
    order_id int8 NOT NULL REFERENCES customer_order (id) ON DELETE CASCADE,
    product_id int8 NOT NULL REFERENCES product (id),
    amount int4 NOT NULL,
    UNIQUE (order_id, product_id)
);

CREATE INDEX IF NOT EXISTS i_product_order_product_id ON product_order (product_id);