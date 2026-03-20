create table users(
id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
login varchar(100),
pass_hash varchar(100)
);

comment on column users.id is 'ID записи';
comment on column users.login is 'Логин';
comment on column users.pass_hash is 'Хэш-пароль';