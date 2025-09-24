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

var DepartmentFieldDescriptions = []pgconn.FieldDescription{
	{Name: "id", DataTypeOID: 20},           // int8 (uint64)
	{Name: "name", DataTypeOID: 25},         // text (string)
	{Name: "description", DataTypeOID: 25},  // text (string)
	{Name: "parent_id", DataTypeOID: 20},    // int8 (uint64, nullable)
	{Name: "head_id", DataTypeOID: 20},      // int8 (uint64, nullable)
	{Name: "created_at", DataTypeOID: 1114}, // timestamp
	{Name: "updated_at", DataTypeOID: 1114}, // timestamp
}

func TestDepartmentController_GetDepartments(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*MockDB)
		expectError bool
		expectedLen int
	}{
		{
			name: "successful get departments",
			setupMocks: func(mockDB *MockDB) {
				now := time.Now()
				rows := NewMockRows([][]interface{}{
					{uint64(1), "Engineering", "Software Engineering Department", (*uint64)(nil), uint64(1), now, now},
					{uint64(2), "HR", "Human Resources Department", (*uint64)(nil), uint64(2), now, now},
				}, nil, DepartmentFieldDescriptions)

				mockDB.On("Query", mock.Anything, mock.AnythingOfType("string")).Return(rows, nil)
			},
			expectError: false,
			expectedLen: 2,
		},
		{
			name: "empty departments list",
			setupMocks: func(mockDB *MockDB) {
				rows := NewMockRows([][]interface{}{}, nil, DepartmentFieldDescriptions)
				mockDB.On("Query", mock.Anything, mock.AnythingOfType("string")).Return(rows, nil)
			},
			expectError: false,
			expectedLen: 0,
		},
		{
			name: "database query error",
			setupMocks: func(mockDB *MockDB) {
				mockDB.On("Query", mock.Anything, mock.AnythingOfType("string")).Return((*MockRows)(nil), errors.New("query error"))
			},
			expectError: true,
		},
		{
			name: "rows collection error",
			setupMocks: func(mockDB *MockDB) {
				rows := NewMockRows(nil, errors.New("collection error"), DepartmentFieldDescriptions)
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

			controller := NewDepartmentController(deps)
			departments, err := controller.GetDepartments()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, departments)
			} else {
				assert.NoError(t, err)
				assert.Len(t, departments, tt.expectedLen)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestDepartmentController_GetDepartmentByID(t *testing.T) {
	tests := []struct {
		name          string
		departmentID  uint64
		setupMocks    func(*MockDB)
		expectError   bool
		errorContains string
	}{
		{
			name:         "successful get department",
			departmentID: 1,
			setupMocks: func(mockDB *MockDB) {
				now := time.Now()
				rows := NewMockRows([][]interface{}{
					{uint64(1), "Engineering", "Software Engineering Department", uint64(0), uint64(1), now, now},
				}, nil, DepartmentFieldDescriptions)
				mockDB.On("Query", mock.Anything, mock.AnythingOfType("string"), uint64(1)).Return(rows, nil)
			},
			expectError: false,
		},
		{
			name:         "department not found",
			departmentID: 999,
			setupMocks: func(mockDB *MockDB) {
				rows := NewMockRows(nil, pgx.ErrNoRows, DepartmentFieldDescriptions)
				mockDB.On("Query", mock.Anything, mock.AnythingOfType("string"), uint64(999)).Return(rows, nil)
			},
			expectError:   true,
			errorContains: "department not found",
		},
		{
			name:         "database error",
			departmentID: 1,
			setupMocks: func(mockDB *MockDB) {
				mockDB.On("Query", mock.Anything, mock.AnythingOfType("string"), uint64(1)).Return((*MockRows)(nil), errors.New("db error"))
			},
			expectError: true,
		},
		{
			name:         "collection error",
			departmentID: 1,
			setupMocks: func(mockDB *MockDB) {
				rows := NewMockRows(nil, errors.New("collection error"), DepartmentFieldDescriptions)
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

			controller := NewDepartmentController(deps)
			department, err := controller.GetDepartmentByID(tt.departmentID)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, department)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, department)
				assert.Equal(t, tt.departmentID, department.ID)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestDepartmentController_CreateDepartment(t *testing.T) {
	tests := []struct {
		name          string
		department    entity.Department
		setupMocks    func(*MockDB)
		expectError   bool
		errorContains string
	}{
		{
			name: "successful create department",
			department: entity.Department{
				Name:        "Engineering",
				Description: "Software Engineering Department",
				ParentID:    Uint64Ptr(0),
				HeadID:      Uint64Ptr(1),
			},
			setupMocks: func(mockDB *MockDB) {
				// Mock insert operation
				insertRow := NewMockRow([]interface{}{uint64(1)}, nil, DepartmentFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, mock.MatchedBy(func(query string) bool {
					return query[:6] == "INSERT"
				}), "Engineering", "Software Engineering Department", mock.Anything, mock.Anything, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(insertRow)
			},
			expectError: false,
		},
		{
			name: "create department with minimal data",
			department: entity.Department{
				Name:        "HR",
				Description: "",
				ParentID:    nil,
				HeadID:      nil,
			},
			setupMocks: func(mockDB *MockDB) {
				insertRow := NewMockRow([]interface{}{uint64(2)}, nil, DepartmentFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, mock.MatchedBy(func(query string) bool {
					return query[:6] == "INSERT"
				}), "HR", "", (*uint64)(nil), (*uint64)(nil), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(insertRow)
			},
			expectError: false,
		},
		{
			name: "missing required name",
			department: entity.Department{
				Name:        "",
				Description: "Test Department",
			},
			setupMocks:    func(mockDB *MockDB) {},
			expectError:   true,
			errorContains: "name is required",
		},
		{
			name: "database insert error",
			department: entity.Department{
				Name:        "Engineering",
				Description: "Software Engineering Department",
			},
			setupMocks: func(mockDB *MockDB) {
				insertRow := NewMockRow(nil, errors.New("insert error"), DepartmentFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, mock.MatchedBy(func(query string) bool {
					return query[:6] == "INSERT"
				}), "Engineering", "Software Engineering Department", (*uint64)(nil), (*uint64)(nil), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(insertRow)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}
			deps := CreateTestDependencies(mockDB, &MockRedis{})

			tt.setupMocks(mockDB)

			controller := NewDepartmentController(deps)
			result, err := controller.CreateDepartment(tt.department)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotZero(t, result.ID)
				assert.NotZero(t, result.CreatedAt)
				assert.NotZero(t, result.UpdatedAt)
				assert.Equal(t, tt.department.Name, result.Name)
				assert.Equal(t, tt.department.Description, result.Description)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestDepartmentController_UpdateDepartment(t *testing.T) {
	tests := []struct {
		name          string
		department    entity.Department
		departmentID  uint64
		setupMocks    func(*MockDB)
		expectError   bool
		errorContains string
	}{
		{
			name: "successful update department",
			department: entity.Department{
				Name:        "Updated Engineering",
				Description: "Updated Software Engineering Department",
				ParentID:    Uint64Ptr(1),
				HeadID:      Uint64Ptr(2),
			},
			departmentID: 1,
			setupMocks: func(mockDB *MockDB) {
				now := time.Now()
				updateRow := NewMockRow([]interface{}{
					uint64(1), "Updated Engineering", "Updated Software Engineering Department",
					uint64(1), uint64(2), now, now,
				}, nil, DepartmentFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, mock.MatchedBy(func(query string) bool {
					return query[:6] == "UPDATE"
				}), "Updated Engineering", "Updated Software Engineering Department", mock.Anything, mock.Anything, mock.AnythingOfType("time.Time"), uint64(1)).Return(updateRow)
			},
			expectError: false,
		},
		{
			name: "update department with minimal data",
			department: entity.Department{
				Name:        "Updated HR",
				Description: "",
				ParentID:    nil,
				HeadID:      nil,
			},
			departmentID: 2,
			setupMocks: func(mockDB *MockDB) {
				now := time.Now()
				updateRow := NewMockRow([]interface{}{
					uint64(2), "Updated HR", "", nil, nil, now, now,
				}, nil, DepartmentFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, mock.MatchedBy(func(query string) bool {
					return query[:6] == "UPDATE"
				}), "Updated HR", "", (*uint64)(nil), (*uint64)(nil), mock.AnythingOfType("time.Time"), uint64(2)).Return(updateRow)
			},
			expectError: false,
		},
		{
			name: "department not found",
			department: entity.Department{
				Name:        "Updated Engineering",
				Description: "Updated Software Engineering Department",
			},
			departmentID: 999,
			setupMocks: func(mockDB *MockDB) {
				updateRow := NewMockRow(nil, pgx.ErrNoRows, DepartmentFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, mock.MatchedBy(func(query string) bool {
					return query[:6] == "UPDATE"
				}), "Updated Engineering", "Updated Software Engineering Department", (*uint64)(nil), (*uint64)(nil), mock.AnythingOfType("time.Time"), uint64(999)).Return(updateRow)
			},
			expectError:   true,
			errorContains: "department not found",
		},
		{
			name: "database update error",
			department: entity.Department{
				Name:        "Updated Engineering",
				Description: "Updated Software Engineering Department",
			},
			departmentID: 1,
			setupMocks: func(mockDB *MockDB) {
				updateRow := NewMockRow(nil, errors.New("update error"), DepartmentFieldDescriptions)
				mockDB.On("QueryRow", mock.Anything, mock.MatchedBy(func(query string) bool {
					return query[:6] == "UPDATE"
				}), "Updated Engineering", "Updated Software Engineering Department", (*uint64)(nil), (*uint64)(nil), mock.AnythingOfType("time.Time"), uint64(1)).Return(updateRow)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}
			deps := CreateTestDependencies(mockDB, &MockRedis{})

			tt.setupMocks(mockDB)

			controller := NewDepartmentController(deps)
			result, err := controller.UpdateDepartment(tt.department, tt.departmentID)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.departmentID, result.ID)
				assert.Equal(t, tt.department.Name, result.Name)
				assert.Equal(t, tt.department.Description, result.Description)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestDepartmentController_DeleteDepartment(t *testing.T) {
	tests := []struct {
		name          string
		departmentID  uint64
		setupMocks    func(*MockDB)
		expectError   bool
		errorContains string
	}{
		{
			name:         "successful delete",
			departmentID: 1,
			setupMocks: func(mockDB *MockDB) {
				commandTag := NewMockCommandTag(1)

				mockDB.On("Exec", mock.Anything, "DELETE FROM departments WHERE id = $1", uint64(1)).Return(commandTag, nil)
			},
			expectError: false,
		},
		{
			name:         "department not found - no error returned per implementation",
			departmentID: 999,
			setupMocks: func(mockDB *MockDB) {
				commandTag := NewMockCommandTag(0)
				mockDB.On("Exec", mock.Anything, "DELETE FROM departments WHERE id = $1", uint64(999)).Return(commandTag, nil)
			},
			expectError: false,
		},
		{
			name:         "database error",
			departmentID: 1,
			setupMocks: func(mockDB *MockDB) {
				commandTag := NewMockCommandTag(0)
				mockDB.On("Exec", mock.Anything, "DELETE FROM departments WHERE id = $1", uint64(1)).Return(commandTag, errors.New("db error"))
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}
			deps := CreateTestDependencies(mockDB, &MockRedis{})

			tt.setupMocks(mockDB)

			controller := NewDepartmentController(deps)
			err := controller.DeleteDepartment(tt.departmentID)

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

func TestDepartmentController_DeleteDepartment_ExpectedBehavior(t *testing.T) {
	tests := []struct {
		name          string
		departmentID  uint64
		setupMocks    func(*MockDB)
		expectError   bool
		errorContains string
	}{
		{
			name:         "successful delete",
			departmentID: 1,
			setupMocks: func(mockDB *MockDB) {
				commandTag := NewMockCommandTag(1)
				mockDB.On("Exec", mock.Anything, "DELETE FROM departments WHERE id = $1", uint64(1)).Return(commandTag, nil)
			},
			expectError: false,
		},
		{
			name:         "department not found",
			departmentID: 999,
			setupMocks: func(mockDB *MockDB) {
				commandTag := NewMockCommandTag(0)
				mockDB.On("Exec", mock.Anything, "DELETE FROM departments WHERE id = $1", uint64(999)).Return(commandTag, nil)
			},
			expectError: false,
		},
		{
			name:         "database error",
			departmentID: 1,
			setupMocks: func(mockDB *MockDB) {
				commandTag := NewMockCommandTag(0)
				mockDB.On("Exec", mock.Anything, "DELETE FROM departments WHERE id = $1", uint64(1)).Return(commandTag, errors.New("db error"))
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}
			deps := CreateTestDependencies(mockDB, &MockRedis{})

			tt.setupMocks(mockDB)

			controller := NewDepartmentController(deps)
			err := controller.DeleteDepartment(tt.departmentID)

			assert.NoError(t, err)

			mockDB.AssertExpectations(t)
		})
	}
}

func TestNewDepartmentController(t *testing.T) {
	mockDB := &MockDB{}
	mockRedis := &MockRedis{}
	deps := CreateTestDependencies(mockDB, mockRedis)

	controller := NewDepartmentController(deps)

	assert.NotNil(t, controller)
	assert.Equal(t, deps, controller.deps)
}

func TestDepartmentController_EdgeCases(t *testing.T) {
	t.Run("GetDepartments with single department", func(t *testing.T) {
		mockDB := &MockDB{}
		deps := CreateTestDependencies(mockDB, &MockRedis{})

		now := time.Now()
		rows := NewMockRows([][]interface{}{
			{uint64(1), "Engineering", "Software Engineering Department", uint64(0), uint64(1), now, now},
		}, nil, DepartmentFieldDescriptions)
		mockDB.On("Query", mock.Anything, mock.AnythingOfType("string")).Return(rows, nil)

		controller := NewDepartmentController(deps)
		departments, err := controller.GetDepartments()

		assert.NoError(t, err)
		assert.Len(t, departments, 1)
		assert.Equal(t, "Engineering", departments[0].Name)

		mockDB.AssertExpectations(t)
	})

	t.Run("CreateDepartment with very long name", func(t *testing.T) {
		mockDB := &MockDB{}
		deps := CreateTestDependencies(mockDB, &MockRedis{})

		longName := string(make([]byte, 1000))
		for i := range longName {
			longName = string(append([]byte(longName[:i]), 'A'))
		}

		department := entity.Department{
			Name:        longName,
			Description: "Test Department with very long name",
		}

		insertRow := NewMockRow([]interface{}{uint64(1)}, nil, DepartmentFieldDescriptions)
		mockDB.On("QueryRow", mock.Anything, mock.MatchedBy(func(query string) bool {
			return query[:6] == "INSERT"
		}), longName, "Test Department with very long name", (*uint64)(nil), (*uint64)(nil), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(insertRow)

		controller := NewDepartmentController(deps)
		result, err := controller.CreateDepartment(department)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, longName, result.Name)

		mockDB.AssertExpectations(t)
	})

	t.Run("UpdateDepartment with same data", func(t *testing.T) {
		mockDB := &MockDB{}
		deps := CreateTestDependencies(mockDB, &MockRedis{})

		department := entity.Department{
			Name:        "Engineering",
			Description: "Software Engineering Department",
		}

		now := time.Now()
		updateRow := NewMockRow([]interface{}{
			uint64(1), "Engineering", "Software Engineering Department", nil, nil, now, now,
		}, nil, DepartmentFieldDescriptions)
		mockDB.On("QueryRow", mock.Anything, mock.MatchedBy(func(query string) bool {
			return query[:6] == "UPDATE"
		}), "Engineering", "Software Engineering Department", (*uint64)(nil), (*uint64)(nil), mock.AnythingOfType("time.Time"), uint64(1)).Return(updateRow)

		controller := NewDepartmentController(deps)
		result, err := controller.UpdateDepartment(department, 1)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, department.Name, result.Name)

		mockDB.AssertExpectations(t)
	})
}
