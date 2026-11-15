-- создание таблицы заказов
CREATE TABLE orders(
    number text PRIMARY KEY,
    login varchar(255) NOT NULL,
    status varchar(20) NOT NULL DEFAULT 'NEW',
    accrual numeric(15,2),
    uploaded_at timestamptz NOT NULL DEFAULT now()
);