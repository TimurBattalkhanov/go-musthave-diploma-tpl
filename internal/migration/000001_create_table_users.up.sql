-- создание таблицы пользователей
CREATE TABLE users(
    login varchar(255) PRIMARY KEY,
    password varchar(255) NOT NULL,
    balance numeric(15,2) NOT NULL DEFAULT 0,
    withdrawn numeric(15,2) NOT NULL DEFAULT 0
);