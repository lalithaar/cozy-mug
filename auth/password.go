package auth

// ChangePassword verifies currentPassword against currentHash and, if it
// matches, returns a hash for newPassword. It is intentionally password-only:
// account lookup, session updates, and persistence remain app concerns.
func ChangePassword(currentHash, currentPassword, newPassword string) (string, error) {
	if !CheckPassword(currentHash, currentPassword) {
		return "", ErrInvalidPassword
	}
	return HashPassword(newPassword)
}
