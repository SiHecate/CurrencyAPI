package service

import (
	"Currency/database"
	"Currency/model"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type TokenService interface {
	Generate() (string, error)
	List() ([]model.Token, error)
	Check(c *fiber.Ctx, token string) error
}

type tokenService struct{}

func NewTokenService() TokenService {
	return &tokenService{}
}

// Generate methodu admin panelinde token oluşturmak için kullanılacak (büyük ihtimalle hiç bir zaman admin paneli olmayacak ama whatever...)
func (ts *tokenService) Generate() (string, error) {
	token := tokenGenerator()
	return token, nil
}

// List methodu admin panelinde tokenları listelemek için kullanılacak (büyük ihtimalle hiç bir zaman admin paneli olmayacak ama whatever...)
func (ts *tokenService) List() ([]model.Token, error) {
	var tokens []model.Token
	if err := database.Conn.Find(&tokens).Error; err != nil {
		return nil, err
	}

	return tokens, nil
}

// Check methodu middleware olarak kullanılacak
func (ts *tokenService) Check(c *fiber.Ctx, token string) error {
	token = c.Query("token")
	if token == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Token not found",
		})
	}
	var tokens []model.Token
	if err := database.Conn.Where("token = ?", token).First(&tokens).Error; err != nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}
	return c.Next()
}

func tokenGenerator() string {
	newToken := make([]byte, 12)
	_, err := rand.Read(newToken)
	if err != nil {
		log.Fatalf("Rastgele token oluşturulurken bir hata oluştu: %v", err)
	}
	tokenHex := hex.EncodeToString(newToken)

	if err := database.Conn.Create(&model.Token{Token: tokenHex}).Error; err != nil {
		log.Fatalf("Token database'e kaydedilirken bir hata oluştu: %v", err)
	}

	fmt.Println("Token:", tokenHex)
	return tokenHex
}
