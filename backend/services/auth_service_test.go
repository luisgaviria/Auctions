package services

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"regexp"
	"testing"
	"time"

	"backendAuction/models"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type validBcryptHash struct{}

func (validBcryptHash) Match(value driver.Value) bool {
	hash, ok := value.(string)
	return ok && bcrypt.CompareHashAndPassword([]byte(hash), []byte("secret-password")) == nil
}

func TestAuthServiceSignUp(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectExec(regexp.QuoteMeta(insertIntoUserTable)).
		WithArgs("person@example.com", validBcryptHash{}).
		WillReturnResult(sqlmock.NewResult(1, 1))

	data, status, err := NewAuthService(db).SignUp(&models.Credentials{Email: "person@example.com", Password: "secret-password"})
	if err != nil || status != http.StatusOK {
		t.Fatalf("SignUp() = status %d, err %v", status, err)
	}
	var response SignUpResponse
	if err := json.Unmarshal(data, &response); err != nil || response.Message != "Successfully registered user" {
		t.Errorf("unexpected signup response %q, err %v", data, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthServiceSignUpDatabaseError(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectExec(regexp.QuoteMeta(insertIntoUserTable)).WillReturnError(sql.ErrConnDone)
	data, status, err := NewAuthService(db).SignUp(&models.Credentials{Email: "person@example.com", Password: "secret-password"})
	if status != http.StatusInternalServerError || err == nil || data != nil {
		t.Errorf("SignUp() = (%q, %d, %v), want nil/500/error", data, status, err)
	}
}

func TestAuthServiceLogin(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	hash, err := bcrypt.GenerateFromPassword([]byte("secret-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid credentials", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(selectFromUserTable)).WithArgs("person@example.com").
			WillReturnRows(sqlmock.NewRows([]string{"email", "password"}).AddRow("person@example.com", string(hash)))
		data, status, err := NewAuthService(db).Login(&models.Credentials{Email: "person@example.com", Password: "secret-password"})
		if err != nil || status != http.StatusOK {
			t.Fatalf("Login() = status %d, err %v", status, err)
		}
		var response LoginResponse
		if err := json.Unmarshal(data, &response); err != nil {
			t.Fatal(err)
		}
		parsed, err := jwt.Parse(response.JwtToken, func(token *jwt.Token) (interface{}, error) {
			return []byte("test-secret"), nil
		})
		if err != nil || !parsed.Valid {
			t.Fatalf("returned token invalid: %v", err)
		}
		claims := parsed.Claims.(jwt.MapClaims)
		if claims["sub"] != "person@example.com" {
			t.Errorf("token subject = %v", claims["sub"])
		}
		if exp, ok := claims["exp"].(float64); !ok || time.Unix(int64(exp), 0).Before(time.Now()) {
			t.Errorf("token expiration is missing or already expired: %v", claims["exp"])
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("unknown user", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(selectFromUserTable)).WithArgs("missing@example.com").WillReturnError(sql.ErrNoRows)
		data, status, err := NewAuthService(db).Login(&models.Credentials{Email: "missing@example.com", Password: "secret-password"})
		if status != http.StatusUnauthorized || err == nil || data != nil {
			t.Errorf("Login() = (%q, %d, %v), want nil/401/error", data, status, err)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(selectFromUserTable)).WithArgs("person@example.com").
			WillReturnRows(sqlmock.NewRows([]string{"email", "password"}).AddRow("person@example.com", string(hash)))
		data, status, err := NewAuthService(db).Login(&models.Credentials{Email: "person@example.com", Password: "wrong-password"})
		if status != http.StatusUnauthorized || err == nil || data != nil {
			t.Errorf("Login() = (%q, %d, %v), want nil/401/error", data, status, err)
		}
	})
}

func TestAuthServiceLogout(t *testing.T) {
	data, status, err := NewAuthService(nil).Logout()
	if err != nil || status != http.StatusOK {
		t.Fatalf("Logout() = status %d, err %v", status, err)
	}
	var response map[string]string
	if err := json.Unmarshal(data, &response); err != nil || response["message"] != "Successfully logged out" {
		t.Errorf("unexpected logout response %q, err %v", data, err)
	}
}
