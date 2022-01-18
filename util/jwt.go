package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"github.com/byteintellect/protos_go/users/v1"
	"github.com/dgrijalva/jwt-go"
	"io"
	"time"
)

type Claims struct {
	jwt.StandardClaims
	usersv1.UserDto
}

func GenerateAccessRefreshKeyPair(accessTokenDuration, refreshTokenDuration string, secretKey string, dto usersv1.UserDto) (map[string]string, error) {
	accessTokenExp, err := time.ParseDuration(accessTokenDuration)
	if err != nil {
		return nil, err
	}
	refreshTokenExp, err := time.ParseDuration(refreshTokenDuration)
	if err != nil {
		return nil, err
	}
	accessToken, err := CreateToken(dto, secretKey, time.Now().Add(accessTokenExp))
	if err != nil {
		return nil, err
	}
	refreshToken, err := CreateToken(dto, secretKey, time.Now().Add(refreshTokenExp))
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	}, nil
}

func CreateToken(dto usersv1.UserDto, secretKey string, expirationTime time.Time) (string, error) {
	var err error
	claims := &Claims{
		UserDto: dto,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := at.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}
	return token, nil
}

func ValidateTokenExpiry(secretKey string, token string) (*Claims, bool) {
	claims := &Claims{}
	at, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})
	if err != nil || !at.Valid {
		return nil, false
	}
	return claims, true
}

func EncryptAES(cipherKey, text string) (string, error) {
	textInBytes := []byte(text)
	block, err := aes.NewCipher([]byte(cipherKey))
	if err != nil {
		return "", err
	}
	b := base64.StdEncoding.EncodeToString(textInBytes)
	ciphertext := make([]byte, aes.BlockSize+len(b))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(b))
	return text, nil
}
