package main

import "testing"

func TestAuthErrorClassification(t *testing.T) {
	cases := []struct {
		name        string
		err         error
		unconfirmed bool
		invalid     bool
	}{
		{
			name:        "email not confirmed",
			err:         &apiError{Status: 400, Code: "email_not_confirmed", Message: "Email not confirmed"},
			unconfirmed: true,
		},
		{
			name:        "email not confirmed by message only",
			err:         &apiError{Status: 400, Message: "Email not confirmed"},
			unconfirmed: true,
		},
		{
			name:    "invalid credentials",
			err:     &apiError{Status: 400, Code: "invalid_credentials", Message: "Invalid login credentials"},
			invalid: true,
		},
		{
			name: "unrelated error",
			err:  &apiError{Status: 500, Message: "boom"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isEmailNotConfirmed(tc.err); got != tc.unconfirmed {
				t.Errorf("isEmailNotConfirmed = %v, want %v", got, tc.unconfirmed)
			}
			if got := isInvalidCredentials(tc.err); got != tc.invalid {
				t.Errorf("isInvalidCredentials = %v, want %v", got, tc.invalid)
			}
		})
	}
}
