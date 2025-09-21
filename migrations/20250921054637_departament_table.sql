-- +goose Up
-- +goose StatementBegin
CREATE TABLE departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR NOT NULL,
    description TEXT,
    parent_id INTEGER,
    head_id INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_parent_id FOREIGN KEY (parent_id) REFERENCES departments(id) ON DELETE SET NULL
);

-- Создание индекса для name (уникальность)
CREATE UNIQUE INDEX idx_departments_name ON departments(name);

-- Создание индекса для parent_id (для быстрых поисков по иерархии)
CREATE INDEX idx_departments_parent_id ON departments(parent_id);

-- Создание индекса для head_id (для быстрых поисков по руководителю)
CREATE INDEX idx_departments_head_id ON departments(head_id);

-- Создание индекса для created_at (для сортировки/фильтрации по дате)
CREATE INDEX idx_departments_created_at ON departments(created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_departments_name;
DROP INDEX IF EXISTS idx_departments_parent_id;
DROP INDEX IF EXISTS idx_departments_head_id;
DROP INDEX IF EXISTS idx_departments_created_at;
DROP TABLE departments;
-- +goose StatementEnd