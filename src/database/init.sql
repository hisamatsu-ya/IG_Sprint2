-- src/database/init.sql

-- Расширения для UUID (одно из двух точно сработает)
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Пользователи
CREATE TABLE IF NOT EXISTS users (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name       TEXT NOT NULL,
  email      TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Фильмы
CREATE TABLE IF NOT EXISTS movies (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title      TEXT NOT NULL,
  year       INT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Платежи
CREATE TABLE IF NOT EXISTS payments (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID,
  amount     NUMERIC(10,2) NOT NULL DEFAULT 0,
  status     TEXT NOT NULL DEFAULT 'created',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Подписки
CREATE TABLE IF NOT EXISTS subscriptions (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID,
  plan       TEXT NOT NULL,
  status     TEXT NOT NULL DEFAULT 'active',
  started_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
