package utils

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

func GenerateReferralLink(botUsername string, shopID uint, role string) string {
	// Шифруем параметры (shopID и роль) с помощью base64
	payload := fmt.Sprintf("%d|%s", shopID, role)
	encodedPayload := base64.URLEncoding.EncodeToString([]byte(payload))

	// Формируем ссылку
	return fmt.Sprintf("https://t.me/%s?start=%s", botUsername, encodedPayload)
}

func DecodePayload(payload string) (uint, string, error) {
	// Декодируем base64-параметр
	decodedPayload, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return 0, "", fmt.Errorf("ошибка декодирования: %v", err)
	}

	// Разделяем параметры (shopID и роль)
	parts := strings.Split(string(decodedPayload), "|")
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("неверный формат параметра")
	}

	// Преобразуем shopID в число
	shopID, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return 0, "", fmt.Errorf("ошибка преобразования shopID: %v", err)
	}

	role := parts[1]
	return uint(shopID), role, nil
}