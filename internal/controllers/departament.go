package controllers

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/adamanr/employes_service/internal/entity"
	"github.com/jackc/pgx/v5"
)

type DepartmentController struct {
	deps *Dependens
}

func NewDepartmentController(deps *Dependens) *DepartmentController {
	return &DepartmentController{
		deps: deps,
	}
}

func (c *DepartmentController) GetDepartments() ([]entity.Department, error) {
	query := `SELECT id, name, description, parent_id, head_id, created_at, updated_at FROM departments`

	rows, err := c.deps.DB.Query(context.Background(), query)
	if err != nil {
		c.deps.Logger.Error("Error querying departments", slog.String("error", err.Error()))
		return nil, err
	}
	defer rows.Close()

	departments, err := pgx.CollectRows(rows, pgx.RowToStructByName[entity.Department])
	if err != nil {
		c.deps.Logger.Error("Error collecting rows", slog.String("error", err.Error()))
		return nil, err
	}

	return departments, nil
}

func (c *DepartmentController) GetDepartmentByID(id uint64) (*entity.Department, error) {
	query := `SELECT id, name, description, parent_id, head_id, created_at, updated_at FROM departments WHERE id = $1`

	rows, err := c.deps.DB.Query(context.Background(), query, id)
	if err != nil {
		c.deps.Logger.Error("Error querying department", slog.String("error", err.Error()))
		return nil, err
	}
	defer rows.Close()

	department, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.Department])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.deps.Logger.Error("Department not found", slog.Any("id", id))
			return nil, errors.New("department not found")
		}

		c.deps.Logger.Error("Error collecting row", slog.String("error", err.Error()))
		return nil, err
	}

	return &department, nil
}

func (c *DepartmentController) CreateDepartment(dept entity.Department) (*entity.Department, error) {
	if dept.Name == "" {
		c.deps.Logger.Warn("Name is required", slog.String("name", dept.Name))
		return nil, errors.New("name is required")
	}

	now := time.Now()
	query := `INSERT INTO departments (name, description, parent_id, head_id, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6)
              RETURNING id`

	if err := c.deps.DB.QueryRow(context.Background(), query, dept.Name, dept.Description, dept.ParentID, dept.HeadID, now, now).Scan(&dept.ID); err != nil {
		c.deps.Logger.Error("Error inserting department", slog.String("error", err.Error()))
		return nil, err
	}

	dept.CreatedAt = now
	dept.UpdatedAt = now

	return &dept, nil
}

func (c *DepartmentController) UpdateDepartment(dept entity.Department, id uint64) (*entity.Department, error) {
	dept.UpdatedAt = time.Now()

	query := `UPDATE departments 
              SET name = $1, description = $2, parent_id = $3, head_id = $4, updated_at = $5 
              WHERE id = $6 
              RETURNING id, name, description, parent_id, head_id, created_at, updated_at`

	if err := c.deps.DB.QueryRow(context.Background(), query, dept.Name, dept.Description, dept.ParentID, dept.HeadID, dept.UpdatedAt, id).Scan(
		&dept.ID, &dept.Name, &dept.Description, &dept.ParentID, &dept.HeadID, &dept.CreatedAt, &dept.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.deps.Logger.Error("Department not found", slog.Any("id", id))
			return nil, errors.New("department not found")
		}

		c.deps.Logger.Error("Error updating department", slog.String("error", err.Error()))
		return nil, err
	}

	return &dept, nil
}

func (c *DepartmentController) DeleteDepartment(id uint64) error {
	result, err := c.deps.DB.Exec(context.Background(), "DELETE FROM departments WHERE id = $1", id)
	if err != nil {
		c.deps.Logger.Error("Error deleting department", slog.String("error", err.Error()))
		return nil
	}

	if result.RowsAffected() == 0 {
		c.deps.Logger.Error("Department not found", slog.Any("id", id))
		return nil
	}

	return nil
}
