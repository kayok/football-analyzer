package domain

import (
	"strings"
	"testing"
)

func TestRegistrationPasswordPolicy(t *testing.T) {
	for _, password := range []string{"abcdefghi", "123456789", "Abc!@#$%^", "abcdefghij", "0123456789", "Abc!@#$%^&", strings.Repeat("A", 72), "Aa!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"} {
		if err := ValidatePassword(password); err != nil {
			t.Errorf("allowed password rejected: %v", err)
		}
	}
	for _, password := range []string{"", strings.Repeat("A", 8), strings.Repeat("A", 73), "ภาษาไทยทดสอบ", "abcdefghij😀", "abcd efghi", "abcdefghi\t", "abcdefghi\n", "abcdefghi\x00", "abcdefghi\x7f", "abcdefghijé", "abcdefghij\u200b"} {
		if err := ValidatePassword(password); err == nil {
			t.Error("unsupported or incorrectly sized password accepted")
		}
	}
}
