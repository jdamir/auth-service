-- 1. Подключаем официальное расширение для работы с криптографией
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 2. Создаем тип данных для разделения пользователей
CREATE TYPE user_type_enum AS ENUM ('REAL', 'DIGITAL');

-- 3. Создаем основную таблицу пользователей
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    user_type user_type_enum NOT NULL DEFAULT 'REAL',
    group_id UUID NOT NULL DEFAULT gen_random_uuid(), 
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 4. Создаем индексы для быстрого поиска при авторизации
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_type ON users(user_type);
CREATE INDEX idx_users_group ON users(group_id); 

-- 5. Вставляем базовых пользователей (База сама сгенерирует точный bcrypt)
INSERT INTO users (email, password_hash, user_type) VALUES
('mobile.user@example.com', crypt('SuperSecret2026', gen_salt('bf', 10)), 'REAL'),
('simulator.bot@example.com', crypt('SuperSecret2026', gen_salt('bf', 10)), 'DIGITAL');

-- 6. Автоматически генерируем 500 цифровых твинов для симулятора
INSERT INTO users (email, password_hash, user_type)
SELECT 
    'twin.' || i || '@example.com' AS email,
    crypt('SuperSecret2026', gen_salt('bf', 10)) AS password_hash,
    'DIGITAL' AS user_type
FROM generate_series(1, 500) AS i;