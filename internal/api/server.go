package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
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
	ID      uint   `json:"id"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	TokenID string `json:"token_id"`
	jwt.RegisteredClaims
}

func generateTokenID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s Server) getUserFromToken(r *http.Request) (*Claims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("authorization header missing")
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenStr == authHeader {
		return nil, fmt.Errorf("invalid bearer token")
	}

	ctx := context.Background()
	if s.Redis.Get(ctx, "access_token:"+tokenStr).Err() == redis.Nil {
		return nil, fmt.Errorf("token revoked or not found")
	}

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.Cfg.Server.JWTSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid claims")
}

func (s Server) requireAdminOrHR(user *Claims) error {
	if user.Role != "admin" && user.Role != "hr" {
		return fmt.Errorf("insufficient permissions")
	}
	return nil
}

// PostAuthLogin implements ServerInterface.
func (s Server) PostAuthLogin(w http.ResponseWriter, r *http.Request) {
	var req entity.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"}, "error")
		return
	}

	if req.Email == "" || req.Password == "" {
		httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Email and password required"}, "error")
		return
	}

	var emp entity.Employee
	err := s.DB.QueryRow(context.Background(), "SELECT id, email, password, role FROM employees WHERE email = $1", req.Email).Scan(
		&emp.ID, &emp.Email, &emp.PasswordHash, &emp.Role,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			s.Logger.Warn("Invalid login attempt", slog.String("email", req.Email))
			httpResponse(w, http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"}, "error")
			return
		}
		s.Logger.Error("Error querying employee", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to authenticate"}, "error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(emp.PasswordHash), []byte(req.Password)); err != nil {
		s.Logger.Warn("Invalid password", slog.String("email", req.Email))
		httpResponse(w, http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"}, "error")
		return
	}

	tokenID, err := generateTokenID()
	if err != nil {
		s.Logger.Error("Error generating token ID", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate token"}, "error")
		return
	}

	accessClaims := Claims{
		ID:      emp.ID,
		Email:   emp.Email,
		Role:    emp.Role,
		TokenID: tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.Cfg.Redis.AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenStr, err := accessToken.SignedString([]byte(s.Cfg.Server.JWTSecret))
	if err != nil {
		s.Logger.Error("Error signing access token", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate access token"}, "error")
		return
	}

	refreshTokenID, err := generateTokenID()
	if err != nil {
		s.Logger.Error("Error generating refresh token ID", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate refresh token"}, "error")
		return
	}
	refreshClaims := Claims{
		ID:      emp.ID,
		Email:   emp.Email,
		Role:    emp.Role,
		TokenID: refreshTokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.Cfg.Redis.RefreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenStr, err := refreshToken.SignedString([]byte(s.Cfg.Server.JWTSecret))
	if err != nil {
		s.Logger.Error("Error signing refresh token", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate refresh token"}, "error")
		return
	}

	ctx := context.Background()
	if err = s.Redis.Set(ctx, "access_token:"+accessTokenStr, emp.ID, s.Cfg.Redis.AccessTokenTTL).Err(); err != nil {
		s.Logger.Error("Error saving access token to Redis", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save access token"}, "error")
		return
	}

	if err = s.Redis.Set(ctx, "refresh_token:"+refreshTokenStr, emp.ID, s.Cfg.Redis.RefreshTokenTTL).Err(); err != nil {
		s.Logger.Error("Error saving refresh token to Redis", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save refresh token"}, "error")
		return
	}

	httpResponse(w, http.StatusOK, entity.LoginResponse{
		AccessToken:  accessTokenStr,
		RefreshToken: refreshTokenStr,
	}, "success")
}

// PostAuthLogout implements ServerInterface.
func (s Server) PostAuthLogout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	ctx := context.Background()
	err := s.Redis.Del(ctx, "access_token:"+tokenStr).Err()
	if err != nil {
		s.Logger.Error("Error deleting access token from Redis", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to logout"}, "error")
		return
	}

	err = s.Redis.Del(ctx, "refresh_token:*").Err()
	if err != nil {
		s.Logger.Error("Error deleting refresh tokens from Redis", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to logout"}, "error")
		return
	}

	httpResponse(w, http.StatusOK, map[string]string{"message": "Logged out successfully"}, "success")
}

// PostAuthRegister implements ServerInterface.
func (s Server) PostAuthRegister(w http.ResponseWriter, r *http.Request) {
	var emp Employee
	if err := json.NewDecoder(r.Body).Decode(&emp); err != nil {
		s.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"}, "error")
		return
	}

	if emp.FirstName == "" || emp.LastName == "" || emp.Email == nil || *emp.Email == "" || *emp.Password == "" {
		httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Required fields: first_name, last_name, email, password"}, "error")
		return
	}

	var exists int
	err := s.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM employees WHERE email = $1 OR (personal_number IS NOT NULL AND personal_number = $2)", emp.Email, emp.PersonalNumber).Scan(&exists)
	if err != nil {
		s.Logger.Error("Error checking uniqueness", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to check uniqueness"}, "error")
		return
	}
	if exists > 0 {
		httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Email or personal number already exists"}, "error")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(*emp.Password), bcrypt.DefaultCost)
	if err != nil {
		s.Logger.Error("Error hashing password", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to hash password"}, "error")
		return
	}

	hashPassword := string(passwordHash)
	emp.Password = &hashPassword

	now := time.Now()
	query := `INSERT INTO employees (first_name, last_name, middle_name, phone, personal_number, email, password, role, is_active, department_id, position, manager_id, hire_date, fire_date, birthday, address, vacation_days, sick_days, status, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
              RETURNING id`

	err = s.DB.QueryRow(context.Background(), query,
		emp.FirstName, emp.LastName, emp.MiddleName, emp.Phone, emp.PersonalNumber,
		emp.Email, emp.Password, emp.Role, emp.IsActive, emp.DepartmentId, emp.Position,
		emp.ManagerId, emp.HireDate, emp.FireDate, emp.Birthday, emp.Address,
		emp.VacationDays, emp.SickDays, emp.Status, now, now,
	).Scan(&emp.Id)
	if err != nil {
		s.Logger.Error("Error inserting employee", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create employee"}, "error")
		return
	}

	emp.CreatedAt = &now
	emp.UpdatedAt = &now
	emp.Password = nil

	httpResponse(w, http.StatusCreated, emp, "success")
}

// GetDepartments implements ServerInterface.
func (s Server) GetDepartments(w http.ResponseWriter, r *http.Request) {
	_, err := s.getUserFromToken(r)
	if err != nil {
		s.Logger.Error("Error getting user from token", slog.String("error", err.Error()))
		httpResponse(w, http.StatusUnauthorized, err, "error")
		return
	}

	rows, err := s.DB.Query(context.Background(), "SELECT id, name, description, parent_id, head_id, created_at, updated_at FROM departments")
	if err != nil {
		s.Logger.Error("Error querying departments", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to query departments"}, "error")
		return
	}
	defer rows.Close()

	departments, err := pgx.CollectRows(rows, pgx.RowToStructByName[entity.Department])
	if err != nil {
		s.Logger.Error("Error collecting rows", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to scan rows"}, "error")
		return
	}

	httpResponse(w, http.StatusOK, departments, "success")
}

// GetDepartmentsId implements ServerInterface.
func (s Server) GetDepartmentsId(w http.ResponseWriter, r *http.Request, id int) {
	_, err := s.getUserFromToken(r)
	if err != nil {
		httpResponse(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"}, "error")
		return
	}

	rows, err := s.DB.Query(context.Background(), "SELECT id, name, description, parent_id, head_id, created_at, updated_at FROM departments WHERE id = $1", id)
	if err != nil {
		s.Logger.Error("Error querying department", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to query department"}, "error")
		return
	}
	defer rows.Close()

	department, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.Department])
	if err != nil {
		if err == pgx.ErrNoRows {
			httpResponse(w, http.StatusNotFound, map[string]string{"error": "Department not found"}, "error")
			return
		}
		s.Logger.Error("Error collecting row", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to scan row"}, "error")
		return
	}

	httpResponse(w, http.StatusOK, department, "success")
}

// PostDepartments implements ServerInterface.
func (s Server) PostDepartments(w http.ResponseWriter, r *http.Request) {
	user, err := s.getUserFromToken(r)
	if err != nil {
		httpResponse(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"}, "error")
		return
	}
	if err := s.requireAdminOrHR(user); err != nil {
		httpResponse(w, http.StatusForbidden, map[string]string{"error": "Insufficient permissions"}, "error")
		return
	}

	var dept entity.Department
	if err := json.NewDecoder(r.Body).Decode(&dept); err != nil {
		s.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"}, "error")
		return
	}

	if dept.Name == "" {
		httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Name is required"}, "error")
		return
	}

	now := time.Now()
	query := `INSERT INTO departments (name, description, parent_id, head_id, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6)
              RETURNING id`

	err = s.DB.QueryRow(context.Background(), query, dept.Name, dept.Description, dept.ParentID, dept.HeadID, now, now).Scan(&dept.ID)
	if err != nil {
		s.Logger.Error("Error inserting department", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create department"}, "error")
		return
	}

	dept.CreatedAt = now
	dept.UpdatedAt = now

	httpResponse(w, http.StatusCreated, dept, "success")
}

// PutDepartmentsId implements ServerInterface.
func (s Server) PutDepartmentsId(w http.ResponseWriter, r *http.Request, id int) {
	user, err := s.getUserFromToken(r)
	if err != nil {
		httpResponse(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"}, "error")
		return
	}
	if err := s.requireAdminOrHR(user); err != nil {
		httpResponse(w, http.StatusForbidden, map[string]string{"error": "Insufficient permissions"}, "error")
		return
	}

	var dept entity.Department
	if err := json.NewDecoder(r.Body).Decode(&dept); err != nil {
		s.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"}, "error")
		return
	}

	dept.UpdatedAt = time.Now()

	query := `UPDATE departments 
              SET name = $1, description = $2, parent_id = $3, head_id = $4, updated_at = $5 
              WHERE id = $6 
              RETURNING id, name, description, parent_id, head_id, created_at, updated_at`

	err = s.DB.QueryRow(context.Background(), query, dept.Name, dept.Description, dept.ParentID, dept.HeadID, dept.UpdatedAt, id).Scan(
		&dept.ID, &dept.Name, &dept.Description, &dept.ParentID, &dept.HeadID, &dept.CreatedAt, &dept.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			httpResponse(w, http.StatusNotFound, map[string]string{"error": "Department not found"}, "error")
			return
		}
		s.Logger.Error("Error updating department", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update department"}, "error")
		return
	}

	httpResponse(w, http.StatusOK, dept, "success")
}

// DeleteDepartmentsId implements ServerInterface.
func (s Server) DeleteDepartmentsId(w http.ResponseWriter, r *http.Request, id int) {
	user, err := s.getUserFromToken(r)
	if err != nil {
		httpResponse(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"}, "error")
		return
	}
	if user.Role != "admin" {
		httpResponse(w, http.StatusForbidden, map[string]string{"error": "Insufficient permissions"}, "error")
		return
	}

	result, err := s.DB.Exec(context.Background(), "DELETE FROM departments WHERE id = $1", id)
	if err != nil {
		s.Logger.Error("Error deleting department", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete department"}, "error")
		return
	}

	if result.RowsAffected() == 0 {
		httpResponse(w, http.StatusNotFound, map[string]string{"error": "Department not found"}, "error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetEmployees implements ServerInterface.
func (s Server) GetEmployees(w http.ResponseWriter, r *http.Request, params GetEmployeesParams) {
	user, err := s.getUserFromToken(r)
	if err != nil {
		httpResponse(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"}, "error")
		return
	}

	if err := s.requireAdminOrHR(user); err != nil {
		s.Logger.Warn("Insufficient permissions for get employees", slog.String("role", user.Role))
		httpResponse(w, http.StatusForbidden, map[string]string{"error": "Insufficient permissions"}, "error")
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
		argIdx++
	}

	rows, err := s.DB.Query(context.Background(), query, args...)
	if err != nil {
		s.Logger.Error("Error querying employees", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to query employees"}, "error")
		return
	}
	defer rows.Close()

	employees, err := pgx.CollectRows(rows, pgx.RowToStructByName[entity.Employee])
	if err != nil {
		s.Logger.Error("Error collecting rows", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to scan rows"}, "error")
		return
	}

	for i := range employees {
		employees[i].PasswordHash = ""
	}

	httpResponse(w, http.StatusOK, employees, "success")
}

// GetEmployeesId implements ServerInterface.
func (s Server) GetEmployeesId(w http.ResponseWriter, r *http.Request, id int) {
	user, err := s.getUserFromToken(r)
	if err != nil {
		httpResponse(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"}, "error")
		return
	}

	if user.Role != "admin" && user.Role != "hr" && user.ID != uint(id) {
		s.Logger.Warn("Insufficient permissions for get employee", slog.String("role", user.Role), slog.Int("user_id", int(user.ID)))
		httpResponse(w, http.StatusForbidden, map[string]string{"error": "Insufficient permissions"}, "error")
		return
	}

	rows, err := s.DB.Query(context.Background(), "SELECT * FROM employees WHERE id = $1", id)
	if err != nil {
		s.Logger.Error("Error querying employee", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to query employee"}, "error")
		return
	}
	defer rows.Close()

	employee, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.Employee])
	if err != nil {
		if err == pgx.ErrNoRows {
			httpResponse(w, http.StatusNotFound, map[string]string{"error": "Employee not found"}, "error")
			return
		}
		s.Logger.Error("Error collecting row", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to scan row"}, "error")
		return
	}

	employee.PasswordHash = ""

	httpResponse(w, http.StatusOK, employee, "success")
}

// PostEmployees implements ServerInterface.
func (s Server) PostEmployees(w http.ResponseWriter, r *http.Request) {
	user, err := s.getUserFromToken(r)
	if err != nil {
		httpResponse(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"}, "error")
		return
	}
	if err := s.requireAdminOrHR(user); err != nil {
		httpResponse(w, http.StatusForbidden, map[string]string{"error": "Insufficient permissions"}, "error")
		return
	}

	var emp Employee
	if err := json.NewDecoder(r.Body).Decode(&emp); err != nil {
		s.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"}, "error")
		return
	}

	if emp.FirstName == "" || emp.LastName == "" || emp.Email == nil || *emp.Email == "" {
		httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Required fields: first_name, last_name, email"}, "error")
		return
	}

	if *emp.Password == "" {
		*emp.Password = "default123"
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(*emp.Password), bcrypt.DefaultCost)
	if err != nil {
		s.Logger.Error("Error hashing password", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to hash password"}, "error")
		return
	}

	hashPassword := string(passwordHash)
	emp.Password = &hashPassword

	var exists int
	err = s.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM employees WHERE email = $1 OR (personal_number IS NOT NULL AND personal_number = $2)", emp.Email, emp.PersonalNumber).Scan(&exists)
	if err != nil {
		s.Logger.Error("Error checking uniqueness", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to check uniqueness"}, "error")
		return
	}
	if exists > 0 {
		httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Email or personal number already exists"}, "error")
		return
	}

	now := time.Now()
	query := `INSERT INTO employees (first_name, last_name, middle_name, phone, personal_number, email, password, role, is_active, department_id, position, manager_id, hire_date, fire_date, birthday, address, vacation_days, sick_days, status, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
              RETURNING id`

	err = s.DB.QueryRow(context.Background(), query,
		emp.FirstName, emp.LastName, emp.MiddleName, emp.Phone, emp.PersonalNumber,
		emp.Email, emp.Password, emp.Role, emp.IsActive, emp.DepartmentId, emp.Position,
		emp.ManagerId, emp.HireDate, emp.FireDate, emp.Birthday, emp.Address,
		emp.VacationDays, emp.SickDays, emp.Status, now, now,
	).Scan(&emp.Id)
	if err != nil {
		s.Logger.Error("Error inserting employee", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create employee"}, "error")
		return
	}

	emp.CreatedAt = &now
	emp.UpdatedAt = &now

	httpResponse(w, http.StatusCreated, emp, "success")
}

// PostEmployeesIdVacation implements ServerInterface.
func (s Server) PostEmployeesIdVacation(w http.ResponseWriter, r *http.Request, id int) {
	user, err := s.getUserFromToken(r)
	if err != nil {
		s.Logger.Warn("Unauthorized vacation request attempt", slog.String("error", err.Error()))
		httpResponse(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"}, "error")
		return
	}

	if user.Role != "admin" && user.Role != "hr" && user.ID != uint(id) {
		s.Logger.Warn("Insufficient permissions for vacation request", slog.String("role", user.Role), slog.Int("user_id", int(user.ID)))
		httpResponse(w, http.StatusForbidden, map[string]string{"error": "Insufficient permissions"}, "error")
		return
	}

	var req entity.VacationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"}, "error")
		return
	}

	if req.Days <= 0 {
		httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Days must be positive"}, "error")
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
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update vacation days"}, "error")
		return
	}
	defer rows.Close()

	var updatedEmp entity.Employee
	updatedEmp, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.Employee])
	if err != nil {
		if err == pgx.ErrNoRows {
			httpResponse(w, http.StatusNotFound, map[string]string{"error": "Employee not found"}, "error")
			return
		}
		s.Logger.Error("Error collecting row", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to scan row"}, "error")
		return
	}

	updatedEmp.PasswordHash = ""

	httpResponse(w, http.StatusOK, updatedEmp, "success")
}

// PutEmployeesId implements ServerInterface.
func (s Server) PutEmployeesId(w http.ResponseWriter, r *http.Request, id int) {
	user, err := s.getUserFromToken(r)
	if err != nil {
		if err := s.requireAdminOrHR(user); err != nil {
			httpResponse(w, http.StatusForbidden, map[string]string{"error": "Insufficient permissions"}, "error")
			return
		}

		httpResponse(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"}, "error")
		return
	}

	var emp Employee
	if err := json.NewDecoder(r.Body).Decode(&emp); err != nil {
		s.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"}, "error")
		return
	}

	if emp.FirstName == "" || emp.LastName == "" || emp.Email == nil || *emp.Email == "" {
		httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Required fields: first_name, last_name, email"}, "error")
		return
	}

	if *emp.Password != "" {
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(*emp.Password), bcrypt.DefaultCost)
		if err != nil {
			s.Logger.Error("Error hashing password", slog.String("error", err.Error()))
			httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to hash password"}, "error")
			return
		}
		hashPassword := string(passwordHash)
		emp.Password = &hashPassword
	} else {
		var currentHash string
		err := s.DB.QueryRow(context.Background(), "SELECT password FROM employees WHERE id = $1", id).Scan(&currentHash)
		if err != nil {
			if err == pgx.ErrNoRows {
				httpResponse(w, http.StatusNotFound, map[string]string{"error": "Employee not found"}, "error")
				return
			}
			s.Logger.Error("Error fetching current password hash", slog.String("error", err.Error()))
			httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch employee"}, "error")
			return
		}
		emp.Password = &currentHash
	}

	var exists int
	err = s.DB.QueryRow(context.Background(), "SELECT COUNT(*) FROM employees WHERE (email = $1 OR (personal_number IS NOT NULL AND personal_number = $2)) AND id != $3", emp.Email, emp.PersonalNumber, id).Scan(&exists)
	if err != nil {
		s.Logger.Error("Error checking uniqueness", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to check uniqueness"}, "error")
		return
	}

	if exists > 0 {
		httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Email or personal number already exists"}, "error")
		return
	}

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
		emp.Email, emp.Password, emp.Role, emp.IsActive, emp.DepartmentId,
		emp.Position, emp.ManagerId, emp.HireDate, emp.FireDate, emp.Birthday,
		emp.Address, emp.VacationDays, emp.SickDays, emp.Status, emp.UpdatedAt, id,
	)
	if err != nil {
		s.Logger.Error("Error updating employee", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update employee"}, "error")
		return
	}
	defer rows.Close()

	var updatedEmp entity.Employee
	updatedEmp, err = pgx.CollectOneRow(rows, pgx.RowToStructByName[entity.Employee])
	if err != nil {
		if err == pgx.ErrNoRows {
			httpResponse(w, http.StatusNotFound, map[string]string{"error": "Employee not found"}, "error")
			return
		}
		s.Logger.Error("Error collecting row", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to scan row"}, "error")
		return
	}

	updatedEmp.PasswordHash = ""

	httpResponse(w, http.StatusOK, updatedEmp, "success")
}

// DeleteEmployeesId implements ServerInterface.
func (s Server) DeleteEmployeesId(w http.ResponseWriter, r *http.Request, id int) {
	user, err := s.getUserFromToken(r)
	if err != nil {
		if err := s.requireAdminOrHR(user); err != nil {
			httpResponse(w, http.StatusForbidden, map[string]string{"error": "Insufficient permissions"}, "error")
			return
		}

		httpResponse(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"}, "error")
		return
	}

	result, err := s.DB.Exec(context.Background(), "DELETE FROM employees WHERE id = $1", id)
	if err != nil {
		s.Logger.Error("Error deleting employee", slog.String("error", err.Error()))
		httpResponse(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete employee"}, "error")
		return
	}

	if result.RowsAffected() == 0 {
		httpResponse(w, http.StatusNotFound, map[string]string{"error": "Employee not found"}, "error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func httpResponse(w http.ResponseWriter, status int, data any, respType string) {
	resp := map[string]any{
		"status": status,
		"type":   respType,
		"data":   data,
	}

	respData, marshalErr := json.Marshal(resp)
	if marshalErr != nil {
		slog.Error("Error marshaling response", slog.String("error", marshalErr.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err := w.Write(respData); err != nil {
		slog.Error("Error writing response", slog.String("error", err.Error()))
	}
}
