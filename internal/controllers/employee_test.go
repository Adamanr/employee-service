package controllers

import (
	"errors"
	"testing"
	"time"

	"github.com/adamanr/employes_service/internal/entity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var EmployeeFieldDescriptions = []pgconn.FieldDescription{
	{Name: "id", DataTypeOID: 20},              // int8 (uint64)
	{Name: "first_name", DataTypeOID: 25},      // text (string)
	{Name: "last_name", DataTypeOID: 25},       // text (string)
	{Name: "email", DataTypeOID: 25},           // text (string)
	{Name: "password", DataTypeOID: 25},        // text (string)
	{Name: "role", DataTypeOID: 25},            // text (string)
	{Name: "status", DataTypeOID: 25},          // text (string)
	{Name: "department_id", DataTypeOID: 20},   // int8 (uint64, nullable)
	{Name: "manager_id", DataTypeOID: 20},      // int8 (uint64, nullable)
	{Name: "position", DataTypeOID: 25},        // text (string, nullable)
	{Name: "address", DataTypeOID: 25},         // text (string, nullable)
	{Name: "phone", DataTypeOID: 25},           // text (string, nullable)
	{Name: "personal_number", DataTypeOID: 25}, // text (string, nullable)
	{Name: "middle_name", DataTypeOID: 25},     // text (string, nullable)
	{Name: "birthday", DataTypeOID: 1114},      // timestamp (nullable)
	{Name: "hire_date", DataTypeOID: 1114},     // timestamp (nullable)
	{Name: "fire_date", DataTypeOID: 1114},     // timestamp (nullable)
	{Name: "is_active", DataTypeOID: 16},       // boolean (nullable)
	{Name: "vacation_days", DataTypeOID: 20},   // int8 (uint64, nullable)
	{Name: "sick_days", DataTypeOID: 20},       // int8 (uint64, nullable)
	{Name: "created_at", DataTypeOID: 1114},    // timestamp (nullable)
	{Name: "updated_at", DataTypeOID: 1114},    // timestamp (nullable)
}

func TestEmployeeController_GetEmployees(t *testing.T) {
	tests := []struct {
		name        string
		params      *entity.GetEmployeesParams
		setupMocks  func(*MockDB)
		expectError bool
		expectedLen int
	}{
		{
			name:   "get all employees",
			params: nil,
			setupMocks: func(mockDB *MockDB) {
				now := time.Now()
				rows := NewMockRows([][]interface{}{
					{
						Uint64Ptr(1),
						"John",
						"Doe",
						StringPtr("john@example.com"),
						StringPtr("hashedpassword"),
						"employee",
						"active",
						Uint64Ptr(1),
						nil,
						StringPtr("Developer"),
						nil,
						nil,
						nil,
						nil,
						nil,
						TimePtr(now),
						nil,
						BoolPtr(true),
						Uint64Ptr(25),
						Uint64Ptr(10),
						TimePtr(now),
						TimePtr(now),
					},
					{
						Uint64Ptr(2),
						"Jane",
						"Smith",
						StringPtr("jane@example.com"),
						StringPtr("hashedpassword"),
						"manager",
						"active",
						Uint64Ptr(2),
						nil,
						StringPtr("Manager"),
						nil,
						nil,
						nil,
						nil,
						nil,
						TimePtr(now),
						nil,
						BoolPtr(true),
						Uint64Ptr(30),
						Uint64Ptr(5),
						TimePtr(now),
						TimePtr(now),
					},
				}, nil, EmployeeFieldDescriptions)
				mockDB.On("Query", mock.Anything, mock.AnythingOfType("string")).Return(rows, nil)
			},
			expectError: false,
			expectedLen: 2,
		},
		{
			name: "get employees with role filter",
			params: &entity.GetEmployeesParams{
				Role: StringPtr("manager"),
			},
			setupMocks: func(mockDB *MockDB) {
				now := time.Now()
				rows := NewMockRows([][]interface{}{
					{
						Uint64Ptr(2),
						"Jane",
						"Smith",
						StringPtr("jane@example.com"),
						StringPtr("hashedpassword"),
						"manager",
						"active",
						Uint64Ptr(2),
						nil,
						StringPtr("Manager"),
						nil,
						nil,
						nil,
						nil,
						nil,
						TimePtr(now),
						nil,
						BoolPtr(true),
						Uint64Ptr(30),
						Uint64Ptr(5),
						TimePtr(now),
						TimePtr(now),
					},
				}, nil, EmployeeFieldDescriptions)
				mockDB.On("Query", mock.Anything, mock.AnythingOfType("string"), "manager").Return(rows, nil)
			},
			expectError: false,
			expectedLen: 1,
		},
		{
			name: "get employees with department filter",
			params: &entity.GetEmployeesParams{
				DepartmentID: Uint64Ptr(1),
			},
			setupMocks: func(mockDB *MockDB) {
				now := time.Now()
				rows := NewMockRows([][]interface{}{
					{
						Uint64Ptr(1),
						"John",
						"Doe",
						StringPtr("john@example.com"),
						StringPtr("hashedpassword"),
						"employee",
						"active",
						Uint64Ptr(1),
						nil,
						StringPtr("Developer"),
						nil,
						nil,
						nil,
						nil,
						nil,
						TimePtr(now),
						nil,
						BoolPtr(true),
						Uint64Ptr(25),
						Uint64Ptr(10),
						TimePtr(now),
						TimePtr(now),
					},
				}, nil, EmployeeFieldDescriptions)
				mockDB.On("Query", mock.Anything, mock.AnythingOfType("string"), uint64(1)).Return(rows, nil)
			},
			expectError: false,
			expectedLen: 1,
		},
		{
			name: "get employees with status filter",
			params: &entity.GetEmployeesParams{
				Status: StringPtr("active"),
			},
			setupMocks: func(mockDB *MockDB) {
				now := time.Now()
				rows := NewMockRows([][]interface{}{
					{
						Uint64Ptr(1),
						"John",
						"Doe",
						StringPtr("john@example.com"),
						StringPtr("hashedpassword"),
						"employee",
						"active",
						Uint64Ptr(1),
						nil,
						StringPtr("Developer"),
						nil,
						nil,
						nil,
						nil,
						nil,
						TimePtr(now),
						nil,
						BoolPtr(true),
						Uint64Ptr(25),
						Uint64Ptr(10),
						TimePtr(now),
						TimePtr(now),
					},
				}, nil, EmployeeFieldDescriptions)
				mockDB.On("Query", mock.Anything, mock.AnythingOfType("string"), "active").Return(rows, nil)
			},
			expectError: false,
			expectedLen: 1,
		},
		{
			name: "get employees with all filters",
			params: &entity.GetEmployeesParams{
				Role:         StringPtr("manager"),
				DepartmentID: Uint64Ptr(1),
				Status:       StringPtr("active"),
			},
			setupMocks: func(mockDB *MockDB) {
				now := time.Now()
				rows := NewMockRows([][]interface{}{
					{
						Uint64Ptr(2),
						"Jane",
						"Smith",
						StringPtr("jane@example.com"),
						StringPtr("hashedpassword"),
						"manager",
						"active",
						Uint64Ptr(1),
						nil,
						StringPtr("Manager"),
						nil,
						nil,
						nil,
						nil,
						nil,
						TimePtr(now),
						nil,
						BoolPtr(true),
						Uint64Ptr(30),
						Uint64Ptr(5),
						TimePtr(now),
						TimePtr(now),
					},
				}, nil, EmployeeFieldDescriptions)
				mockDB.On("Query", mock.Anything, mock.AnythingOfType("string"), "manager", uint64(1), "active").Return(rows, nil)
			},
			expectError: false,
			expectedLen: 1,
		},
		{
			name:   "database query error",
			params: nil,
			setupMocks: func(mockDB *MockDB) {
				mockDB.On("Query", mock.Anything, mock.AnythingOfType("string")).Return((*MockRows)(nil), errors.New("query error"))
			},
			expectError: true,
		},
		{
			name:   "rows collection error",
			params: nil,
			setupMocks: func(mockDB *MockDB) {
				rows := NewMockRows(nil, errors.New("collection error"), EmployeeFieldDescriptions)
				mockDB.On("Query", mock.Anything, mock.AnythingOfType("string")).Return(rows, nil)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}
			deps := CreateTestDependencies(mockDB, &MockRedis{})

			tt.setupMocks(mockDB)

			controller := NewEmployeeController(deps)
			employees, err := controller.GetEmployees(tt.params)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, employees)
			} else {
				assert.NoError(t, err)
				assert.Len(t, employees, tt.expectedLen)
				for _, emp := range employees {
					assert.Nil(t, emp.Password)
				}
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestEmployeeController_GetEmployeeByID(t *testing.T) {
	tests := []struct {
		name          string
		employeeID    uint64
		setupMocks    func(*MockDB)
		expectError   bool
		errorContains string
	}{
		{
			name:       "successful get employee",
			employeeID: 1,
			setupMocks: func(mockDB *MockDB) {
				rows := NewMockRows([][]interface{}{
					{uint64(1), "John", "Doe", "john@example.com", "employee"},
				}, nil, EmployeeFieldDescriptions)
				mockDB.On("Query", mock.Anything, mock.AnythingOfType("string"), uint64(1)).Return(rows, nil)
			},
			expectError: false,
		},
		{
			name:       "employee not found",
			employeeID: 999,
			setupMocks: func(mockDB *MockDB) {
				rows := NewMockRows(nil, pgx.ErrNoRows, EmployeeFieldDescriptions)
				mockDB.On("Query", mock.Anything, mock.AnythingOfType("string"), uint64(999)).Return(rows, nil)
			},
			expectError:   true,
			errorContains: "employee not found",
		},
		{
			name:       "database error",
			employeeID: 1,
			setupMocks: func(mockDB *MockDB) {
				mockDB.On("Query", mock.Anything, mock.AnythingOfType("string"), uint64(1)).Return((*MockRows)(nil), errors.New("db error"))
			},
			expectError: true,
		},
		{
			name:       "collection error",
			employeeID: 1,
			setupMocks: func(mockDB *MockDB) {
				rows := NewMockRows(nil, errors.New("collection error"), EmployeeFieldDescriptions)
				mockDB.On("Query", mock.Anything, mock.AnythingOfType("string"), uint64(1)).Return(rows, nil)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}
			deps := CreateTestDependencies(mockDB, &MockRedis{})

			tt.setupMocks(mockDB)

			controller := NewEmployeeController(deps)
			employee, err := controller.GetEmployeeByID(tt.employeeID)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, employee)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, employee)
				assert.Nil(t, employee.Password)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestEmployeeController_CreateEmployee(t *testing.T) {
	tests := []struct {
		name          string
		employee      entity.Employee
		setupMocks    func(*MockDB)
		expectError   bool
		errorContains string
	}{
		{
			name: "missing required fields",
			employee: entity.Employee{
				FirstName: "",
				LastName:  "Doe",
				Email:     StringPtr("john@example.com"),
			},
			setupMocks:    func(mockDB *MockDB) {},
			expectError:   true,
			errorContains: "required fields",
		},
		{
			name: "empty email",
			employee: entity.Employee{
				FirstName: "John",
				LastName:  "Doe",
				Email:     StringPtr(""),
			},
			setupMocks:    func(mockDB *MockDB) {},
			expectError:   true,
			errorContains: "required fields",
		},
		{
			name: "nil email",
			employee: entity.Employee{
				FirstName: "John",
				LastName:  "Doe",
				Email:     nil,
			},
			setupMocks:    func(mockDB *MockDB) {},
			expectError:   true,
			errorContains: "required fields",
		},
		{
			name: "uniqueness check database error",
			employee: entity.Employee{
				FirstName: "John",
				LastName:  "Doe",
				Email:     StringPtr("john@example.com"),
			},
			setupMocks: func(mockDB *MockDB) {
				countRow := NewMockRow(nil, errors.New("db error"), []pgconn.FieldDescription{{Name: "count", DataTypeOID: 20}})
				mockDB.On("QueryRow", mock.Anything, "SELECT COUNT(*) FROM employees WHERE email = $1 OR (personal_number IS NOT NULL AND personal_number = $2)", mock.Anything, mock.Anything).Return(countRow)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}
			deps := CreateTestDependencies(mockDB, &MockRedis{})

			tt.setupMocks(mockDB)

			controller := NewEmployeeController(deps)
			result, err := controller.CreateEmployee(tt.employee)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotNil(t, result.ID)
				assert.NotNil(t, result.CreatedAt)
				assert.NotNil(t, result.UpdatedAt)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestEmployeeController_RequestVacation(t *testing.T) {
	tests := []struct {
		name          string
		employeeID    uint64
		days          uint64
		setupMocks    func(*MockDB)
		expectError   bool
		errorContains string
	}{
		{
			name:       "successful vacation request",
			employeeID: 1,
			days:       5,
			setupMocks: func(mockDB *MockDB) {
				rows := NewMockRows([][]interface{}{
					{uint64(1), "John", "Doe", "john@example.com", "employee", uint64(30)},
				}, nil, EmployeeFieldDescriptions)
				mockDB.On("Query", mock.Anything, mock.MatchedBy(func(query string) bool {
					return query[:6] == "UPDATE"
				}), uint64(5), mock.AnythingOfType("time.Time"), uint64(1)).Return(rows, nil)
			},
			expectError: false,
		},
		{
			name:          "invalid vacation days - zero",
			employeeID:    1,
			days:          0,
			setupMocks:    func(mockDB *MockDB) {},
			expectError:   true,
			errorContains: "invalid vacation days",
		},
		{
			name:       "employee not found",
			employeeID: 999,
			days:       5,
			setupMocks: func(mockDB *MockDB) {
				rows := NewMockRows(nil, pgx.ErrNoRows, EmployeeFieldDescriptions)
				mockDB.On("Query", mock.Anything, mock.MatchedBy(func(query string) bool {
					return query[:6] == "UPDATE"
				}), uint64(5), mock.AnythingOfType("time.Time"), uint64(999)).Return(rows, nil)
			},
			expectError:   true,
			errorContains: "employee not found",
		},
		{
			name:       "database update error",
			employeeID: 1,
			days:       5,
			setupMocks: func(mockDB *MockDB) {
				mockDB.On("Query", mock.Anything, mock.MatchedBy(func(query string) bool {
					return query[:6] == "UPDATE"
				}), uint64(5), mock.AnythingOfType("time.Time"), uint64(1)).Return((*MockRows)(nil), errors.New("update error"))
			},
			expectError: true,
		},
		{
			name:       "collection error",
			employeeID: 1,
			days:       5,
			setupMocks: func(mockDB *MockDB) {
				rows := NewMockRows(nil, errors.New("collection error"), EmployeeFieldDescriptions)
				mockDB.On("Query", mock.Anything, mock.MatchedBy(func(query string) bool {
					return query[:6] == "UPDATE"
				}), uint64(5), mock.AnythingOfType("time.Time"), uint64(1)).Return(rows, nil)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}
			deps := CreateTestDependencies(mockDB, &MockRedis{})

			tt.setupMocks(mockDB)

			controller := NewEmployeeController(deps)
			result, err := controller.RequestVacation(tt.employeeID, tt.days)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Nil(t, result.Password)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestEmployeeController_getPasswordHash(t *testing.T) {
	tests := []struct {
		name        string
		newPassword *string
		employeeID  uint64
		setupMocks  func(*MockDB)
		expectError bool
	}{
		{
			name:        "new password provided",
			newPassword: StringPtr("newpassword123"),
			employeeID:  1,
			setupMocks:  func(mockDB *MockDB) {},
			expectError: false,
		},
		{
			name:        "no new password - get existing",
			newPassword: nil,
			employeeID:  1,
			setupMocks: func(mockDB *MockDB) {
				row := NewMockRow([]interface{}{"existinghash"}, nil, EmployeeFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, "SELECT password FROM employees WHERE id = $1", uint64(1)).Return(row)
			},
			expectError: false,
		},
		{
			name:        "no new password - employee not found",
			newPassword: nil,
			employeeID:  999,
			setupMocks: func(mockDB *MockDB) {
				row := NewMockRow(nil, pgx.ErrNoRows, EmployeeFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, "SELECT password FROM employees WHERE id = $1", uint64(999)).Return(row)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}
			deps := CreateTestDependencies(mockDB, &MockRedis{})

			tt.setupMocks(mockDB)

			controller := NewEmployeeController(deps)
			result, err := controller.getPasswordHash(tt.newPassword, tt.employeeID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, result, "")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestEmployeeController_UpdateEmployee(t *testing.T) {
	tests := []struct {
		name          string
		employeeID    uint64
		employee      entity.Employee
		setupMocks    func(*MockDB)
		expectError   bool
		errorContains string
	}{
		{
			name:       "successful update",
			employeeID: 1,
			employee: entity.Employee{
				FirstName: "John",
				LastName:  "Doe",
				Email:     StringPtr("john.updated@example.com"),
				Role:      "manager",
			},
			setupMocks: func(mockDB *MockDB) {
				passwordRow := NewMockRow([]interface{}{"existinghash"}, nil, EmployeeFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, "SELECT password FROM employees WHERE id = $1", uint64(1)).Return(passwordRow)

				countRow := NewMockRow([]interface{}{0}, nil, EmployeeFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, mock.MatchedBy(func(query string) bool {
					return query == "SELECT COUNT(*) FROM employees WHERE (email = $1 OR (personal_number IS NOT NULL AND personal_number = $2)) AND id != $3"
				}), mock.Anything, mock.Anything, uint64(1)).Return(countRow)

				updateRows := NewMockRows([][]interface{}{
					{uint64(1), "John", "Doe", "john.updated@example.com", "manager"},
				}, nil, EmployeeFieldDescriptions)
				mockDB.On("Query", mock.Anything, mock.MatchedBy(func(query string) bool {
					return query[:6] == "UPDATE"
				}), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, uint64(1)).Return(updateRows, nil)
			},
			expectError: false,
		},
		{
			name:       "missing required fields",
			employeeID: 1,
			employee: entity.Employee{
				FirstName: "",
				LastName:  "Doe",
				Email:     StringPtr("john@example.com"),
			},
			setupMocks:    func(mockDB *MockDB) {},
			expectError:   true,
			errorContains: "invalid employee data",
		},
		{
			name:       "get password hash error - employee not found",
			employeeID: 999,
			employee: entity.Employee{
				FirstName: "John",
				LastName:  "Doe",
				Email:     StringPtr("john@example.com"),
			},
			setupMocks: func(mockDB *MockDB) {
				passwordRow := NewMockRow(nil, pgx.ErrNoRows, EmployeeFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, "SELECT password FROM employees WHERE id = $1", uint64(999)).Return(passwordRow)
			},
			expectError:   true,
			errorContains: "employee not found",
		},
		{
			name:       "update database error",
			employeeID: 1,
			employee: entity.Employee{
				FirstName: "John",
				LastName:  "Doe",
				Email:     StringPtr("john@example.com"),
			},
			setupMocks: func(mockDB *MockDB) {
				passwordRow := NewMockRow([]interface{}{"existinghash"}, nil, EmployeeFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, "SELECT password FROM employees WHERE id = $1", uint64(1)).Return(passwordRow)

				countRow := NewMockRow([]interface{}{0}, nil, EmployeeFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, mock.MatchedBy(func(query string) bool {
					return query == "SELECT COUNT(*) FROM employees WHERE (email = $1 OR (personal_number IS NOT NULL AND personal_number = $2)) AND id != $3"
				}), mock.Anything, mock.Anything, uint64(1)).Return(countRow)

				mockDB.On("Query", mock.Anything, mock.MatchedBy(func(query string) bool {
					return query[:6] == "UPDATE"
				}), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, uint64(1)).Return((*MockRows)(nil), errors.New("update error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}
			deps := CreateTestDependencies(mockDB, &MockRedis{})

			tt.setupMocks(mockDB)

			controller := NewEmployeeController(deps)
			result, err := controller.UpdateEmployee(tt.employeeID, tt.employee)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestEmployeeController_DeleteEmployee(t *testing.T) {
	tests := []struct {
		name          string
		employeeID    uint64
		setupMocks    func(*MockDB)
		expectError   bool
		errorContains string
	}{
		{
			name:       "successful delete",
			employeeID: 1,
			setupMocks: func(mockDB *MockDB) {
				commandTag := NewMockCommandTag(1)
				mockDB.On("Exec", mock.Anything, "DELETE FROM employees WHERE id = $1", uint64(1)).Return(commandTag, nil)
			},
			expectError: false,
		},
		{
			name:       "employee not found",
			employeeID: 999,
			setupMocks: func(mockDB *MockDB) {
				commandTag := NewMockCommandTag(0)
				mockDB.On("Exec", mock.Anything, "DELETE FROM employees WHERE id = $1", uint64(999)).Return(commandTag, nil)
			},
			expectError:   true,
			errorContains: "employee not found",
		},
		{
			name:       "database error",
			employeeID: 1,
			setupMocks: func(mockDB *MockDB) {
				commandTag := NewMockCommandTag(0)
				mockDB.On("Exec", mock.Anything, "DELETE FROM employees WHERE id = $1", uint64(1)).Return(commandTag, errors.New("db error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}
			deps := CreateTestDependencies(mockDB, &MockRedis{})

			tt.setupMocks(mockDB)

			controller := NewEmployeeController(deps)
			err := controller.DeleteEmployee(tt.employeeID)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestEmployeeController_updateEmployeeInDB(t *testing.T) {
	tests := []struct {
		name          string
		employee      *entity.Employee
		employeeID    uint64
		setupMocks    func(*MockDB)
		expectError   bool
		errorContains string
	}{
		{
			name: "successful update in DB",
			employee: &entity.Employee{
				FirstName: "John",
				LastName:  "Doe",
				Email:     StringPtr("john@example.com"),
			},
			employeeID: 1,
			setupMocks: func(mockDB *MockDB) {
				updateRows := NewMockRows([][]interface{}{
					{uint64(1), "John", "Doe", "john@example.com", "employee"},
				}, nil, EmployeeFieldDescriptions)
				mockDB.On("Query", mock.Anything, mock.MatchedBy(func(query string) bool {
					return query[:6] == "UPDATE"
				}), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, uint64(1)).Return(updateRows, nil)
			},
			expectError: false,
		},
		{
			name: "update query error",
			employee: &entity.Employee{
				FirstName: "John",
				LastName:  "Doe",
				Email:     StringPtr("john@example.com"),
			},
			employeeID: 1,
			setupMocks: func(mockDB *MockDB) {
				mockDB.On("Query", mock.Anything, mock.MatchedBy(func(query string) bool {
					return query[:6] == "UPDATE"
				}), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, uint64(1)).Return((*MockRows)(nil), errors.New("query error"))
			},
			expectError:   true,
			errorContains: "failed to update employee",
		},
		{
			name: "employee not found in update",
			employee: &entity.Employee{
				FirstName: "John",
				LastName:  "Doe",
				Email:     StringPtr("john@example.com"),
			},
			employeeID: 999,
			setupMocks: func(mockDB *MockDB) {
				updateRows := NewMockRows(nil, pgx.ErrNoRows, EmployeeFieldDescriptions)
				mockDB.On("Query", mock.Anything, mock.MatchedBy(func(query string) bool {
					return query[:6] == "UPDATE"
				}), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, uint64(999)).Return(updateRows, nil)
			},
			expectError:   true,
			errorContains: "employee not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}
			deps := CreateTestDependencies(mockDB, &MockRedis{})

			tt.setupMocks(mockDB)

			controller := NewEmployeeController(deps)
			result, err := controller.updateEmployeeInDB(tt.employee, tt.employeeID)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Nil(t, result.Password)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestNewEmployeeController(t *testing.T) {
	mockDB := &MockDB{}
	mockRedis := &MockRedis{}
	deps := CreateTestDependencies(mockDB, mockRedis)

	controller := NewEmployeeController(deps)

	assert.NotNil(t, controller)
	assert.Equal(t, deps, controller.deps)
}
