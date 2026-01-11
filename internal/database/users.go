package database

import "time"

// User represents a user account
type User struct {
	ID                 int64     `json:"id"`
	Username           string    `json:"username"`
	PasswordHash       string    `json:"-"` // Never expose in JSON
	Role               string    `json:"role"` // admin, user, kid
	ContentRatingLimit *string   `json:"contentRatingLimit,omitempty"` // G, PG, PG-13, R, NC-17, or nil (no limit)
	PinHash            *string   `json:"-"`                            // PIN hash, never expose
	RequirePin         bool      `json:"requirePin"`                   // Require PIN for elevated content
	CreatedAt          time.Time `json:"createdAt"`
}

// Profile represents a viewing profile within a user account (Netflix-style)
type Profile struct {
	ID                 int64     `json:"id"`
	UserID             int64     `json:"userId"`
	Name               string    `json:"name"`
	AvatarURL          *string   `json:"avatarUrl,omitempty"`
	IsDefault          bool      `json:"isDefault"`
	IsKid              bool      `json:"isKid"`
	ContentRatingLimit *string   `json:"contentRatingLimit,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
}

// ContentRatingLevel returns the numeric level for a content rating (for comparison)
func ContentRatingLevel(rating string) int {
	switch rating {
	case "G":
		return 1
	case "PG":
		return 2
	case "PG-13":
		return 3
	case "R":
		return 4
	case "NC-17":
		return 5
	default:
		return 0 // Unknown or unrated
	}
}

// NormalizeContentRating converts various content rating formats to US MPAA ratings
func NormalizeContentRating(rating string, country string) string {
	if rating == "" {
		return ""
	}

	// Already normalized US ratings
	switch rating {
	case "G", "PG", "PG-13", "R", "NC-17":
		return rating
	}

	// US TV ratings
	switch rating {
	case "TV-Y", "TV-Y7", "TV-G":
		return "G"
	case "TV-PG":
		return "PG"
	case "TV-14":
		return "PG-13"
	case "TV-MA":
		return "R"
	}

	// UK ratings (BBFC)
	switch rating {
	case "U", "Uc":
		return "G"
	case "12", "12A":
		return "PG-13"
	case "15":
		return "R"
	case "18", "R18":
		return "NC-17"
	}

	// Australia ratings
	switch rating {
	case "E":
		return "G"
	case "M":
		return "PG"
	case "MA", "MA15+", "M15+":
		return "PG-13"
	case "R18+":
		return "R"
	case "X", "X18+":
		return "NC-17"
	}

	// Germany ratings (FSK)
	switch rating {
	case "FSK 0":
		return "G"
	case "FSK 6":
		return "PG"
	case "FSK 12":
		return "PG-13"
	case "FSK 16":
		return "R"
	case "FSK 18":
		return "NC-17"
	}

	// Canada ratings
	switch rating {
	case "14A", "14+":
		return "PG-13"
	case "18A", "18+":
		return "R"
	case "A":
		return "NC-17"
	}

	// Default: try to match common patterns
	ratingUpper := rating
	if len(rating) > 0 {
		// Handle numeric ratings
		switch rating {
		case "0", "6":
			return "G"
		case "7", "10":
			return "PG"
		case "12", "13":
			return "PG-13"
		case "16", "17":
			return "R"
		case "18", "21":
			return "NC-17"
		}
	}

	// If we can't determine, return as-is (will show as "Unrated" in UI)
	return ratingUpper
}

// Session represents an active user session
type Session struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"userId"`
	Token           string    `json:"token"`
	ExpiresAt       time.Time `json:"expiresAt"`
	ActiveProfileID *int64    `json:"activeProfileId,omitempty"`
}

// PinElevation represents a temporary elevated access session after PIN verification
type PinElevation struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// User operations

func (d *Database) CreateUser(user *User) error {
	result, err := d.db.Exec(
		"INSERT INTO users (username, password_hash, role, content_rating_limit, pin_hash, require_pin) VALUES (?, ?, ?, ?, ?, ?)",
		user.Username, user.PasswordHash, user.Role, user.ContentRatingLimit, user.PinHash, user.RequirePin,
	)
	if err != nil {
		return err
	}
	user.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetUserByUsername(username string) (*User, error) {
	var u User
	var requirePin int
	err := d.db.QueryRow(
		"SELECT id, username, password_hash, role, content_rating_limit, pin_hash, require_pin, created_at FROM users WHERE username = ?", username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.ContentRatingLimit, &u.PinHash, &requirePin, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.RequirePin = requirePin == 1
	return &u, nil
}

func (d *Database) GetUserByID(id int64) (*User, error) {
	var u User
	var requirePin int
	err := d.db.QueryRow(
		"SELECT id, username, password_hash, role, content_rating_limit, pin_hash, require_pin, created_at FROM users WHERE id = ?", id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.ContentRatingLimit, &u.PinHash, &requirePin, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.RequirePin = requirePin == 1
	return &u, nil
}

func (d *Database) GetUsers() ([]User, error) {
	rows, err := d.db.Query("SELECT id, username, password_hash, role, content_rating_limit, pin_hash, require_pin, created_at FROM users ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var requirePin int
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.ContentRatingLimit, &u.PinHash, &requirePin, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.RequirePin = requirePin == 1
		users = append(users, u)
	}
	return users, nil
}

func (d *Database) UpdateUser(user *User) error {
	_, err := d.db.Exec(
		"UPDATE users SET username = ?, role = ?, content_rating_limit = ?, require_pin = ? WHERE id = ?",
		user.Username, user.Role, user.ContentRatingLimit, user.RequirePin, user.ID,
	)
	return err
}

func (d *Database) UpdateUserPin(id int64, pinHash *string) error {
	_, err := d.db.Exec("UPDATE users SET pin_hash = ? WHERE id = ?", pinHash, id)
	return err
}

func (d *Database) UpdateUserPassword(id int64, passwordHash string) error {
	_, err := d.db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", passwordHash, id)
	return err
}

func (d *Database) DeleteUser(id int64) error {
	_, err := d.db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

func (d *Database) CountUsers() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

// Profile operations

func (d *Database) CreateProfile(profile *Profile) error {
	result, err := d.db.Exec(
		`INSERT INTO profiles (user_id, name, avatar_url, is_default, is_kid, content_rating_limit)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		profile.UserID, profile.Name, profile.AvatarURL, profile.IsDefault, profile.IsKid, profile.ContentRatingLimit,
	)
	if err != nil {
		return err
	}
	profile.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetProfile(id int64) (*Profile, error) {
	var p Profile
	var isDefault, isKid int
	err := d.db.QueryRow(
		`SELECT id, user_id, name, avatar_url, is_default, is_kid, content_rating_limit, created_at
		 FROM profiles WHERE id = ?`, id,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.AvatarURL, &isDefault, &isKid, &p.ContentRatingLimit, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	p.IsDefault = isDefault == 1
	p.IsKid = isKid == 1
	return &p, nil
}

func (d *Database) GetProfilesByUser(userID int64) ([]Profile, error) {
	rows, err := d.db.Query(
		`SELECT id, user_id, name, avatar_url, is_default, is_kid, content_rating_limit, created_at
		 FROM profiles WHERE user_id = ? ORDER BY is_default DESC, created_at ASC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var p Profile
		var isDefault, isKid int
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.AvatarURL, &isDefault, &isKid, &p.ContentRatingLimit, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.IsDefault = isDefault == 1
		p.IsKid = isKid == 1
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (d *Database) GetDefaultProfile(userID int64) (*Profile, error) {
	var p Profile
	var isDefault, isKid int
	err := d.db.QueryRow(
		`SELECT id, user_id, name, avatar_url, is_default, is_kid, content_rating_limit, created_at
		 FROM profiles WHERE user_id = ? AND is_default = 1`, userID,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.AvatarURL, &isDefault, &isKid, &p.ContentRatingLimit, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	p.IsDefault = isDefault == 1
	p.IsKid = isKid == 1
	return &p, nil
}

func (d *Database) UpdateProfile(profile *Profile) error {
	_, err := d.db.Exec(
		`UPDATE profiles SET name = ?, avatar_url = ?, is_kid = ?, content_rating_limit = ?
		 WHERE id = ?`,
		profile.Name, profile.AvatarURL, profile.IsKid, profile.ContentRatingLimit, profile.ID,
	)
	return err
}

func (d *Database) DeleteProfile(id int64) error {
	_, err := d.db.Exec("DELETE FROM profiles WHERE id = ?", id)
	return err
}

func (d *Database) CountProfilesByUser(userID int64) (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM profiles WHERE user_id = ?", userID).Scan(&count)
	return count, err
}

func (d *Database) CreateDefaultProfileForUser(userID int64, username string) (*Profile, error) {
	profile := &Profile{
		UserID:    userID,
		Name:      username,
		IsDefault: true,
		IsKid:     false,
	}
	if err := d.CreateProfile(profile); err != nil {
		return nil, err
	}
	return profile, nil
}

// Session operations

func (d *Database) CreateSession(session *Session) error {
	result, err := d.db.Exec(
		"INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)",
		session.UserID, session.Token, session.ExpiresAt,
	)
	if err != nil {
		return err
	}
	session.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetSessionByToken(token string) (*Session, error) {
	var s Session
	err := d.db.QueryRow(
		"SELECT id, user_id, token, expires_at, active_profile_id FROM sessions WHERE token = ?", token,
	).Scan(&s.ID, &s.UserID, &s.Token, &s.ExpiresAt, &s.ActiveProfileID)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *Database) SetActiveProfile(token string, profileID int64) error {
	_, err := d.db.Exec("UPDATE sessions SET active_profile_id = ? WHERE token = ?", profileID, token)
	return err
}

func (d *Database) DeleteSession(token string) error {
	_, err := d.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}

func (d *Database) DeleteExpiredSessions() error {
	_, err := d.db.Exec("DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP")
	return err
}

func (d *Database) DeleteUserSessions(userID int64) error {
	_, err := d.db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	return err
}

// PIN elevation operations

func (d *Database) CreatePinElevation(elevation *PinElevation) error {
	result, err := d.db.Exec(
		"INSERT INTO pin_elevations (user_id, token, expires_at) VALUES (?, ?, ?)",
		elevation.UserID, elevation.Token, elevation.ExpiresAt,
	)
	if err != nil {
		return err
	}
	elevation.ID, _ = result.LastInsertId()
	return nil
}

func (d *Database) GetPinElevationByToken(token string) (*PinElevation, error) {
	var e PinElevation
	err := d.db.QueryRow(
		"SELECT id, user_id, token, expires_at FROM pin_elevations WHERE token = ? AND expires_at > CURRENT_TIMESTAMP", token,
	).Scan(&e.ID, &e.UserID, &e.Token, &e.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (d *Database) DeletePinElevation(token string) error {
	_, err := d.db.Exec("DELETE FROM pin_elevations WHERE token = ?", token)
	return err
}

func (d *Database) DeleteExpiredPinElevations() error {
	_, err := d.db.Exec("DELETE FROM pin_elevations WHERE expires_at < CURRENT_TIMESTAMP")
	return err
}

func (d *Database) DeleteUserPinElevations(userID int64) error {
	_, err := d.db.Exec("DELETE FROM pin_elevations WHERE user_id = ?", userID)
	return err
}
