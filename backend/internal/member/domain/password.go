package domain

import "errors"

// New accounts use visible ASCII characters, so character and byte lengths agree.
// Existing passwords remain valid at login; this policy applies to registration only.
func ValidatePassword(password string) error {
	if len(password) < 9 || len(password) > 72 {
		return errors.New("รหัสผ่านต้องมี 9–72 ตัวอักษร")
	}
	for _, ch := range password {
		if ch < '!' || ch > '~' {
			return errors.New("รหัสผ่านใช้ได้เฉพาะภาษาอังกฤษ ตัวเลข และอักขระพิเศษทั่วไป โดยไม่มีช่องว่าง")
		}
	}
	return nil
}
