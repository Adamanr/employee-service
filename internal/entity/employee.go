package entity

import (
	"time"
)

type Employee struct {
	ID             uint   `json:"id"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	MiddleName     string `json:"middle_name"`
	Phone          string `json:"phone"`
	PersonalNumber string `json:"personal_number"`

	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	Role         string `json:"role" `
	IsActive     bool   `json:"is_active"`

	DepartmentID uint   `json:"department_id"`
	Position     string `json:"position"`
	ManagerID    *uint  `json:"manager_id"`

	HireDate time.Time  `json:"hire_date"`
	FireDate *time.Time `json:"fire_date"`

	Birthday *time.Time `json:"birthday"`
	Address  string     `json:"address"`

	VacationDays int `json:"vacation_days"`
	SickDays     int `json:"sick_days"`

	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type VacationRequest struct {
	Days int `json:"days"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
