package api

//go:generate go tool oapi-codegen -config ../../../configs/cfg.yaml ../../../cmd/api.swagger.yaml

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/adamanr/employes_service/internal/controllers"
	"github.com/adamanr/employes_service/internal/entity"
)

type Server struct {
	deps        *controllers.Dependens
	Controllers *controllers.Controllers
}

func NewServer(deps *controllers.Dependens) *Server {
	return &Server{
		deps: deps,
		Controllers: &controllers.Controllers{
			AuthController:       controllers.NewAuthController(deps),
			DepartmentController: controllers.NewDepartmentController(deps),
			EmployeeController:   controllers.NewEmployeeController(deps),
		},
	}
}

var _ ServerInterface = Server{}

// getUserFromToken extracts user information from the token.
func (s Server) getUserFromToken(r *http.Request) (*entity.Claims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		s.deps.Logger.Error("Authorization header missing")
		return nil, errors.New("authorization header missing")
	}

	claims, err := s.Controllers.AuthController.CheckUserToken(authHeader)
	if err != nil {
		s.deps.Logger.Error("Error checking token", slog.String("error", err.Error()))
		return nil, err
	}

	return claims, nil
}

// func (s Server) requireAdminOrHR(user *entity.Claims) error {
// 	if user.Role != "admin" && user.Role != "hr" {
// 		return errors.New("insufficient permissions")
// 	}

// 	return nil
// }

// AuthLogin authenticates a user and returns a JWT token.
func (s Server) AuthLogin(w http.ResponseWriter, r *http.Request) {
	var req entity.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.deps.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"}, "error")
		return
	}

	accessToken, refreshToken, err := s.Controllers.AuthController.AuthLogin(&req)
	if err != nil {
		s.deps.Logger.Error("Error logging in", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, map[string]string{"error": err.Error()}, "error")
		return
	}

	s.httpResponse(w, http.StatusOK, entity.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, "success")
}

// AuthLogout make logout user.
func (s Server) AuthLogout(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

	ctx := context.Background()
	if err := s.deps.Redis.Del(ctx, "access_token:"+tokenStr).Err(); err != nil {
		s.deps.Logger.Error("Error deleting access token from Redis", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to logout", "error")
		return
	}

	if err := s.deps.Redis.Del(ctx, "refresh_token:*").Err(); err != nil {
		s.deps.Logger.Error("Error deleting refresh tokens from Redis", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to logout", "error")
		return
	}

	s.httpResponse(w, http.StatusOK, map[string]string{"message": "Logged out successfully"}, "success")
}

// GetDepartments get all departments.
func (s Server) GetDepartments(w http.ResponseWriter, r *http.Request) {
	if err := s.checkAuthUser(r); err != nil {
		s.deps.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	departments, err := s.Controllers.DepartmentController.GetDepartments()
	if err != nil {
		s.deps.Logger.Error("Error getting departments", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to get departments", "error")
		return
	}

	s.httpResponse(w, http.StatusOK, departments, "success")
}

// GetDepartmentByID get department by id.
func (s Server) GetDepartmentByID(w http.ResponseWriter, r *http.Request, id uint64) {
	if err := s.checkAuthUser(r); err != nil {
		s.deps.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	department, err := s.Controllers.DepartmentController.GetDepartmentByID(id)
	if err != nil {
		s.deps.Logger.Error("Error getting department", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to get department", "error")
		return
	}

	s.httpResponse(w, http.StatusOK, department, "success")
}

// CreateDepartment create new department.
//
//nolint:dupl // This is not duplicate!!
func (s Server) CreateDepartment(w http.ResponseWriter, r *http.Request) {
	if err := s.checkAuthUser(r); err != nil {
		s.deps.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	var dept entity.Department
	if err := json.NewDecoder(r.Body).Decode(&dept); err != nil {
		s.deps.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusBadRequest, "Invalid request body", "error")
		return
	}

	department, err := s.Controllers.DepartmentController.CreateDepartment(dept)
	if err != nil {
		s.deps.Logger.Error("Error creating department", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to create department", "error")
		return
	}

	s.httpResponse(w, http.StatusCreated, department, "success")
}

// UpdateDepartment update department by id.
//
//nolint:dupl // This is not duplicate!!
func (s Server) UpdateDepartment(w http.ResponseWriter, r *http.Request, id uint64) {
	if err := s.checkAuthUser(r); err != nil {
		s.deps.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	var dept entity.Department
	if err := json.NewDecoder(r.Body).Decode(&dept); err != nil {
		s.deps.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusBadRequest, "Invalid request body", "error")
		return
	}

	department, err := s.Controllers.DepartmentController.UpdateDepartment(dept, id)
	if err != nil {
		s.deps.Logger.Error("Error updating department", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to update department", "error")
		return
	}

	s.httpResponse(w, http.StatusOK, department, "success")
}

// DeleteDepartment delete department.
func (s Server) DeleteDepartment(w http.ResponseWriter, r *http.Request, id uint64) {
	if err := s.checkAuthUser(r); err != nil {
		s.deps.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	if err := s.Controllers.DepartmentController.DeleteDepartment(id); err != nil {
		s.deps.Logger.Error("Error deleting department", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to delete department", "error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetEmployees get all employees.
func (s Server) GetEmployees(w http.ResponseWriter, r *http.Request, params GetEmployeesParams) {
	if err := s.checkAuthUser(r); err != nil {
		s.deps.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	entityParams := entity.GetEmployeesParams(params)

	employees, err := s.Controllers.EmployeeController.GetEmployees(&entityParams)
	if err != nil {
		s.deps.Logger.Error("Error getting employees", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to get employees", "error")
		return
	}

	s.httpResponse(w, http.StatusOK, employees, "success")
}

// GetEmployeesByID get employee by id.
func (s Server) GetEmployeesByID(w http.ResponseWriter, r *http.Request, id uint64) {
	if err := s.checkAuthUser(r); err != nil {
		s.deps.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	employee, err := s.Controllers.EmployeeController.GetEmployeeByID(id)
	if err != nil {
		s.deps.Logger.Error("Error getting employee", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to get employee", "error")
		return
	}

	s.httpResponse(w, http.StatusOK, employee, "success")
}

// CreateEmployee create new employee.
//
//nolint:dupl // This is not duplicate!!
func (s Server) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	if err := s.checkAuthUser(r); err != nil {
		s.deps.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	var emp entity.Employee
	if err := json.NewDecoder(r.Body).Decode(&emp); err != nil {
		s.deps.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusBadRequest, "Invalid request body", "error")
		return
	}

	employee, err := s.Controllers.EmployeeController.CreateEmployee(emp)
	if err != nil {
		s.deps.Logger.Error("Error creating employee", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to create employee", "error")
		return
	}

	s.httpResponse(w, http.StatusCreated, employee, "success")
}

// RequestVacation make request to vacate employee.
func (s Server) RequestVacation(w http.ResponseWriter, r *http.Request, id uint64) {
	if err := s.checkAuthUser(r); err != nil {
		s.deps.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	var req entity.VacationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.deps.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusBadRequest, "Invalid request body", "error")
		return
	}

	updateEmp, err := s.Controllers.EmployeeController.RequestVacation(id, req.Days)
	if err != nil {
		s.deps.Logger.Error("Error updating employee", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to update employee", "error")
		return
	}

	s.httpResponse(w, http.StatusOK, updateEmp, "success")
}

// UpdateEmployee is method to update employee.
//
//nolint:dupl // This is not duplicate!!
func (s Server) UpdateEmployee(w http.ResponseWriter, r *http.Request, id uint64) {
	if err := s.checkAuthUser(r); err != nil {
		s.deps.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	var emp entity.Employee
	if err := json.NewDecoder(r.Body).Decode(&emp); err != nil {
		s.deps.Logger.Error("Error decoding request body", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusBadRequest, "Invalid request body", "error")
		return
	}

	updateEmp, err := s.Controllers.EmployeeController.UpdateEmployee(id, emp)
	if err != nil {
		s.deps.Logger.Error("Error updating employee", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to update employee", "error")
		return
	}

	s.httpResponse(w, http.StatusOK, updateEmp, "success")
}

// DeleteEmployee implements ServerInterface.
func (s Server) DeleteEmployee(w http.ResponseWriter, r *http.Request, id uint64) {
	if err := s.checkAuthUser(r); err != nil {
		s.deps.Logger.Error("Error checking auth", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusUnauthorized, "Unauthorized", "error")
		return
	}

	if err := s.Controllers.EmployeeController.DeleteEmployee(id); err != nil {
		s.deps.Logger.Error("Error deleting employee", slog.String("error", err.Error()))
		s.httpResponse(w, http.StatusInternalServerError, "Failed to delete employee", "error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s Server) checkAuthUser(r *http.Request) error {
	if _, err := s.getUserFromToken(r); err != nil {
		s.deps.Logger.Warn("Unauthorized vacation request attempt", slog.String("error", err.Error()))
		return errors.New("Error getting user from token")
	}

	// if roleErr := s.requireAdminOrHR(user); roleErr != nil {
	// 	s.deps.Logger.Error("Error checking role", slog.String("error", roleErr.Error()))
	// 	return errors.New("insufficient permissions")
	// }

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
		s.deps.Logger.Error("Error marshaling response", slog.String("error", marshalErr.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err := w.Write(respData); err != nil {
		s.deps.Logger.Error("Error writing response", slog.String("error", err.Error()))
	}
}
