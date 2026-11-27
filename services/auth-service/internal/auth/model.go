package auth

import (
	"time"
)

type User struct {
	ID           string    `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	FullName     string    `db:"full_name" json:"full_name"`
	PhoneNumber  *string   `db:"phone_number" json:"phone_number"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`

	// Joined fields
	RoleName string `db:"role_name" json:"role,omitempty"`
}

type Role struct {
	ID          string  `db:"id" json:"id"`
	RoleName    string  `db:"role_name" json:"role_name"`
	Description *string `db:"description" json:"description"`
}

type Permission struct {
	ID            string  `db:"id" json:"id"`
	PermissionKey string  `db:"permission_key" json:"permission_key"`
	Description   *string `db:"description" json:"description"`
}

type UserProfile struct {
	ID        string     `db:"id" json:"id"`
	UserID    string     `db:"user_id" json:"user_id"`
	Gender    *string    `db:"gender" json:"gender"`
	BirthDate *time.Time `db:"birth_date" json:"birth_date"`
	AvatarURL *string    `db:"avatar_url" json:"avatar_url"`
	Bio       *string    `db:"bio" json:"bio"`
}

type UserAddress struct {
	ID            string   `db:"id" json:"id"`
	UserID        string   `db:"user_id" json:"user_id"`
	AddressLabel  string   `db:"address_label" json:"address_label"`
	RecipientName string   `db:"recipient_name" json:"recipient_name"`
	PhoneNumber   string   `db:"phone_number" json:"phone_number"`
	StreetAddress string   `db:"street_address" json:"street_address"`
	City          string   `db:"city" json:"city"`
	Province      string   `db:"province" json:"province"`
	PostalCode    string   `db:"postal_code" json:"postal_code"`
	Latitude      *float64 `db:"latitude" json:"latitude"`
	Longitude     *float64 `db:"longitude" json:"longitude"`
	IsPrimary     bool     `db:"is_primary" json:"is_primary"`
}

type UserKYCDocument struct {
	ID                 string     `db:"id" json:"id"`
	UserID             string     `db:"user_id" json:"user_id"`
	IDCardNumber       string     `db:"id_card_number" json:"id_card_number"`
	IDCardImageURL     string     `db:"id_card_image_url" json:"id_card_image_url"`
	SelfieImageURL     string     `db:"selfie_image_url" json:"selfie_image_url"`
	VerificationStatus string     `db:"verification_status" json:"verification_status"`
	VerifiedAt         *time.Time `db:"verified_at" json:"verified_at"`
}
