package entity

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Employee struct {
	Address        *string    `json:"address"`
	Birthday       *time.Time `json:"birthday"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
	DepartmentID   *uint64    `json:"department_id"`
	Email          *string    `json:"email"`
	FireDate       *time.Time `json:"fire_date"`
	FirstName      string     `json:"first_name"`
	HireDate       *time.Time `json:"hire_date"`
	ID             *uint64    `json:"id,omitempty"`
	IsActive       *bool      `json:"is_active,omitempty"`
	LastName       string     `json:"last_name"`
	ManagerID      *uint64    `json:"manager_id"`
	MiddleName     *string    `json:"middle_name"`
	Password       *string    `json:"password"`
	PersonalNumber *string    `json:"personal_number"`
	Phone          *string    `json:"phone"`
	Position       *string    `json:"position"`
	Role           string     `json:"role"`
	SickDays       *uint64    `json:"sick_days,omitempty"`
	Status         string     `json:"status"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
	VacationDays   *uint64    `json:"vacation_days,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type VacationRequest struct {
	Days uint64 `json:"days"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Claims struct {
	jwt.RegisteredClaims

	ID      uint64 `json:"id"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	TokenID string `json:"token_id"`
}

type GetEmployeesParams struct {
	Role         *string `json:"role,omitempty"`
	DepartmentID *uint64 `json:"department_id,omitempty"`
	Status       *string `json:"status,omitempty"`
}
