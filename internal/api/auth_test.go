package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"trainingApp/internal/auth"
)

var testSecret = []byte("test-secret-at-least-32-bytes-long!!")

func TestParseToken(t *testing.T) {
	token, _, err := auth.NewToken(testSecret, 42, time.Hour)
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}

	t.Run("действующий токен", func(t *testing.T) {
		userID, err := ParseToken(testSecret, token)
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if userID != 42 {
			t.Fatalf("userID = %d, ожидалось 42", userID)
		}
	})

	t.Run("чужая подпись", func(t *testing.T) {
		if _, err := ParseToken([]byte("another-secret-at-least-32-bytes!!!!"), token); err == nil {
			t.Fatal("токен, подписанный другим ключом, должен отклоняться")
		}
	})

	t.Run("истёкший токен", func(t *testing.T) {
		expired, _, err := auth.NewToken(testSecret, 42, -time.Minute)
		if err != nil {
			t.Fatalf("NewToken: %v", err)
		}
		if _, err := ParseToken(testSecret, expired); err == nil {
			t.Fatal("истёкший токен должен отклоняться")
		}
	})

	t.Run("испорченная строка", func(t *testing.T) {
		if _, err := ParseToken(testSecret, "not-a-jwt"); err == nil {
			t.Fatal("строка без структуры JWT должна отклоняться")
		}
	})
}

func TestRequireAuth(t *testing.T) {
	handler := NewUserHandler(nil, testSecret, time.Hour)

	var seenUserID int64
	var reached bool
	protected := handler.RequireAuth(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		reached = true
		seenUserID, _ = userIDFrom(r.Context())
	}))

	token, _, err := auth.NewToken(testSecret, 7, time.Hour)
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}

	tests := []struct {
		name       string
		header     string
		wantStatus int
		wantUserID int64
	}{
		{name: "корректный токен", header: "Bearer " + token, wantStatus: http.StatusOK, wantUserID: 7},
		{name: "заголовок отсутствует", header: "", wantStatus: http.StatusUnauthorized},
		{name: "без префикса Bearer", header: token, wantStatus: http.StatusUnauthorized},
		{name: "пустой токен после префикса", header: "Bearer ", wantStatus: http.StatusUnauthorized},
		{name: "мусор вместо токена", header: "Bearer garbage", wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reached, seenUserID = false, 0

			req := httptest.NewRequest(http.MethodGet, "/api/v1/workouts", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			protected.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("статус = %d, ожидался %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusOK {
				if !reached {
					t.Fatal("следующий обработчик не был вызван")
				}
				if seenUserID != tt.wantUserID {
					t.Fatalf("userID в контексте = %d, ожидался %d", seenUserID, tt.wantUserID)
				}
			} else if reached {
				t.Fatal("следующий обработчик не должен вызываться при отказе")
			}
		})
	}
}

func TestUserIDFromEmptyContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if _, ok := userIDFrom(req.Context()); ok {
		t.Fatal("в контексте без авторизации userID быть не должно")
	}
}
