create table if not exists users(
id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
login varchar(100) NOT NULL UNIQUE,
pass_hash varchar(100) NOT NULL
);

comment on column users.id is 'ID записи';
comment on column users.login is 'Логин';
comment on column users.pass_hash is 'Хэш-пароль';

create table if not exists orders(
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    order_number varchar(50) NOT NULL,
    status varchar(20) NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL ,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    accrual  BIGINT, -- для хранения сотых
    UNIQUE (order_number)
);

create table if not exists withdraw(
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    order_number varchar(50) NOT NULL,
    withdraw BIGINT, 
    processed_at TIMESTAMPTZ,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE
);