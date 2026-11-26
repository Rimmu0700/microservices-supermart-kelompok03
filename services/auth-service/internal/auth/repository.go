package auth

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id string) (*User, error)
	CreateUser(user *User, roleID string) error
	GetAllUsers() ([]User, error)
	UpdateUser(user *User) error
	AdminCreateUser(u *User, roleName string) error
	AdminUpdateUser(id, fullName, phoneNumber, roleName string, isActive *bool, newPasswordHash string) error
	ToggleUserStatus(id string, isActive bool) error
	DeleteUser(id string) error
	GetUserProfile(userID string) (*UserProfile, error)
	UpsertUserProfile(profile *UserProfile) error
	GetUserAddresses(userID string) ([]UserAddress, error)
	CreateUserAddress(addr *UserAddress) error
	DeleteUserAddress(id, userID string) error
	GetKYC(userID string) (*UserKYCDocument, error)
	CreateKYC(kyc *UserKYCDocument) error
	UpdateKYCStatus(id, status string) error
	GetRoles() ([]Role, error)
	GetRoleByName(name string) (*Role, error)
	GetPermissions() ([]Permission, error)
}

type mysqlRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &mysqlRepository{db: db}
}

func (r *mysqlRepository) GetUserByEmail(email string) (*User, error) {
	query := `
		SELECT u.id, u.email, u.password_hash, u.full_name, u.phone_number, u.is_active, u.created_at, u.updated_at,
		       COALESCE(r.role_name, 'CUSTOMER') AS role_name
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles r ON ur.role_id = r.id
		WHERE u.email = ? LIMIT 1
	`
	var u User
	err := r.db.Get(&u, query, email)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *mysqlRepository) GetUserByID(id string) (*User, error) {
	query := `
		SELECT u.id, u.email, u.password_hash, u.full_name, u.phone_number, u.is_active, u.created_at, u.updated_at,
		       COALESCE(r.role_name, 'CUSTOMER') AS role_name
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles r ON ur.role_id = r.id
		WHERE u.id = ? LIMIT 1
	`
	var u User
	err := r.db.Get(&u, query, id)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *mysqlRepository) CreateUser(u *User, roleID string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now

	insertUser := `
		INSERT INTO users (id, email, password_hash, full_name, phone_number, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = tx.Exec(insertUser, u.ID, u.Email, u.PasswordHash, u.FullName, u.PhoneNumber, u.IsActive, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	if roleID != "" {
		insertRole := `INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)`
		_, err = tx.Exec(insertRole, u.ID, roleID)
		if err != nil {
			return fmt.Errorf("insert user_role: %w", err)
		}
	}

	return tx.Commit()
}

func (r *mysqlRepository) GetAllUsers() ([]User, error) {
	query := `
		SELECT u.id, u.email, u.full_name, u.phone_number, u.is_active, u.created_at, u.updated_at,
		       COALESCE(r.role_name, 'CUSTOMER') AS role_name
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles r ON ur.role_id = r.id
		ORDER BY u.created_at DESC
	`
	var users []User
	err := r.db.Select(&users, query)
	return users, err
}

func (r *mysqlRepository) UpdateUser(u *User) error {
	query := `
		UPDATE users SET full_name = ?, phone_number = ?, updated_at = NOW()
		WHERE id = ?
	`
	_, err := r.db.Exec(query, u.FullName, u.PhoneNumber, u.ID)
	return err
}

func (r *mysqlRepository) GetUserProfile(userID string) (*UserProfile, error) {
	var profile UserProfile
	err := r.db.Get(&profile, "SELECT * FROM user_profiles WHERE user_id = ?", userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &profile, err
}

func (r *mysqlRepository) UpsertUserProfile(p *UserProfile) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	query := `
		INSERT INTO user_profiles (id, user_id, gender, birth_date, avatar_url, bio)
		VALUES (:id, :user_id, :gender, :birth_date, :avatar_url, :bio)
		ON DUPLICATE KEY UPDATE
		gender = VALUES(gender), birth_date = VALUES(birth_date),
		avatar_url = VALUES(avatar_url), bio = VALUES(bio)
	`
	_, err := r.db.NamedExec(query, p)
	return err
}

func (r *mysqlRepository) GetUserAddresses(userID string) ([]UserAddress, error) {
	var addresses []UserAddress
	err := r.db.Select(&addresses, "SELECT * FROM user_addresses WHERE user_id = ? ORDER BY is_primary DESC, id ASC", userID)
	return addresses, err
}

func (r *mysqlRepository) CreateUserAddress(addr *UserAddress) error {
	if addr.ID == "" {
		addr.ID = uuid.NewString()
	}
	query := `
		INSERT INTO user_addresses (id, user_id, address_label, recipient_name, phone_number, street_address, city, province, postal_code, latitude, longitude, is_primary)
		VALUES (:id, :user_id, :address_label, :recipient_name, :phone_number, :street_address, :city, :province, :postal_code, :latitude, :longitude, :is_primary)
	`
	_, err := r.db.NamedExec(query, addr)
	return err
}

func (r *mysqlRepository) DeleteUserAddress(id, userID string) error {
	_, err := r.db.Exec("DELETE FROM user_addresses WHERE id = ? AND user_id = ?", id, userID)
	return err
}

func (r *mysqlRepository) GetKYC(userID string) (*UserKYCDocument, error) {
	var kyc UserKYCDocument
	err := r.db.Get(&kyc, "SELECT * FROM user_kyc_documents WHERE user_id = ? ORDER BY id DESC LIMIT 1", userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &kyc, err
}

func (r *mysqlRepository) CreateKYC(kyc *UserKYCDocument) error {
	if kyc.ID == "" {
		kyc.ID = uuid.NewString()
	}
	kyc.VerificationStatus = "PENDING"
	query := `
		INSERT INTO user_kyc_documents (id, user_id, id_card_number, id_card_image_url, selfie_image_url, verification_status)
		VALUES (:id, :user_id, :id_card_number, :id_card_image_url, :selfie_image_url, :verification_status)
	`
	_, err := r.db.NamedExec(query, kyc)
	return err
}

func (r *mysqlRepository) UpdateKYCStatus(id, status string) error {
	now := time.Now()
	_, err := r.db.Exec("UPDATE user_kyc_documents SET verification_status = ?, verified_at = ? WHERE id = ?", status, now, id)
	return err
}

func (r *mysqlRepository) GetRoles() ([]Role, error) {
	var roles []Role
	err := r.db.Select(&roles, "SELECT * FROM roles")
	return roles, err
}

func (r *mysqlRepository) GetRoleByName(name string) (*Role, error) {
	var role Role
	err := r.db.Get(&role, "SELECT * FROM roles WHERE role_name = ? LIMIT 1", name)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *mysqlRepository) GetPermissions() ([]Permission, error) {
	var permissions []Permission
	err := r.db.Select(&permissions, "SELECT * FROM permissions")
	return permissions, err
}

func (r *mysqlRepository) AdminCreateUser(u *User, roleName string) error {
	var role Role
	if roleName == "" {
		roleName = "CUSTOMER"
	}
	err := r.db.Get(&role, "SELECT id, role_name FROM roles WHERE role_name = ? LIMIT 1", roleName)
	if err != nil {
		return fmt.Errorf("role '%s' tidak ditemukan: %w", roleName, err)
	}

	return r.CreateUser(u, role.ID)
}

func (r *mysqlRepository) AdminUpdateUser(id, fullName, phoneNumber, roleName string, isActive *bool, newPasswordHash string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if newPasswordHash != "" && isActive != nil {
		_, err = tx.Exec(`
			UPDATE users 
			SET full_name = ?, phone_number = ?, is_active = ?, password_hash = ?, updated_at = NOW() 
			WHERE id = ?
		`, fullName, phoneNumber, *isActive, newPasswordHash, id)
	} else if newPasswordHash != "" {
		_, err = tx.Exec(`
			UPDATE users 
			SET full_name = ?, phone_number = ?, password_hash = ?, updated_at = NOW() 
			WHERE id = ?
		`, fullName, phoneNumber, newPasswordHash, id)
	} else if isActive != nil {
		_, err = tx.Exec(`
			UPDATE users 
			SET full_name = ?, phone_number = ?, is_active = ?, updated_at = NOW() 
			WHERE id = ?
		`, fullName, phoneNumber, *isActive, id)
	} else {
		_, err = tx.Exec(`
			UPDATE users 
			SET full_name = ?, phone_number = ?, updated_at = NOW() 
			WHERE id = ?
		`, fullName, phoneNumber, id)
	}
	if err != nil {
		return fmt.Errorf("gagal update data user: %w", err)
	}

	if roleName != "" {
		var role Role
		err = tx.Get(&role, "SELECT id, role_name FROM roles WHERE role_name = ? LIMIT 1", roleName)
		if err != nil {
			return fmt.Errorf("role '%s' tidak ditemukan: %w", roleName, err)
		}
		_, _ = tx.Exec("DELETE FROM user_roles WHERE user_id = ?", id)
		_, err = tx.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", id, role.ID)
		if err != nil {
			return fmt.Errorf("gagal update user_roles: %w", err)
		}
	}

	return tx.Commit()
}

func (r *mysqlRepository) ToggleUserStatus(id string, isActive bool) error {
	_, err := r.db.Exec("UPDATE users SET is_active = ?, updated_at = NOW() WHERE id = ?", isActive, id)
	return err
}

func (r *mysqlRepository) DeleteUser(id string) error {
	// Attempt delete; if foreign key blocks, fallback to deactivating
	_, err := r.db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		_, err = r.db.Exec("UPDATE users SET is_active = 0, updated_at = NOW() WHERE id = ?", id)
	}
	return err
}

