package auth

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func IssueToken(w http.ResponseWriter, login string, secret []byte) error {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   login,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
	})
	signed, err := token.SignedString(secret)
	if err != nil {
		return err
	}
	w.Header().Set("Authorization", "Bearer "+signed)
	http.SetCookie(w, &http.Cookie{Name: "token", Value: signed, Path: "/", HttpOnly: true})
	return nil
}
