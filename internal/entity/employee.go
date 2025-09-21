package entity

import (
	"time"
)

type Employee struct {
	ID             uint   `json:"id" gorm:"primaryKey"`
	FirstName      string `json:"first_name" gorm:"not null"`
	LastName       string `json:"last_name" gorm:"not null"`
	MiddleName     string `json:"middle_name"`
	Phone          string `json:"phone"`
	PersonalNumber string `json:"personal_number" gorm:"uniqueIndex"`

	Email        string `json:"email" gorm:"uniqueIndex"`
	PasswordHash string `json:"-"`
	Role         string `json:"role" gorm:"not null"`
	IsActive     bool   `json:"is_active" gorm:"default:true"`

	DepartmentID uint   `json:"department_id"`
	Position     string `json:"position"`
	ManagerID    *uint  `json:"manager_id"`

	HireDate time.Time  `json:"hire_date"`
	FireDate *time.Time `json:"fire_date"`

	Birthday *time.Time `json:"birthday"`
	Address  string     `json:"address"`

	VacationDays int `json:"vacation_days" gorm:"default:28"`
	SickDays     int `json:"sick_days" gorm:"default:0"`

	Status    string    `json:"status" gorm:"default:active"`
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
