package kaspi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type KaspiClient struct {
	BaseURL      string
	AccessToken  string
	RefreshToken string
	ExpireAt     time.Time
	ClientName   string
}

// Response structs
type RegisterResponse struct {
	Data struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		Expiration   string `json:"expirationDate"`
	} `json:"data"`
	StatusCode int    `json:"statusCode"`
	ErrorText  string `json:"errorText"`
}

type DeviceInfoResponse struct {
	Data struct {
		PosNum     string `json:"posNum"`
		SerialNum  string `json:"serialNum"`
		TerminalID string `json:"terminalId"`
	} `json:"data"`
	StatusCode int    `json:"statusCode"`
	ErrorText  string `json:"errorText"`
}

// Function to check device info
func (kc *KaspiClient) CheckDeviceInfo() error {
	url := fmt.Sprintf("%s/v2/deviceinfo", kc.BaseURL)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("accesstoken", kc.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var deviceResp DeviceInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceResp); err != nil {
		return err
	}

	if deviceResp.StatusCode != 0 {
		return errors.New(deviceResp.ErrorText)
	}

	fmt.Println("Device Info:", deviceResp.Data)
	return nil
}

// Function to register the cash register
func (kc *KaspiClient) RegisterCashService() error {
	url := fmt.Sprintf("%s/v2/register?name=%s", kc.BaseURL, kc.ClientName)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var registerResp RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&registerResp); err != nil {
		return err
	}

	if registerResp.StatusCode != 0 {
		return errors.New(registerResp.ErrorText)
	}

	kc.AccessToken = registerResp.Data.AccessToken
	kc.RefreshToken = registerResp.Data.RefreshToken

	expireAt, err := time.Parse("2006-01-02 15:04:05", registerResp.Data.Expiration)
	if err != nil {
		return err
	}
	kc.ExpireAt = expireAt

	fmt.Println("Registration successful! AccessToken:", kc.AccessToken)
	return nil
}

// Function to refresh the token
func (kc *KaspiClient) RefreshTokenService() error {
	url := fmt.Sprintf("%s/v2/revoke?name=%s&refreshToken=%s", kc.BaseURL, kc.ClientName, kc.RefreshToken)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var registerResp RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&registerResp); err != nil {
		return err
	}

	if registerResp.StatusCode != 0 {
		return errors.New(registerResp.ErrorText)
	}

	kc.AccessToken = registerResp.Data.AccessToken
	kc.RefreshToken = registerResp.Data.RefreshToken

	expireAt, err := time.Parse("2006-01-02 15:04:05", registerResp.Data.Expiration)
	if err != nil {
		return err
	}
	kc.ExpireAt = expireAt

	fmt.Println("Token refreshed! New AccessToken:", kc.AccessToken)
	return nil
}

// Example usage
func main() {
	client := KaspiClient{
		BaseURL:    "https://192.168.80.4:8080",
		ClientName: "MyTelegramBot",
	}

	// Step 1: Check device info
	if err := client.CheckDeviceInfo(); err != nil {
		fmt.Println("Device check failed:", err)
	}

	// Step 2: Register the cash register if needed
	if client.AccessToken == "" || time.Now().After(client.ExpireAt) {
		fmt.Println("Registering the cash register...")
		if err := client.RegisterCashService(); err != nil {
			fmt.Println("Registration failed:", err)
			return
		}
	}

	// Step 3: Refresh token if it is close to expiring
	if time.Until(client.ExpireAt) < 1*time.Hour {
		fmt.Println("Refreshing the token...")
		if err := client.RefreshTokenService(); err != nil {
			fmt.Println("Token refresh failed:", err)
		}
	}
}
