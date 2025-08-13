package auth

import (
	"crypto/sha512"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// 固定盐值
const salt = "etaMonitorSalt"

// GeneratePasswordHash 使用固定盐值和双重哈希生成密码哈希
func GeneratePasswordHash(password string) (string, error) {
	// 第一次哈希：使用SHA512(密码+盐)
	sha512Hash := sha512.New()
	sha512Hash.Write([]byte(password))
	sha512Hash.Write([]byte(salt))
	firstHashResult := sha512Hash.Sum(nil)

	// 第二次哈希：使用bcrypt(SHA512结果)
	finalHash, err := bcrypt.GenerateFromPassword(firstHashResult, bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt哈希失败: %w", err)
	}

	return string(finalHash), nil
}

// VerifyPassword 验证密码
func VerifyPassword(password, hashedPassword string) error {
	// 第一次哈希：使用SHA512(密码+盐)
	sha512Hash := sha512.New()
	sha512Hash.Write([]byte(password))
	sha512Hash.Write([]byte(salt))
	firstHashResult := sha512Hash.Sum(nil)

	// 验证bcrypt哈希
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), firstHashResult)
}
