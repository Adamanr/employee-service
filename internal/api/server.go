package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"employes_service/internal/config"
	"employes_service/internal/entity"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmployeeNotFound = errors.New("employee not found")
	ErrPasswordHashing  = errors.New("password hashing failed")
)

const (
	TokenSize = 16
)

type Server struct {
	Cfg    *config.Config
	DB     *pgx.Conn
	Redis  *redis.Client
	Logger *slog.Logger
}

func NewServer(cfg *config.Config, db *pgx.Conn, redis *redis.Client, logger *slog.Logger) *Server {
	return &Server{
		Cfg:    cfg,
		DB:     db,
		Redis:  redis,
		Logger: logger,
	}
}

var _ ServerInterface = Server{}

type Claims struct {
	jwt.RegisteredClaims

	ID      uint   `json:"id"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	TokenID string `json:"token_id"`
}

func generateTokenID(logger *slog.Logger) (string, error) {
	b := make([]byte, TokenSize)
	if _, err := rand.Read(b); err != nil {
		logger.Error("Error generating token ID", slog.String("error", err.Error()))
		return "", err
	}

	return hex.EncodeToString(b), nil
}

// getUserFromToken extracts user information from the token.
func (s Server) getUserFromToken(r *http.Request) (*Claims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		s.Logger.Error("Authorization header missing")
		return nil, errors.New("authorization header missing")
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenStr == authHeader {
		s.Logger.Error("Invalid bearer token", slog.String("token", tokenStr))
		return nil, errors.New("invalid bearer token")
	}

	ctx := context.Background()
	if err := s.Redis.Get(ctx, "access_token:"+tokenStr).Err(); errors.Is(err, redis.Nil) {
		s.Logger.Warn("Token revoked", slog.String("token", tokenStr))
		return nil, errors.New("token revoked")
	}

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(_ *jwt.Token) (any, error) {
		return []byte(s.Cfg.Server.JWTSecret), nil
	})
	if err != nil {
		s.Logger.Error("Error parsing token", slog.String("error", err.Error()))
		return nil, errors.New("invalid token")
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (s Server) requireAdminOrHR(user *Claims) error {
	if user.Role != "admin" && user.Role != "hr" {
		return errors.New("insufficient permissions")
	}

	return nil
}

// AuthLogin authenticates a user and returns a JWT token.
func (s Server) AuthLogin(w http.ResponseWriter, r *http.Request) {
	var req entity.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"}, "error")
		return
	}

	if req.Email == "" || req.Password == "" {
		s.httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Email and password required"}, "error")
		return
	}

	var emp Employee
	if err := s.DB.QueryRow(context.Background(), "SELECT id, email, password, role FROM employees WHERE email = $1", req.Email).Scan(
		&emp.Id, &emp.Email, &emp.Password, &emp.Role,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.Logger.Warn("Invalid login attempt", slog.String("email", req.Email))
			s.httpResponse(w, http.StatusUnauthorized, "Invalid credentials", "error")
			return
		}

		s.Logger.Error("Error querying employee", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to authenticate", "error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*emp.Password), []byte(req.Password)); err != nil {
		s.Logger.Warn("Invalid password", slog.String("email", req.Email))
		s.httpResponse(w, http.StatusUnauthorized, "Invalid credentials", "error")
		return
	}

	refreshToken, err := s.createToken(emp, "refresh")
	if err != nil {
		s.Logger.Error("Error creating refresh token", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to generate refresh token", "error")
		return
	}

	accessToken, err := s.createToken(emp, "access")
	if err != nil {
		s.Logger.Error("Error creating access token", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to generate access token", "error")
		return
	}

	ctx := context.Background()
	if err = s.Redis.Set(ctx, "access_token:"+accessToken, emp.Id, s.Cfg.Redis.AccessTokenTTL).Err(); err != nil {
		s.Logger.Error("Error saving access token to Redis", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to save access token", "error")
		return
	}

	if err = s.Redis.Set(ctx, "refresh_token:"+refreshToken, emp.Id, s.Cfg.Redis.RefreshTokenTTL).Err(); err != nil {
		s.Logger.Error("Error saving refresh token to Redis", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to save refresh token", "error")
		return
	}

	s.httpResponse(w, http.StatusOK, entity.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, "success")
}

func (s Server) createToken(emp Employee, tokenType string) (string, error) {
	tokenID, err := generateTokenID(s.Logger)
	if err != nil {
		s.Logger.Error("Error generating refresh token ID", slog.String("error", err.Error()))
		return "", err
	}

	expiresAt := s.Cfg.Redis.AccessTokenTTL
	if tokenType == "refresh" {
		expiresAt = s.Cfg.Redis.RefreshTokenTTL
	}

	claims := Claims{
		ID:      *emp.Id,
		Email:   *emp.Email,
		Role:    string(emp.Role),
		TokenID: tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresAt)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(s.Cfg.Server.JWTSecret))
	if err != nil {
		s.Logger.Error("Error signing token", slog.String("error", err.Error()), slog.String("Token type", tokenType))
		return "", err
	}

	return tokenStr, nil
}

// AuthLogout make logout user.
func (s Server) AuthLogout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	ctx := context.Background()
	if err := s.Redis.Del(ctx, "access_token:"+tokenStr).Err(); err != nil {
		s.Logger.Error("Error deleting access token from Redis", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to logout", "error")
		return
	}

	if err := s.Redis.Del(ctx, "refresh_token:*").Err(); err != nil {
		s.Logger.Error("Error deleting refresh tokens from Redis", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to logout", "error")
		return
	}

	s.httpResponse(w, http.StatusOK, map[string]string{"message": "Logged out successfully"}, "success")
}

// GetDepartments get all departments.
func (s Server) GetDepartments(w http.ResponseWriter, r *http.Request) {
	if err := s.checkAuthUser(r); err != nil {
		s.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	query := `SELECT id, name, description, parent_id, head_id, created_at, updated_at FROM departments`

	rows, err := s.DB.Query(context.Background(), query)
	if err != nil {
		s.Logger.Error("Error querying departments", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to query departments", "error")
		return
	}
	defer rows.Close()

	departments, err := pgx.CollectRows(rows, pgx.RowToStructByName[entity.Department])
	if err != nil {
		s.Logger.Error("Error collecting rows", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to scan rows", "error")
		return
	}

	s.httpResponse(w, http.StatusOK, departments, "success")
}

// GetDepartmentByID get department by id.
func (s Server) GetDepartmentByID(w http.ResponseWriter, r *http.Request, id int) {
	if err := s.checkAuthUser(r); err != nil {
		s.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	query := `SELECT id, name, description, parent_id, head_id, created_at, updated_at FROM departments WHERE id = $1`

	rows, err := s.DB.Query(context.Background(), query, id)
	if err != nil {
		s.Logger.Error("Error querying department", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to query department", "error")
		return
	}
	defer rows.Close()

	department, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.Department])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.httpResponse(w, http.StatusNotFound, map[string]string{"error": "Department not found"}, "error")
			return
		}

		s.Logger.Error("Error collecting row", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to scan row", "error")
		return
	}

	s.httpResponse(w, http.StatusOK, department, "success")
}

// CreateDepartment create new department.
func (s Server) CreateDepartment(w http.ResponseWriter, r *http.Request) {
	if err := s.checkAuthUser(r); err != nil {
		s.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	var dept entity.Department
	if err := json.NewDecoder(r.Body).Decode(&dept); err != nil {
		s.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusBadRequest, "Invalid request body", "error")
		return
	}

	if dept.Name == "" {
		s.Logger.Warn("Name is required", slog.String("name", dept.Name))
		s.httpResponse(w, http.StatusBadRequest, "Name is required", "error")
		return
	}

	now := time.Now()
	query := `INSERT INTO departments (name, description, parent_id, head_id, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6)
              RETURNING id`

	if err := s.DB.QueryRow(context.Background(), query, dept.Name, dept.Description, dept.ParentID, dept.HeadID, now, now).Scan(&dept.ID); err != nil {
		s.Logger.Error("Error inserting department", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to create department", "error")
		return
	}

	dept.CreatedAt = now
	dept.UpdatedAt = now

	s.httpResponse(w, http.StatusCreated, dept, "success")
}

// UpdateDepartment update department by id.
func (s Server) UpdateDepartment(w http.ResponseWriter, r *http.Request, id int) {
	if err := s.checkAuthUser(r); err != nil {
		s.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	var dept entity.Department
	if err := json.NewDecoder(r.Body).Decode(&dept); err != nil {
		s.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusBadRequest, "Invalid request body", "error")
		return
	}

	dept.UpdatedAt = time.Now()

	query := `UPDATE departments 
              SET name = $1, description = $2, parent_id = $3, head_id = $4, updated_at = $5 
              WHERE id = $6 
              RETURNING id, name, description, parent_id, head_id, created_at, updated_at`

	if err := s.DB.QueryRow(context.Background(), query, dept.Name, dept.Description, dept.ParentID, dept.HeadID, dept.UpdatedAt, id).Scan(
		&dept.ID, &dept.Name, &dept.Description, &dept.ParentID, &dept.HeadID, &dept.CreatedAt, &dept.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.Logger.Error("Department not found", slog.Int("id", id))
			s.httpResponse(w, http.StatusNotFound, "Department not found", "error")
			return
		}

		s.Logger.Error("Error updating department", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to update department", "error")
		return
	}

	s.httpResponse(w, http.StatusOK, dept, "success")
}

// DeleteDepartment delete department.
func (s Server) DeleteDepartment(w http.ResponseWriter, r *http.Request, id int) {
	if err := s.checkAuthUser(r); err != nil {
		s.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	result, err := s.DB.Exec(context.Background(), "DELETE FROM departments WHERE id = $1", id)
	if err != nil {
		s.Logger.Error("Error deleting department", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to delete department", "error")
		return
	}

	if result.RowsAffected() == 0 {
		s.httpResponse(w, http.StatusNotFound, "Department not found", "error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetEmployees get all employees.
func (s Server) GetEmployees(w http.ResponseWriter, r *http.Request, params GetEmployeesParams) {
	if err := s.checkAuthUser(r); err != nil {
		s.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	query := "SELECT * FROM employees WHERE 1=1"
	args := []any{}
	argIdx := 1

	if params.Role != nil {
		query += fmt.Sprintf(" AND role = $%d", argIdx)
		args = append(args, *params.Role)
		argIdx++
	}

	if params.DepartmentId != nil {
		query += fmt.Sprintf(" AND department_id = $%d", argIdx)
		args = append(args, *params.DepartmentId)
		argIdx++
	}

	if params.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *params.Status)
	}

	rows, err := s.DB.Query(context.Background(), query, args...)
	if err != nil {
		s.Logger.Error("Error querying employees", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to query employees", "error")
		return
	}
	defer rows.Close()

	employees, err := pgx.CollectRows(rows, pgx.RowToStructByName[entity.Employee])
	if err != nil {
		s.Logger.Error("Error collecting rows", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to scan rows", "error")
		return
	}

	for i := range employees {
		employees[i].PasswordHash = ""
	}

	s.httpResponse(w, http.StatusOK, employees, "success")
}

// GetEmployeesByID get employee by id.
func (s Server) GetEmployeesByID(w http.ResponseWriter, r *http.Request, id int) {
	if err := s.checkAuthUser(r); err != nil {
		s.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	rows, err := s.DB.Query(context.Background(), "SELECT * FROM employees WHERE id = $1", id)
	if err != nil {
		s.Logger.Error("Error querying employee", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to query employee", "error")
		return
	}
	defer rows.Close()

	employee, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.Employee])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.Logger.Error("Employee not found", slog.Int("id", id))
			s.httpResponse(w, http.StatusNotFound, "Employee not found", "error")
			return
		}

		s.Logger.Error("Error collecting row", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to scan row", "error")
		return
	}

	employee.PasswordHash = ""

	s.httpResponse(w, http.StatusOK, employee, "success")
}

// CreateEmployee create new employee.
func (s Server) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	if err := s.checkAuthUser(r); err != nil {
		s.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	var emp Employee
	if err := json.NewDecoder(r.Body).Decode(&emp); err != nil {
		s.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusBadRequest, "Invalid request body", "error")
		return
	}

	if emp.FirstName == "" || emp.LastName == "" || emp.Email == nil || *emp.Email == "" {
		s.Logger.Error("Required fields: first_name, last_name, email", slog.Any("emp", emp))
		s.httpResponse(w, http.StatusBadRequest, "Required fields: first_name, last_name, email", "error")
		return
	}

	if *emp.Password == "" {
		*emp.Password = "default123"
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(*emp.Password), bcrypt.DefaultCost)
	if err != nil {
		s.Logger.Error("Error hashing password", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to hash password", "error")
		return
	}

	hashPassword := string(passwordHash)
	emp.Password = &hashPassword

	query := `SELECT COUNT(*) FROM employees WHERE email = $1 OR (personal_number IS NOT NULL AND personal_number = $2)`

	var exists int
	if err = s.DB.QueryRow(context.Background(), query, emp.Email, emp.PersonalNumber).Scan(&exists); err != nil {
		s.Logger.Error("Error checking uniqueness", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to check uniqueness", "error")
		return
	}

	if exists > 0 {
		s.httpResponse(w, http.StatusBadRequest, "Email or personal number already exists", "error")
		return
	}

	now := time.Now()
	query = `INSERT INTO employees (first_name, last_name, middle_name, phone, personal_number, email, password, role, is_active, department_id, position, manager_id, hire_date, fire_date, birthday, address, vacation_days, sick_days, status, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
              RETURNING id`

	if err = s.DB.QueryRow(context.Background(), query,
		emp.FirstName, emp.LastName, emp.MiddleName, emp.Phone, emp.PersonalNumber,
		emp.Email, emp.Password, emp.Role, emp.IsActive, emp.DepartmentId, emp.Position,
		emp.ManagerId, emp.HireDate, emp.FireDate, emp.Birthday, emp.Address,
		emp.VacationDays, emp.SickDays, emp.Status, now, now,
	).Scan(&emp.Id); err != nil {
		s.Logger.Error("Error inserting employee", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to create employee", "error")
		return
	}

	emp.CreatedAt = &now
	emp.UpdatedAt = &now

	s.httpResponse(w, http.StatusCreated, emp, "success")
}

// RequestVacation make request to vacate employee.
func (s Server) RequestVacation(w http.ResponseWriter, r *http.Request, id int) {
	if err := s.checkAuthUser(r); err != nil {
		s.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	var req entity.VacationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusBadRequest, "Invalid request body", "error")
		return
	}

	if req.Days <= 0 {
		s.httpResponse(w, http.StatusBadRequest, "Days must be positive", "error")
		return
	}

	now := time.Now()
	query := `UPDATE employees 
              SET vacation_days = vacation_days + $1, updated_at = $2 
              WHERE id = $3 
              RETURNING *`

	rows, err := s.DB.Query(context.Background(), query, req.Days, now, id)
	if err != nil {
		s.Logger.Error("Error updating vacation days", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to update vacation days", "error")
		return
	}
	defer rows.Close()

	var updatedEmp Employee
	updatedEmp, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[Employee])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.Logger.Error("Employee not found", slog.String("error", err.Error()))
			s.httpResponse(w, http.StatusNotFound, "Employee not found", "error")
			return
		}

		s.Logger.Error("Error collecting row", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to scan row", "error")
		return
	}

	updatedEmp.Password = nil

	s.httpResponse(w, http.StatusOK, updatedEmp, "success")
}

// GetPasswordHash is method to get password for update employee.
func (s Server) getPasswordHash(newPassword *string, employeeID int) (*string, error) {
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
	err := s.DB.QueryRow(context.Background(), query, employeeID).Scan(&currentHash)
	return &currentHash, err
}

// UpdateEmployee is method to update employee.
func (s Server) UpdateEmployee(w http.ResponseWriter, r *http.Request, id int) {
	if err := s.checkAuthUser(r); err != nil {
		s.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	var emp Employee
	if err := json.NewDecoder(r.Body).Decode(&emp); err != nil {
		s.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusBadRequest, "Invalid request body", "error")
		return
	}

	if emp.FirstName == "" || emp.LastName == "" || emp.Email == nil || *emp.Email == "" {
		s.httpResponse(w, http.StatusBadRequest, "Required fields: first_name, last_name, email", "error")
		return
	}

	passwordHash, getPassErr := s.getPasswordHash(emp.Password, id)
	if getPassErr != nil {
		if errors.Is(getPassErr, pgx.ErrNoRows) {
			s.Logger.Error("Employee not found", slog.String("error", getPassErr.Error()))
			s.httpResponse(w, http.StatusNotFound, "Employee not found", "error")
			return
		}

		s.Logger.Error("Error getting password", slog.String("error", getPassErr.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to get password", "error")
		return
	}

	emp.Password = passwordHash

	query := `SELECT COUNT(*) FROM employees WHERE (email = $1 OR (personal_number IS NOT NULL AND personal_number = $2)) AND id != $3`
	var exists int
	queryErr := s.DB.QueryRow(context.Background(), query, emp.Email, emp.PersonalNumber, id).Scan(&exists)
	if queryErr != nil {
		s.Logger.Error("Error checking uniqueness", slog.String("error", queryErr.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to check uniqueness", "error")
		return
	}

	if exists > 0 {
		s.Logger.Warn("Email or personal number already exists", slog.String("email", *emp.Email))
		s.httpResponse(w, http.StatusBadRequest, "Email or personal number already exists", "error")
		return
	}

	updatedEmp, err := s.updateEmployeeInDB(w, &emp, id)
	if err != nil {
		s.Logger.Error("Error updating employee", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to update employee", "error")
		return
	}

	s.httpResponse(w, http.StatusOK, updatedEmp, "success")
}

func (s Server) updateEmployeeInDB(w http.ResponseWriter, emp *Employee, id int) (entity.Employee, error) {
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

	rows, err := s.DB.Query(context.Background(), query,
		emp.FirstName, emp.LastName, emp.MiddleName, emp.Phone, emp.PersonalNumber,
		*emp.Email, emp.Password, emp.Role, emp.IsActive, emp.DepartmentId,
		emp.Position, emp.ManagerId, emp.HireDate, emp.FireDate, emp.Birthday,
		emp.Address, emp.VacationDays, emp.SickDays, emp.Status, emp.UpdatedAt, id)
	if err != nil {
		s.Logger.Error("Error updating employee", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to update employee", "error")
		return entity.Employee{}, fmt.Errorf("failed to update employee: %w", err)
	}
	defer rows.Close()

	updatedEmp, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.Employee])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.Logger.Error("Employee not found", slog.String("error", err.Error()))
			s.httpResponse(w, http.StatusNotFound, "Employee not found", "error")
			return entity.Employee{}, errors.New("employee not found")
		}

		s.Logger.Error("Error collecting row", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to scan row", "error")
		return entity.Employee{}, fmt.Errorf("failed to scan row: %w", err)
	}

	updatedEmp.PasswordHash = ""

	return updatedEmp, nil
}

// DeleteEmployee implements ServerInterface.
func (s Server) DeleteEmployee(w http.ResponseWriter, r *http.Request, id int) {
	if err := s.checkAuthUser(r); err != nil {
		s.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	result, err := s.DB.Exec(context.Background(), "DELETE FROM employees WHERE id = $1", id)
	if err != nil {
		s.Logger.Error("Error deleting employee", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to delete employee", "error")
		return
	}

	if result.RowsAffected() == 0 {
		s.Logger.Warn("Employee not found", slog.Int("id", id))
		s.httpResponse(w, http.StatusNotFound, "Employee not found", "error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s Server) checkAuthUser(r *http.Request) error {
	user, err := s.getUserFromToken(r)
	if err != nil {
		s.Logger.Warn("Unauthorized vacation request attempt", slog.String("error", err.Error()))
		return errors.New("Error getting user from token")
	}

	if roleErr := s.requireAdminOrHR(user); roleErr != nil {
		s.Logger.Error("Error checking role", slog.String("error", roleErr.Error()))
		return errors.New("insufficient permissions")
	}

	return nil
}

func (s Server) httpResponse(w http.ResponseWriter, status int, data any, respType string) {
	resp := map[string]any{
		"status": status,
		"type":   respType,
		"data":   data,
	}

	respData, marshalErr := json.Marshal(resp)
	if marshalErr != nil {
		s.Logger.Error("Error marshaling response", slog.String("error", marshalErr.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err := w.Write(respData); err != nil {
		s.Logger.Error("Error writing response", slog.String("error", err.Error()))
	}
}
