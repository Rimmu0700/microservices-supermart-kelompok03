package auth

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	FullName    string `json:"full_name"`
	PhoneNumber string `json:"phone_number"`
}

type UpdateProfileRequest struct {
	FullName    string  `json:"full_name"`
	PhoneNumber *string `json:"phone_number"`
	Gender      *string `json:"gender"`
	Bio         *string `json:"bio"`
	AvatarURL   *string `json:"avatar_url"`
}

type AddressRequest struct {
	AddressLabel  string   `json:"address_label"`
	RecipientName string   `json:"recipient_name"`
	PhoneNumber   string   `json:"phone_number"`
	StreetAddress string   `json:"street_address"`
	City          string   `json:"city"`
	Province      string   `json:"province"`
	PostalCode    string   `json:"postal_code"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	IsPrimary     bool     `json:"is_primary"`
}

type KYCRequest struct {
	IDCardNumber   string `json:"id_card_number"`
	IDCardImageURL string `json:"id_card_image_url"`
	SelfieImageURL string `json:"selfie_image_url"`
}

type VerifyKYCRequest struct {
	Status string `json:"status"` // VERIFIED or REJECTED
}

type AdminCreateUserRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	FullName    string `json:"full_name"`
	PhoneNumber string `json:"phone_number"`
	Role        string `json:"role"` // WAREHOUSE_STAFF, COURIER, CS_AGENT, CUSTOMER, SUPER_ADMIN
	IsActive    *bool  `json:"is_active"`
}

type AdminUpdateUserRequest struct {
	FullName    string `json:"full_name"`
	PhoneNumber string `json:"phone_number"`
	Role        string `json:"role"`
	IsActive    *bool  `json:"is_active"`
	Password    string `json:"password,omitempty"`
}

type ToggleStatusRequest struct {
	IsActive bool `json:"is_active"`
}

