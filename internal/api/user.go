package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"
	"trainingApp/internal/auth"
	"trainingApp/internal/user"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type registerRequest struct {
	Username   string `json:"username"`
	SecondName string `json:"second_name"`
	Email      string `json:"email"`
	Password   string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type loginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type UserHandler struct {
	userStore UserStore
	secret    []byte
	tokenTTL  time.Duration
}

func NewUserHandler(store UserStore, secret []byte, ttl time.Duration) *UserHandler {
	return &UserHandler{userStore: store, secret: secret, tokenTTL: ttl}
}

type UserStore interface {
	Create(ctx context.Context, u user.User) (user.User, error)
	Delete(ctx context.Context, id int64) error
	GetById(ctx context.Context, id int64) (user.User, error)
	List(ctx context.Context, limit int) ([]user.User, error)
	FindByEmail(ctx context.Context, email string) (user.User, error)
}

func ParseToken(secret []byte, tokenStr string) (int64, error) {
	claims := &auth.Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}

func (h *UserHandler) registerUser(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	var req registerRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	userObj := user.User{
		Name:       req.Username,
		SecondName: req.SecondName,
		Email:      req.Email,
	}
	if err := userObj.Validate(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if err := user.ValidatePassword(req.Password); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	userObj.Password = string(hash)
	createdUser, err := h.userStore.Create(r.Context(), userObj)
	if err != nil {
		if errors.Is(err, user.ErrAlreadyExists) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		slog.Error("create user", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.Header().Set("Location", fmt.Sprintf("/api/users/%s", strconv.FormatInt(createdUser.ID, 10)))
	writeJSON(w, http.StatusCreated, createdUser)

	return
}

func (h *UserHandler) loginUser(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	var req loginRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := user.ValidateEmail(req.Email); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	userObj, err := h.userStore.FindByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		slog.Error("login: find user by email", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(userObj.Password), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	token, expiresAt, err := auth.NewToken(h.secret, userObj.ID, h.tokenTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create token")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{token, expiresAt})
	return

}
