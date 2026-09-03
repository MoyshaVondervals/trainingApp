package user

import (
	"strings"
	"testing"
)

func TestUserValidate(t *testing.T) {
	valid := User{Name: "Иван", SecondName: "Иванов", Email: "ivan@example.com"}

	tests := []struct {
		name    string
		mutate  func(u *User)
		wantErr bool
	}{
		{name: "корректный пользователь", mutate: func(*User) {}},
		{name: "пустое имя", mutate: func(u *User) { u.Name = "" }, wantErr: true},
		{name: "имя из пробелов", mutate: func(u *User) { u.Name = "  " }, wantErr: true},
		{
			name:    "имя длиннее предела",
			mutate:  func(u *User) { u.Name = strings.Repeat("я", maxUserNameLen+1) },
			wantErr: true,
		},
		{name: "пустая фамилия", mutate: func(u *User) { u.SecondName = "" }, wantErr: true},
		{name: "email без собаки", mutate: func(u *User) { u.Email = "ivan.example.com" }, wantErr: true},
		{name: "пустой email", mutate: func(u *User) { u.Email = "" }, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := valid
			tt.mutate(&u)
			err := u.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "минимальная длина", password: strings.Repeat("a", minPasswordLen)},
		{name: "предельная длина", password: strings.Repeat("a", maxPasswordLen)},
		{name: "короче предела", password: strings.Repeat("a", minPasswordLen-1), wantErr: true},
		{name: "длиннее предела", password: strings.Repeat("a", maxPasswordLen+1), wantErr: true},
		{name: "пустой", password: "", wantErr: true},
		{name: "только пробелы", password: "         ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePassword() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePasswordCountsBytesNotRunes(t *testing.T) {
	if err := ValidatePassword(strings.Repeat("я", 40)); err == nil {
		t.Fatal("40 кириллических символов дают 80 байт и должны превышать предел bcrypt в 72 байта")
	}
}
