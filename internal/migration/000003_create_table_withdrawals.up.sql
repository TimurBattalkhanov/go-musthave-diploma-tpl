-- создание таблицы списаний
CREATE TABLE withdrawals(
    id serial PRIMARY KEY,
    order_number text NOT NULL,
    login varchar(255) NOT NULL,
    sum numeric(15,2) NOT NULL,
    processed_at timestamptz NOT NULL DEFAULT now()
);