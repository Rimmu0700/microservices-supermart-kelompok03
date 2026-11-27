package auth

import (
	"errors"

	"github.com/google/uuid"
	"github.com/nusantara-supermart/backend/pkg/config"
	"github.com/nusantara-supermart/backend/pkg/middleware"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Login(req LoginRequest) (*LoginResponse, error)
	Register(req RegisterRequest) (*LoginResponse, error)
	GetMe(userID string) (*User, *UserProfile, error)
	UpdateProfile(userID string, req UpdateProfileRequest) error
	GetAddresses(userID string) ([]UserAddress, error)
	AddAddress(userID string, req AddressRequest) error
	DeleteAddress(id, userID string) error
	SubmitKYC(userID string, req KYCRequest) error
	GetKYC(userID string) (*UserKYCDocument, error)
	VerifyKYC(kycID, status string) error
	GetAllUsers() ([]User, error)
	CreateUserByAdmin(req AdminCreateUserRequest) (*User, error)
	UpdateUserByAdmin(id string, req AdminUpdateUserRequest) error
	ToggleUserStatus(id string, isActive bool) error
	DeleteUser(id string) error
	GetRoles() ([]Role, error)
	GetPermissions() ([]Permission, error)
}

type authService struct {
	repo Repository
	cfg  *config.Config
}

func NewService(repo Repository, cfg *config.Config) Service {
	return &authService{repo: repo, cfg: cfg}
}

func (s *authService) Login(req LoginRequest) (*LoginResponse, error) {
	user, err := s.repo.GetUserByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	if !user.IsActive {
		return nil, errors.New("user account is deactivated")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	role := user.RoleName
	if role == "" {
		role = "CUSTOMER"
	}

	token, err := middleware.GenerateToken(user.ID, user.Email, role, s.cfg)
	if err != nil {
		return nil, errors.New("failed to generate auth token")
	}

	return &LoginResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *authService) Register(req RegisterRequest) (*LoginResponse, error) {
	existing, _ := s.repo.GetUserByEmail(req.Email)
	if existing != nil {
		return nil, errors.New("email is already registered")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
	if err != nil {
		return nil, errors.New("failed to process password")
	}

	custRole, err := s.repo.GetRoleByName("CUSTOMER")
	var roleID string
	if err == nil && custRole != nil {
		roleID = custRole.ID
	}

	newUser := &User{
		ID:           uuid.NewString(),
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		FullName:     req.FullName,
		PhoneNumber:  &req.PhoneNumber,
		IsActive:     true,
		RoleName:     "CUSTOMER",
	}

	err = s.repo.CreateUser(newUser, roleID)
	if err != nil {
		return nil, err
	}

	token, err := middleware.GenerateToken(newUser.ID, newUser.Email, "CUSTOMER", s.cfg)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	return &LoginResponse{
		Token: token,
		User:  *newUser,
	}, nil
}

func (s *authService) GetMe(userID string) (*User, *UserProfile, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, nil, err
	}
	profile, _ := s.repo.GetUserProfile(userID)
	return user, profile, nil
}

func (s *authService) UpdateProfile(userID string, req UpdateProfileRequest) error {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return err
	}
	user.FullName = req.FullName
	if req.PhoneNumber != nil {
		user.PhoneNumber = req.PhoneNumber
	}
	if err := s.repo.UpdateUser(user); err != nil {
		return err
	}

	profile := &UserProfile{
		UserID:    userID,
		Gender:    req.Gender,
		Bio:       req.Bio,
		AvatarURL: req.AvatarURL,
	}
	return s.repo.UpsertUserProfile(profile)
}

func (s *authService) GetAddresses(userID string) ([]UserAddress, error) {
	return s.repo.GetUserAddresses(userID)
}

func (s *authService) AddAddress(userID string, req AddressRequest) error {
	addr := &UserAddress{
		UserID:        userID,
		AddressLabel:  req.AddressLabel,
		RecipientName: req.RecipientName,
		PhoneNumber:   req.PhoneNumber,
		StreetAddress: req.StreetAddress,
		City:          req.City,
		Province:      req.Province,
		PostalCode:    req.PostalCode,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		IsPrimary:     req.IsPrimary,
	}
	return s.repo.CreateUserAddress(addr)
}

func (s *authService) DeleteAddress(id, userID string) error {
	return s.repo.DeleteUserAddress(id, userID)
}

func (s *authService) SubmitKYC(userID string, req KYCRequest) error {
	kyc := &UserKYCDocument{
		UserID:         userID,
		IDCardNumber:   req.IDCardNumber,
		IDCardImageURL: req.IDCardImageURL,
		SelfieImageURL: req.SelfieImageURL,
	}
	return s.repo.CreateKYC(kyc)
}

func (s *authService) GetKYC(userID string) (*UserKYCDocument, error) {
	return s.repo.GetKYC(userID)
}

func (s *authService) VerifyKYC(kycID, status string) error {
	return s.repo.UpdateKYCStatus(kycID, status)
}

func (s *authService) GetAllUsers() ([]User, error) {
	return s.repo.GetAllUsers()
}

func (s *authService) GetRoles() ([]Role, error) {
	return s.repo.GetRoles()
}

func (s *authService) GetPermissions() ([]Permission, error) {
	return s.repo.GetPermissions()
}

func (s *authService) CreateUserByAdmin(req AdminCreateUserRequest) (*User, error) {
	if req.Email == "" || req.Password == "" || req.FullName == "" {
		return nil, errors.New("email, password, dan nama lengkap wajib diisi")
	}

	existing, _ := s.repo.GetUserByEmail(req.Email)
	if existing != nil {
		return nil, errors.New("email sudah terdaftar di sistem")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("gagal mengenkripsi password")
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	var phone *string
	if req.PhoneNumber != "" {
		phone = &req.PhoneNumber
	}

	user := &User{
		ID:           uuid.NewString(),
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		FullName:     req.FullName,
		PhoneNumber:  phone,
		IsActive:     isActive,
	}

	roleName := req.Role
	if roleName == "" {
		roleName = "CUSTOMER"
	}

	if err := s.repo.AdminCreateUser(user, roleName); err != nil {
		return nil, err
	}

	user.RoleName = roleName
	return user, nil
}

func (s *authService) UpdateUserByAdmin(id string, req AdminUpdateUserRequest) error {
	existing, err := s.repo.GetUserByID(id)
	if err != nil || existing == nil {
		return errors.New("user tidak ditemukan")
	}

	var newPasswordHash string
	if req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return errors.New("gagal mengenkripsi password baru")
		}
		newPasswordHash = string(hashed)
	}

	fullName := req.FullName
	if fullName == "" {
		fullName = existing.FullName
	}

	return s.repo.AdminUpdateUser(id, fullName, req.PhoneNumber, req.Role, req.IsActive, newPasswordHash)
}

func (s *authService) ToggleUserStatus(id string, isActive bool) error {
	return s.repo.ToggleUserStatus(id, isActive)
}

func (s *authService) DeleteUser(id string) error {
	return s.repo.DeleteUser(id)
}

