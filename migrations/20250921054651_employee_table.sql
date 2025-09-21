-- +goose Up
-- +goose StatementBegin
CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR NOT NULL,
    last_name VARCHAR NOT NULL,
    middle_name VARCHAR,
    phone VARCHAR,
    personal_number VARCHAR,
    email VARCHAR,
    password VARCHAR NOT NULL,
    role VARCHAR NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    department_id INTEGER NOT NULL,
    position VARCHAR,
    manager_id INTEGER,
    hire_date TIMESTAMP NOT NULL,
    fire_date TIMESTAMP,
    birthday TIMESTAMP,
    address TEXT,
    vacation_days INTEGER NOT NULL DEFAULT 28,
    sick_days INTEGER NOT NULL DEFAULT 0,
    status VARCHAR NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_personal_number UNIQUE (personal_number),
    CONSTRAINT unique_email UNIQUE (email),
    CONSTRAINT fk_department_id FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE SET NULL,
    CONSTRAINT fk_manager_id FOREIGN KEY (manager_id) REFERENCES employees(id) ON DELETE SET NULL
);

-- Создание индекса для department_id (для быстрых поисков по департаменту)
CREATE INDEX idx_employees_department_id ON employees(department_id);

-- Создание индекса для manager_id (для быстрых поисков по менеджеру)
CREATE INDEX idx_employees_manager_id ON employees(manager_id);

-- Создание индекса для role (для фильтрации по роли)
CREATE INDEX idx_employees_role ON employees(role);

-- Создание индекса для is_active (для фильтрации активных сотрудников)
CREATE INDEX idx_employees_is_active ON employees(is_active);

-- Создание индекса для status (для фильтрации по статусу)
CREATE INDEX idx_employees_status ON employees(status);

-- Создание индекса для created_at (для сортировки/фильтрации по дате создания)
CREATE INDEX idx_employees_created_at ON employees(created_at);

-- Создание индекса для hire_date (для сортировки/фильтрации по дате найма)
CREATE INDEX idx_employees_hire_date ON employees(hire_date);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_employees_department_id;
DROP INDEX IF EXISTS idx_employees_manager_id;
DROP INDEX IF EXISTS idx_employees_role;
DROP INDEX IF EXISTS idx_employees_is_active;
DROP INDEX IF EXISTS idx_employees_status;
DROP INDEX IF EXISTS idx_employees_created_at;
DROP INDEX IF EXISTS idx_employees_hire_date;
DROP TABLE IF EXISTS employees;
-- +goose StatementEnd