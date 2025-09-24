package controllers

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/adamanr/employes_service/internal/config"
	"github.com/adamanr/employes_service/internal/entity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
)

// DBInterface defines the interface for database operations.
type DBInterface interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
	Close(ctx context.Context) error
	Ping(ctx context.Context) error
}

// RedisInterface defines the interface for Redis operations.
type RedisInterface interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Close() error
}

// MockDB represents a mock database connection.
type MockDB struct {
	mock.Mock
}

func (m *MockDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	mockArgs := append([]interface{}{ctx, sql}, args...)
	callArgs := m.Called(mockArgs...)
	return callArgs.Get(0).(pgx.Rows), callArgs.Error(1)
}

func (m *MockDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	mockArgs := append([]interface{}{ctx, sql}, args...)
	callArgs := m.Called(mockArgs...)
	return callArgs.Get(0).(pgx.Row)
}

func (m *MockDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	mockArgs := append([]interface{}{ctx, sql}, args...)
	callArgs := m.Called(mockArgs...)
	return callArgs.Get(0).(pgconn.CommandTag), callArgs.Error(1)
}

func (m *MockDB) Begin(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	return args.Get(0).(pgx.Tx), args.Error(1)
}

func (m *MockDB) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockDB) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockRow represents a mock database row.
type MockRow struct {
	mock.Mock
	data       []interface{}
	err        error
	fieldDescs []pgconn.FieldDescription
}

// NewMockRow creates a new MockRow instance with custom FieldDescriptions.
func NewMockRow(data []interface{}, err error, fieldDescs []pgconn.FieldDescription) *MockRow {
	if fieldDescs == nil {
		fieldDescs = []pgconn.FieldDescription{
			{Name: "id", DataTypeOID: 20},
			{Name: "first_name", DataTypeOID: 25},
			{Name: "last_name", DataTypeOID: 25},
			{Name: "email", DataTypeOID: 25},
			{Name: "password", DataTypeOID: 25},
			{Name: "role", DataTypeOID: 25},
			{Name: "status", DataTypeOID: 25},
			{Name: "department_id", DataTypeOID: 20},
			{Name: "manager_id", DataTypeOID: 20},
			{Name: "position", DataTypeOID: 25},
			{Name: "address", DataTypeOID: 25},
			{Name: "phone", DataTypeOID: 25},
			{Name: "personal_number", DataTypeOID: 25},
			{Name: "middle_name", DataTypeOID: 25},
			{Name: "birthday", DataTypeOID: 1114},
			{Name: "hire_date", DataTypeOID: 1114},
			{Name: "fire_date", DataTypeOID: 1114},
			{Name: "is_active", DataTypeOID: 16},
			{Name: "vacation_days", DataTypeOID: 20},
			{Name: "sick_days", DataTypeOID: 20},
			{Name: "created_at", DataTypeOID: 1114},
			{Name: "updated_at", DataTypeOID: 1114},
		}
	}
	return &MockRow{
		data:       data,
		err:        err,
		fieldDescs: fieldDescs,
	}
}

// FieldDescriptions returns the field descriptions for the row.
func (m *MockRow) FieldDescriptions() []pgconn.FieldDescription {
	return m.fieldDescs
}

// Scan scans the row data into the provided destinations.
func (m *MockRow) Scan(dest ...interface{}) error {
	if m.err != nil {
		return m.err
	}

	for i, val := range m.data {
		if i < len(dest) {
			switch d := dest[i].(type) {
			case *uint64:
				if v, ok := val.(uint64); ok {
					*d = v
				} else if v, ok := val.(*uint64); ok && v != nil {
					*d = *v
				}
			case *int64:
				if v, ok := val.(int64); ok {
					*d = v
				}
			case *string:
				if v, ok := val.(string); ok {
					*d = v
				} else if v, ok := val.(*string); ok && v != nil {
					*d = *v
				}
			case *time.Time:
				if v, ok := val.(time.Time); ok {
					*d = v
				} else if v, ok := val.(*time.Time); ok && v != nil {
					*d = *v
				}
			case *bool:
				if v, ok := val.(bool); ok {
					*d = v
				} else if v, ok := val.(*bool); ok && v != nil {
					*d = *v
				}
			case **uint64:
				if v, ok := val.(*uint64); ok {
					*d = v
				}
			case **string:
				if v, ok := val.(*string); ok {
					*d = v
				}
			case **time.Time:
				if v, ok := val.(*time.Time); ok {
					*d = v
				}
			case **bool:
				if v, ok := val.(*bool); ok {
					*d = v
				}
			case *interface{}:
				*d = val
			}
		}
	}
	return nil
}

// MockRows represents mock database rows.
type MockRows struct {
	mock.Mock
	rows       [][]interface{}
	pos        int
	err        error
	fieldDescs []pgconn.FieldDescription
}

func NewMockRows(rows [][]interface{}, err error, fieldDescs []pgconn.FieldDescription) *MockRows {
	if fieldDescs == nil {
		fieldDescs = []pgconn.FieldDescription{
			{Name: "id", DataTypeOID: 20},
			{Name: "first_name", DataTypeOID: 25},
			{Name: "last_name", DataTypeOID: 25},
			{Name: "email", DataTypeOID: 25},
			{Name: "password", DataTypeOID: 25},
			{Name: "role", DataTypeOID: 25},
			{Name: "status", DataTypeOID: 25},
			{Name: "department_id", DataTypeOID: 20},
			{Name: "manager_id", DataTypeOID: 20},
			{Name: "position", DataTypeOID: 25},
			{Name: "address", DataTypeOID: 25},
			{Name: "phone", DataTypeOID: 25},
			{Name: "personal_number", DataTypeOID: 25},
			{Name: "middle_name", DataTypeOID: 25},
			{Name: "birthday", DataTypeOID: 1114},
			{Name: "hire_date", DataTypeOID: 1114},
			{Name: "fire_date", DataTypeOID: 1114},
			{Name: "is_active", DataTypeOID: 16},
			{Name: "vacation_days", DataTypeOID: 20},
			{Name: "sick_days", DataTypeOID: 20},
			{Name: "created_at", DataTypeOID: 1114},
			{Name: "updated_at", DataTypeOID: 1114},
		}
	}
	return &MockRows{
		rows:       rows,
		pos:        -1,
		err:        err,
		fieldDescs: fieldDescs,
	}
}

func (m *MockRows) FieldDescriptions() []pgconn.FieldDescription {
	return m.fieldDescs
}

func (m *MockRows) WithFieldDescriptions(fieldDescs []pgconn.FieldDescription) *MockRows {
	m.fieldDescs = fieldDescs
	return m
}

func (m *MockRows) Next() bool {
	m.pos++
	return m.pos < len(m.rows)
}

func (m *MockRows) Close() {}

func (m *MockRows) Scan(dest ...interface{}) error {
	if m.err != nil {
		return m.err
	}
	if m.pos >= len(m.rows) {
		return nil
	}

	row := m.rows[m.pos]
	for i, val := range row {
		if i < len(dest) {
			switch d := dest[i].(type) {
			case *uint64:
				if v, ok := val.(uint64); ok {
					*d = v
				}
			case *int64:
				if v, ok := val.(int64); ok {
					*d = v
				}
			case *string:
				if v, ok := val.(string); ok {
					*d = v
				}
			case *time.Time:
				if v, ok := val.(time.Time); ok {
					*d = v
				}
			case *interface{}:
				switch val := val.(type) {
				case uint64:
					*d = val
				case string:
					*d = val
				case time.Time:
					*d = val
				case *uint64:
					*d = val
				}
			}
		}
	}
	return nil
}

func (m *MockRows) Err() error {
	return m.err
}

func (m *MockRows) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("")
}

func (m *MockRows) Values() ([]interface{}, error) {
	if m.pos >= len(m.rows) {
		return nil, nil
	}
	return m.rows[m.pos], nil
}

func (m *MockRows) RawValues() [][]byte {
	return nil
}

func (m *MockRows) Conn() *pgx.Conn {
	return nil
}

// MockRedis represents a mock Redis client.
type MockRedis struct {
	mock.Mock
}

func (m *MockRedis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	args := m.Called(ctx, key, value, expiration)

	if statusCmd, ok := args.Get(0).(*redis.StatusCmd); ok {
		return statusCmd
	}

	cmd := redis.NewStatusCmd(ctx)

	if len(args) > 0 {
		if args.Get(0) != nil {
			if err, ok := args.Get(0).(error); ok && err != nil {
				cmd.SetErr(err)
			} else {
				cmd.SetVal("OK")
			}
		} else {
			cmd.SetVal("OK")
		}
	} else {
		cmd.SetVal("OK")
	}

	return cmd
}

func (m *MockRedis) Get(ctx context.Context, key string) *redis.StringCmd {
	args := m.Called(ctx, key)

	if stringCmd, ok := args.Get(0).(*redis.StringCmd); ok {
		return stringCmd
	}

	cmd := redis.NewStringCmd(ctx)

	if len(args) > 0 && args.Get(0) != nil {
		if err, ok := args.Get(0).(error); ok {
			cmd.SetErr(err)
		} else if len(args) > 1 {
			if val, ok := args.Get(1).(string); ok && val != "" {
				cmd.SetVal(val)
			}
		}
	}

	return cmd
}

func (m *MockRedis) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	args := m.Called(ctx, keys)

	if intCmd, ok := args.Get(0).(*redis.IntCmd); ok {
		return intCmd
	}

	cmd := redis.NewIntCmd(ctx)

	if len(args) > 0 {
		if args.Get(0) != nil {
			if err, ok := args.Get(0).(error); ok && err != nil {
				cmd.SetErr(err)
			} else {
				cmd.SetVal(1)
			}
		}
	}

	return cmd
}

func (m *MockRedis) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockCommandTag represents a mock command tag.
type MockCommandTag struct {
	rowsAffected int64
}

func NewMockCommandTag(rowsAffected int64) pgconn.CommandTag {
	tag := fmt.Sprintf("DELETE %d", rowsAffected)
	return pgconn.NewCommandTag(tag)
}

func (m MockCommandTag) RowsAffected() int64 {
	return m.rowsAffected
}

func (m MockCommandTag) Insert() bool {
	return false
}

func (m MockCommandTag) Update() bool {
	return false
}

func (m MockCommandTag) Delete() bool {
	return false
}

func (m MockCommandTag) Select() bool {
	return false
}

func (m MockCommandTag) String() string {
	return ""
}

// Test helper functions.
func CreateTestDependencies(mockDB DBInterface, mockRedis RedisInterface) *Dependens {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	cfg := &config.Config{}
	cfg.Server.JWTSecret = "test-secret-key"
	cfg.Redis.AccessTokenTTL = time.Hour
	cfg.Redis.RefreshTokenTTL = time.Hour * 24

	return &Dependens{
		DB:     mockDB,
		Redis:  mockRedis,
		Logger: logger,
		Config: cfg,
	}
}

// Test data helpers.
func CreateTestEmployee() entity.Employee {
	id := uint64(1)
	email := "test@example.com"
	password := "hashedpassword"
	isActive := true
	departmentID := uint64(1)
	position := "Developer"
	vacationDays := uint64(25)
	sickDays := uint64(10)
	now := time.Now()

	return entity.Employee{
		ID:           &id,
		FirstName:    "John",
		LastName:     "Doe",
		Email:        &email,
		Password:     &password,
		Role:         "employee",
		IsActive:     &isActive,
		DepartmentID: &departmentID,
		Position:     &position,
		VacationDays: &vacationDays,
		SickDays:     &sickDays,
		Status:       "active",
		CreatedAt:    &now,
		UpdatedAt:    &now,
	}
}

func CreateTestDepartment() entity.Department {
	parentID := uint64(1)
	headID := uint64(1)

	return entity.Department{
		ID:          1,
		Name:        "Engineering",
		Description: "Software Engineering Department",
		ParentID:    &parentID,
		HeadID:      &headID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func StringPtr(s string) *string {
	return &s
}

func Uint64Ptr(u uint64) *uint64 {
	return &u
}

func BoolPtr(b bool) *bool {
	return &b
}

func TimePtr(t time.Time) *time.Time {
	return &t
}
