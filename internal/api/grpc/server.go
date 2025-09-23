package api

import (
	"context"
	"log/slog"

	pb "github.com/adamanr/employes_service/internal/api/grpc/proto"
	"github.com/adamanr/employes_service/internal/controllers"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	pb.UnimplementedEmployeeServiceServer

	deps        *controllers.Dependens
	Controllers *controllers.Controllers
}

// NewServer create new server.
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

var _ pb.EmployeeServiceServer = &Server{}

// AuthLogin authenticates a user and returns a JWT token.
func (s *Server) AuthLogin(ctx context.Context, req *pb.LoginRequest) (*pb.ApiResponse, error) {
	reqEntity := ProtoToLoginRequest(req)

	accessToken, refreshToken, err := s.Controllers.AuthController.AuthLogin(reqEntity)
	if err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error logging in", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: ErrorStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	loginResponse := &pb.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	return s.grpcResponse(ctx, loginResponse)
}

// AuthLogout make logout user.
func (s *Server) AuthLogout(ctx context.Context, _ *emptypb.Empty) (*pb.ApiResponse, error) {
	if err := s.checkAuthUser(ctx); err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error checking user authorization", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: UnauthorizedStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	authHeader, err := s.GetUserInfoFromMetadata(ctx)
	if err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error getting user info from metadata", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: ErrorStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	redisCtx := context.Background()
	if err = s.deps.Redis.Del(redisCtx, "access_token:"+authHeader).Err(); err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error deleting access token from Redis", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: ErrorStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	if err = s.deps.Redis.Del(redisCtx, "refresh_token:*").Err(); err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error deleting refresh tokens from Redis", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: ErrorStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	return s.grpcResponse(ctx, nil)
}

// CreateDepartment create new department.
func (s *Server) CreateDepartment(ctx context.Context, req *pb.DepartmentForm) (*pb.ApiResponse, error) {
	if err := s.checkAuthUser(ctx); err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error checking user authorization", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: UnauthorizedStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	dept := ProtoToDepartment(req)

	department, err := s.Controllers.DepartmentController.CreateDepartment(*dept)
	if err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error creating department", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: ErrorStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	resp := DepartmentToProto(department)

	return s.grpcResponse(ctx, resp)
}

// CreateEmployee create new employee.
func (s *Server) CreateEmployee(ctx context.Context, req *pb.Employee) (*pb.ApiResponse, error) {
	if err := s.checkAuthUser(ctx); err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error checking user authorization", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: UnauthorizedStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	emp := ProtoToEmployee(req)

	employee, err := s.Controllers.EmployeeController.CreateEmployee(*emp)
	if err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error creating employee", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: ErrorStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	pbEmployee := EmployeeToProto(employee)

	return s.grpcResponse(ctx, pbEmployee)
}

// DeleteDepartment delete department.
func (s *Server) DeleteDepartment(ctx context.Context, req *pb.DeleteDepartmentRequest) (*pb.ApiResponse, error) {
	if err := s.checkAuthUser(ctx); err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error checking user authorization", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: UnauthorizedStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	if err := s.Controllers.DepartmentController.DeleteDepartment(req.GetId()); err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error deleting department", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: ErrorStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	return s.grpcResponse(ctx, nil)
}

// DeleteEmployee delete employee.
func (s *Server) DeleteEmployee(ctx context.Context, req *pb.DeleteEmployeeRequest) (*pb.ApiResponse, error) {
	if err := s.checkAuthUser(ctx); err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error checking user authorization", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: UnauthorizedStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	if err := s.Controllers.EmployeeController.DeleteEmployee(req.GetId()); err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error deleting employee", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: ErrorStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	return s.grpcResponse(ctx, nil)
}

// GetDepartmentByID get department by id.
func (s *Server) GetDepartmentByID(ctx context.Context, req *pb.GetDepartmentByIDRequest) (*pb.ApiResponse, error) {
	if err := s.checkAuthUser(ctx); err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error checking user authorization", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: UnauthorizedStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	department, err := s.Controllers.DepartmentController.GetDepartmentByID(req.GetId())
	if err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error getting department", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: ErrorStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	pbDepartment := DepartmentToProto(department)

	return s.grpcResponse(ctx, pbDepartment)
}

// GetDepartments get all departments.
func (s *Server) GetDepartments(ctx context.Context, _ *emptypb.Empty) (*pb.ApiResponse, error) {
	if err := s.checkAuthUser(ctx); err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error checking user authorization", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: UnauthorizedStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	department, err := s.Controllers.DepartmentController.GetDepartments()
	if err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error getting department", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: ErrorStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	pbEmployee := DepartmentsToProto(department)

	resp := &pb.GetDepartmentsResponse{
		Departments: pbEmployee,
	}

	return s.grpcResponse(ctx, resp)
}

// GetEmployees get all employees.
func (s *Server) GetEmployees(ctx context.Context, _ *pb.GetEmployeesRequest) (*pb.ApiResponse, error) {
	if err := s.checkAuthUser(ctx); err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error checking user authorization", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: UnauthorizedStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	employees, err := s.Controllers.EmployeeController.GetEmployees(nil)
	if err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error getting department", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: ErrorStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	pbEmployee := EmployeesToProto(employees)

	resp := &pb.GetEmployeesResponse{
		Employees: pbEmployee,
	}

	return s.grpcResponse(ctx, resp)
}

// GetEmployeesByID get employee by id.
func (s *Server) GetEmployeesByID(ctx context.Context, req *pb.GetEmployeeByIDRequest) (*pb.ApiResponse, error) {
	if err := s.checkAuthUser(ctx); err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error checking user authorization", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: UnauthorizedStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	employee, err := s.Controllers.EmployeeController.GetEmployeeByID(req.GetId())
	if err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error getting department", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: ErrorStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	pbEmployee := EmployeeToProto(employee)

	return s.grpcResponse(ctx, pbEmployee)
}

// RequestVacation make request to vacation.
func (s *Server) RequestVacation(ctx context.Context, req *pb.RequestVacationRequest) (*pb.ApiResponse, error) {
	if err := s.checkAuthUser(ctx); err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error checking user authorization", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: UnauthorizedStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	vacation, err := s.Controllers.EmployeeController.RequestVacation(req.GetId(), req.GetVacation().GetDays())
	if err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error updating employee", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: ErrorStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	pbEmployee := EmployeeToProto(vacation)

	return s.grpcResponse(ctx, pbEmployee)
}

// UpdateDepartment update department.
func (s *Server) UpdateDepartment(ctx context.Context, req *pb.UpdateDepartmentRequest) (*pb.ApiResponse, error) {
	if err := s.checkAuthUser(ctx); err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error checking user authorization", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: UnauthorizedStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	enDep := ProtoToDepartment(req.GetDepartment())

	department, err := s.Controllers.DepartmentController.UpdateDepartment(*enDep, req.GetId())
	if err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error updating department", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: ErrorStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	pbDepartment := DepartmentToProto(department)

	return s.grpcResponse(ctx, pbDepartment)
}

// UpdateEmployee update employee.
func (s *Server) UpdateEmployee(ctx context.Context, req *pb.UpdateEmployeeRequest) (*pb.ApiResponse, error) {
	if err := s.checkAuthUser(ctx); err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error checking user authorization", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: UnauthorizedStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	emEntity := ProtoToEmployee(req.GetEmployee())

	updateEmp, updErr := s.Controllers.EmployeeController.UpdateEmployee(req.GetId(), *emEntity)
	if updErr != nil {
		s.deps.Logger.ErrorContext(ctx, "Error updating employee", slog.String("error", updErr.Error()))
		return &pb.ApiResponse{
			Status: ErrorStatus,
			Type:   "error",
			Data:   nil,
		}, updErr
	}

	pbEmployee := EmployeeToProto(updateEmp)

	return s.grpcResponse(ctx, pbEmployee)
}
