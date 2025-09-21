-- +goose Up
-- +goose StatementBegin
ALTER TABLE departments
ADD CONSTRAINT fk_head_id FOREIGN KEY (head_id) REFERENCES employees(id) ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE departments
DROP CONSTRAINT IF EXISTS fk_head_id;
-- +goose StatementEnd