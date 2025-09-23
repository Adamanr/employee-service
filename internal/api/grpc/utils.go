package api

import (
	"context"
	"errors"
	"log/slog"

	pb "github.com/adamanr/employes_service/internal/api/grpc/proto"
	"github.com/adamanr/employes_service/internal/entity"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	SuccessStatus      = 200
	ErrorStatus        = 500
	UnauthorizedStatus = 401
)

// ProtoToLoginRequest convert proto message LoginRequest to entity LoginRequest.
func ProtoToLoginRequest(req *pb.LoginRequest) *entity.LoginRequest {
	return &entity.LoginRequest{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	}
}

// LoginRequestToProto convert entity LoginRequest to proto message LoginRequest.
func LoginRequestToProto(req *entity.LoginRequest) *pb.LoginRequest {
	return &pb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}
}

// VacationToProto convert entity VacationRequest to proto message VacationRequest.
func VacationToProto(req *entity.VacationRequest) *pb.VacationRequest {
	return &pb.VacationRequest{
		Days: req.Days,
	}
}

// ProtoToVacation convert proto message VacationRequest to entity VacationRequest.
func ProtoToVacation(req *pb.VacationRequest) *entity.VacationRequest {
	return &entity.VacationRequest{
		Days: req.GetDays(),
	}
}

// EmployeesToProto convert entity Employee to proto message Employee.
func EmployeesToProto(employees []entity.Employee) []*pb.Employee {
	pbEmployees := make([]*pb.Employee, 0, len(employees))
	for _, v := range employees {
		pbEmployees = append(pbEmployees, EmployeeToProto(&v))
	}

	return pbEmployees
}

// ProtoToEmployee convert proto message to entity Employee.
func ProtoToEmployee(proto *pb.Employee) *entity.Employee {
	if proto == nil {
		return nil
	}

	getActivity := proto.GetIsActive()
	employee := &entity.Employee{
		FirstName:    proto.GetFirstName(),
		LastName:     proto.GetLastName(),
		Role:         proto.GetRole(),
		Status:       proto.GetStatus(),
		IsActive:     &(getActivity),
		DepartmentID: nil,
		ManagerID:    nil,
	}

	if proto.GetId() != 0 {
		id := proto.GetId()
		employee.ID = &id
	}

	if proto.GetDepartmentId() != 0 {
		deptID := proto.GetDepartmentId()
		employee.DepartmentID = &deptID
	}

	if proto.GetManagerId() != 0 {
		managerID := proto.GetManagerId()
		employee.ManagerID = &managerID
	}

	if proto.GetVacationDays() > 0 {
		vacationDays := proto.GetVacationDays()
		employee.VacationDays = &vacationDays
	}

	if proto.GetSickDays() > 0 {
		sickDays := proto.GetSickDays()
		employee.SickDays = &sickDays
	}

	employee.MiddleName = proto.MiddleName
	employee.Phone = proto.Phone
	employee.PersonalNumber = proto.PersonalNumber
	employee.Email = proto.Email
	employee.Password = proto.Password
	employee.Position = proto.Position
	employee.Address = proto.Address

	if proto.GetHireDate() != nil {
		hireDate := proto.GetHireDate().AsTime()
		employee.HireDate = &hireDate
	}

	if proto.GetFireDate() != nil {
		fireDate := proto.GetFireDate().AsTime()
		employee.FireDate = &fireDate
	}

	if proto.GetBirthday() != nil {
		birthday := proto.GetBirthday().AsTime()
		employee.Birthday = &birthday
	}

	if proto.GetCreatedAt() != nil {
		createdAt := proto.GetCreatedAt().AsTime()
		employee.CreatedAt = &createdAt
	}

	if proto.GetUpdatedAt() != nil {
		updatedAt := proto.GetUpdatedAt().AsTime()
		employee.UpdatedAt = &updatedAt
	}

	return employee
}

// EmployeeToProto covert entity Employee to proto message Employee.
func EmployeeToProto(employee *entity.Employee) *pb.Employee {
	if employee == nil {
		return nil
	}

	proto := &pb.Employee{
		FirstName: employee.FirstName,
		LastName:  employee.LastName,
		Role:      employee.Role,
		Status:    employee.Status,
	}

	if employee.ID != nil {
		proto.Id = *employee.ID
	}

	if employee.IsActive != nil {
		proto.IsActive = *employee.IsActive
	}

	if employee.DepartmentID != nil {
		proto.DepartmentId = employee.DepartmentID
	}

	if employee.ManagerID != nil {
		proto.ManagerId = employee.ManagerID
	}

	if employee.VacationDays != nil {
		proto.VacationDays = *employee.VacationDays
	}

	if employee.SickDays != nil {
		proto.SickDays = *employee.SickDays
	}

	proto.MiddleName = employee.MiddleName
	proto.Phone = employee.Phone
	proto.PersonalNumber = employee.PersonalNumber
	proto.Email = employee.Email
	proto.Password = employee.Password
	proto.Position = employee.Position
	proto.Address = employee.Address

	if employee.HireDate != nil {
		proto.HireDate = timestamppb.New(*employee.HireDate)
	}

	if employee.FireDate != nil {
		proto.FireDate = timestamppb.New(*employee.FireDate)
	}

	if employee.Birthday != nil {
		proto.Birthday = timestamppb.New(*employee.Birthday)
	}

	if employee.CreatedAt != nil {
		proto.CreatedAt = timestamppb.New(*employee.CreatedAt)
	}

	if employee.UpdatedAt != nil {
		proto.UpdatedAt = timestamppb.New(*employee.UpdatedAt)
	}

	return proto
}

// ProtoToDepartment covert proto message DepartmentForm to entity Department.
func ProtoToDepartment(proto *pb.DepartmentForm) *entity.Department {
	if proto == nil {
		return nil
	}

	department := &entity.Department{
		Name: proto.GetName(),
	}

	if proto.Description != nil {
		department.Description = proto.GetDescription()
	}

	if proto.ParentId != nil {
		parentID := proto.GetParentId()
		department.ParentID = &parentID
	}

	if proto.HeadId != nil {
		headID := proto.GetHeadId()
		department.HeadID = &headID
	}

	return department
}

// DepartmentsToProto convert entity list Department to proto list message Department.
func DepartmentsToProto(departments []entity.Department) []*pb.Department {
	pbDepartments := make([]*pb.Department, 0, len(departments))
	for _, v := range departments {
		pbDepartments = append(pbDepartments, DepartmentToProto(&v))
	}

	return pbDepartments
}

// DepartmentToProto covert entity Department to proto message Department.
func DepartmentToProto(department *entity.Department) *pb.Department {
	if department == nil {
		return nil
	}

	proto := &pb.Department{
		Id:   department.ID,
		Name: department.Name,
	}

	if department.Description != "" {
		proto.Description = &department.Description
	}

	if department.ParentID != nil {
		proto.ParentId = department.ParentID
	}

	if department.HeadID != nil {
		proto.HeadId = department.HeadID
	}

	if !department.CreatedAt.IsZero() {
		proto.CreatedAt = timestamppb.New(department.CreatedAt)
	}

	if !department.UpdatedAt.IsZero() {
		proto.UpdatedAt = timestamppb.New(department.UpdatedAt)
	}

	return proto
}

// GetUserInfoFromMetadata get authorization from metadata.
func (s *Server) GetUserInfoFromMetadata(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		s.deps.Logger.ErrorContext(ctx, "Error getting metadata", slog.Any("ctx", ctx))
		return "", errors.New("missing metadata")
	}

	s.deps.Logger.InfoContext(ctx, "Get User", slog.Any("Context", ctx))

	userAgents := md.Get("user-agent")
	xForwardedFor := md.Get("x-forwarded-for")
	authorization := md.Get("authorization")

	s.deps.Logger.InfoContext(ctx, "Request metadata",
		slog.Any("user-agent", userAgents),
		slog.Any("x-forwarded-for", xForwardedFor),
		slog.Any("authorization", authorization),
	)

	if len(authorization) > 0 {
		s.deps.Logger.InfoContext(ctx, "Authorization", slog.Any("authorization", authorization))
		return authorization[0], nil
	}

	return "", errors.New("missing authorization")
}

// checkAuthUser check authorization.
func (s *Server) checkAuthUser(ctx context.Context) error {
	token, err := s.GetUserInfoFromMetadata(ctx)
	if err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error getting user info from metadata", slog.String("error", err.Error()))
		return err
	}

	if _, err = s.Controllers.AuthController.CheckUserToken(token); err != nil {
		s.deps.Logger.ErrorContext(ctx, "Unauthorized vacation request attempt", slog.String("error", err.Error()))
		return errors.New("error getting user from token")
	}

	return nil
}

// grpcResponse return ApiResponse struct with data and error.
func (s *Server) grpcResponse(ctx context.Context, msg protoreflect.ProtoMessage) (*pb.ApiResponse, error) {
	data, err := anypb.New(msg)
	if err != nil {
		s.deps.Logger.ErrorContext(ctx, "Error convert to anypb", slog.String("error", err.Error()))
		return &pb.ApiResponse{
			Status: ErrorStatus,
			Type:   "error",
			Data:   nil,
		}, err
	}

	return &pb.ApiResponse{
		Status: SuccessStatus,
		Type:   "success",
		Data:   data,
	}, nil
}
