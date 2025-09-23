package controllers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/adamanr/employes_service/internal/entity"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type EmployeeController struct {
	deps *Dependens
}

func NewEmployeeController(deps *Dependens) *EmployeeController {
	return &EmployeeController{
		deps: deps,
	}
}

func (c *EmployeeController) GetEmployees(params *entity.GetEmployeesParams) ([]entity.Employee, error) {
	query := "SELECT * FROM employees WHERE 1=1"
	args := []any{}
	argIdx := 1

	if params != nil {
		if params.Role != nil {
			query += fmt.Sprintf(" AND role = $%d", argIdx)
			args = append(args, *params.Role)
			argIdx++
		}

		if params.DepartmentID != nil {
			query += fmt.Sprintf(" AND department_id = $%d", argIdx)
			args = append(args, *params.DepartmentID)
			argIdx++
		}

		if params.Status != nil {
			query += fmt.Sprintf(" AND status = $%d", argIdx)
			args = append(args, *params.Status)
		}
	}

	rows, err := c.deps.DB.Query(context.Background(), query, args...)
	if err != nil {
		c.deps.Logger.Error("Error querying employees", slog.String("error", err.Error()))
		return nil, err
	}
	defer rows.Close()

	employees, err := pgx.CollectRows(rows, pgx.RowToStructByName[entity.Employee])
	if err != nil {
		c.deps.Logger.Error("Error collecting rows", slog.String("error", err.Error()))
		return nil, err
	}

	for i := range employees {
		employees[i].Password = nil
	}

	return employees, nil
}

func (c *EmployeeController) GetEmployeeByID(id uint64) (*entity.Employee, error) {
	rows, err := c.deps.DB.Query(context.Background(), "SELECT * FROM employees WHERE id = $1", id)
	if err != nil {
		c.deps.Logger.Error("Error querying employee", slog.String("error", err.Error()))
		return nil, err
	}
	defer rows.Close()

	employee, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.Employee])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.deps.Logger.Error("Employee not found", slog.Any("id", id))
			return nil, errors.New("employee not found")
		}

		c.deps.Logger.Error("Error collecting row", slog.String("error", err.Error()))
		return nil, err
	}

	employee.Password = nil

	return &employee, nil
}

func (c *EmployeeController) CreateEmployee(emp entity.Employee) (*entity.Employee, error) {
	if emp.FirstName == "" || emp.LastName == "" || emp.Email == nil || *emp.Email == "" {
		c.deps.Logger.Error("Required fields: first_name, last_name, email", slog.Any("emp", emp))
		return nil, errors.New("required fields: first_name, last_name, email")
	}

	if *emp.Password == "" {
		*emp.Password = "default123"
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(*emp.Password), bcrypt.DefaultCost)
	if err != nil {
		c.deps.Logger.Error("Error hashing password", slog.String("error", err.Error()))
		return nil, err
	}

	hashPassword := string(passwordHash)
	emp.Password = &hashPassword

	query := `SELECT COUNT(*) FROM employees WHERE email = $1 OR (personal_number IS NOT NULL AND personal_number = $2)`

	var exists int
	if err = c.deps.DB.QueryRow(context.Background(), query, emp.Email, emp.PersonalNumber).Scan(&exists); err != nil {
		c.deps.Logger.Error("Error checking uniqueness", slog.String("error", err.Error()))
		return nil, err
	}

	if exists > 0 {
		c.deps.Logger.Error("Employee already exists", slog.String("email", *emp.Email))
		return nil, errors.New("employee already exists")
	}

	c.deps.Logger.Info("Employee created", slog.Any("emp", emp.IsActive))

	now := time.Now()
	query = `INSERT INTO employees (first_name, last_name, middle_name, phone, personal_number, email, password, role, is_active, department_id, position, manager_id, hire_date, fire_date, birthday, address, vacation_days, sick_days, status, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
              RETURNING id`

	if err = c.deps.DB.QueryRow(context.Background(), query,
		emp.FirstName, emp.LastName, emp.MiddleName, emp.Phone, emp.PersonalNumber,
		emp.Email, emp.Password, emp.Role, emp.IsActive, emp.DepartmentID, emp.Position,
		emp.ManagerID, emp.HireDate, emp.FireDate, emp.Birthday, emp.Address,
		emp.VacationDays, emp.SickDays, emp.Status, now, now,
	).Scan(&emp.ID); err != nil {
		c.deps.Logger.Error("Error inserting employee", slog.String("error", err.Error()))
		return nil, err
	}

	emp.CreatedAt = &now
	emp.UpdatedAt = &now

	return &emp, nil
}

func (c *EmployeeController) RequestVacation(id uint64, days uint64) (*entity.Employee, error) {
	if days <= 0 {
		c.deps.Logger.Error("Invalid vacation days", slog.Any("days", days))
		return nil, errors.New("invalid vacation days")
	}

	now := time.Now()
	query := `UPDATE employees 
              SET vacation_days = vacation_days + $1, updated_at = $2 
              WHERE id = $3 
              RETURNING *`

	rows, err := c.deps.DB.Query(context.Background(), query, days, now, id)
	if err != nil {
		c.deps.Logger.Error("Error updating vacation days", slog.String("error", err.Error()))
		return nil, err
	}
	defer rows.Close()

	var updatedEmp entity.Employee
	updatedEmp, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.Employee])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.deps.Logger.Error("Employee not found", slog.String("error", err.Error()))
			return nil, errors.New("employee not found")
		}

		c.deps.Logger.Error("Error collecting row", slog.String("error", err.Error()))
		return nil, err
	}

	updatedEmp.Password = nil

	return &updatedEmp, nil
}

// GetPasswordHash is method to get password for update employee.
func (c *EmployeeController) getPasswordHash(newPassword *string, employeeID uint64) (*string, error) {
	if newPassword != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(*newPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}

		hashStr := string(hash)
		return &hashStr, nil
	}

	query := `SELECT password FROM employees WHERE id = $1`
	var currentHash string
	err := c.deps.DB.QueryRow(context.Background(), query, employeeID).Scan(&currentHash)
	return &currentHash, err
}

func (c *EmployeeController) UpdateEmployee(id uint64, emp entity.Employee) (*entity.Employee, error) {
	if emp.FirstName == "" || emp.LastName == "" || emp.Email == nil || *emp.Email == "" {
		c.deps.Logger.Error("Invalid employee data", slog.String("error", "First name, last name, email are required"))
		return nil, errors.New("invalid employee data")
	}

	passwordHash, err := c.getPasswordHash(emp.Password, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.deps.Logger.Error("Employee not found", slog.String("error", err.Error()))
			return nil, errors.New("employee not found")
		}

		c.deps.Logger.Error("Error getting password", slog.String("error", err.Error()))
		return nil, err
	}

	emp.Password = passwordHash

	query := `SELECT COUNT(*) FROM employees WHERE (email = $1 OR (personal_number IS NOT NULL AND personal_number = $2)) AND id != $3`

	var exists int
	if err = c.deps.DB.QueryRow(context.Background(), query, emp.Email, emp.PersonalNumber, id).Scan(&exists); err != nil {
		c.deps.Logger.Error("Error checking uniqueness", slog.String("error", err.Error()))
		return nil, err
	}

	if exists > 0 {
		c.deps.Logger.Warn("Email or personal number already exists", slog.String("email", *emp.Email))
		return nil, errors.New("email or personal number already exists")
	}

	updatedEmp, err := c.updateEmployeeInDB(&emp, id)
	if err != nil {
		c.deps.Logger.Error("Error updating employee", slog.String("error", err.Error()))
		return nil, err
	}

	return &updatedEmp, nil
}

func (c *EmployeeController) updateEmployeeInDB(emp *entity.Employee, id uint64) (entity.Employee, error) {
	now := time.Now()
	emp.UpdatedAt = &now

	query := `UPDATE employees 
              SET first_name = $1, last_name = $2, middle_name = $3, phone = $4, personal_number = $5, 
                  email = $6, password = $7, role = $8, is_active = $9, department_id = $10, 
                  position = $11, manager_id = $12, hire_date = $13, fire_date = $14, 
                  birthday = $15, address = $16, vacation_days = $17, sick_days = $18, 
                  status = $19, updated_at = $20 
              WHERE id = $21 
              RETURNING *`

	rows, err := c.deps.DB.Query(context.Background(), query,
		emp.FirstName, emp.LastName, emp.MiddleName, emp.Phone, emp.PersonalNumber,
		*emp.Email, emp.Password, emp.Role, emp.IsActive, emp.DepartmentID,
		emp.Position, emp.ManagerID, emp.HireDate, emp.FireDate, emp.Birthday,
		emp.Address, emp.VacationDays, emp.SickDays, emp.Status, emp.UpdatedAt, id)
	if err != nil {
		c.deps.Logger.Error("Error updating employee", slog.String("error", err.Error()))
		return entity.Employee{}, fmt.Errorf("failed to update employee: %w", err)
	}
	defer rows.Close()

	updatedEmp, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.Employee])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.deps.Logger.Error("Employee not found", slog.String("error", err.Error()))
			return entity.Employee{}, errors.New("employee not found")
		}

		c.deps.Logger.Error("Error collecting row", slog.String("error", err.Error()))
		return entity.Employee{}, fmt.Errorf("failed to scan row: %w", err)
	}

	updatedEmp.Password = nil

	return updatedEmp, nil
}

func (c *EmployeeController) DeleteEmployee(id uint64) error {
	result, err := c.deps.DB.Exec(context.Background(), "DELETE FROM employees WHERE id = $1", id)
	if err != nil {
		c.deps.Logger.Error("Error deleting employee", slog.String("error", err.Error()))
		return err
	}

	if result.RowsAffected() == 0 {
		c.deps.Logger.Warn("Employee not found", slog.Any("id", id))
		return errors.New("employee not found")
	}

	return nil
}
