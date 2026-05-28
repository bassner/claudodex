package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type jwtPayload struct {
	Email   string `json:"email"`
	Profile struct {
		Email string `json:"email"`
	} `json:"https://api.openai.com/profile"`
	Auth struct {
		PlanType  string `json:"chatgpt_plan_type"`
		UserID    string `json:"chatgpt_user_id"`
		AltUserID string `json:"user_id"`
		AccountID string `json:"chatgpt_account_id"`
		FedRAMP   bool   `json:"chatgpt_account_is_fedramp"`
	} `json:"https://api.openai.com/auth"`
	Exp int64 `json:"exp"`
}

type Claims struct {
	Email     string
	PlanType  string
	UserID    string
	AccountID string
	FedRAMP   bool
	Exp       int64
}

func ParseClaims(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[1] == "" {
		return Claims{}, fmt.Errorf("invalid jwt format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, err
	}
	var decoded jwtPayload
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return Claims{}, err
	}
	email := decoded.Email
	if email == "" {
		email = decoded.Profile.Email
	}
	userID := decoded.Auth.UserID
	if userID == "" {
		userID = decoded.Auth.AltUserID
	}
	return Claims{
		Email:     email,
		PlanType:  decoded.Auth.PlanType,
		UserID:    userID,
		AccountID: decoded.Auth.AccountID,
		FedRAMP:   decoded.Auth.FedRAMP,
		Exp:       decoded.Exp,
	}, nil
}

func ApplyClaims(file *File) {
	if file == nil {
		return
	}
	if claims, err := ParseClaims(file.Tokens.IDToken); err == nil {
		if claims.Email != "" {
			file.Tokens.Email = claims.Email
		}
		if claims.PlanType != "" {
			file.Tokens.PlanType = claims.PlanType
		}
		if claims.UserID != "" {
			file.Tokens.UserID = claims.UserID
		}
		if claims.AccountID != "" {
			file.Tokens.AccountID = claims.AccountID
		}
		file.Tokens.ChatGPTAccountIsFedRAMP = claims.FedRAMP
		if claims.Exp != 0 {
			file.Tokens.IDTokenExpiresAt = claims.Exp
		}
	}
	if claims, err := ParseClaims(file.Tokens.AccessToken); err == nil {
		if claims.Exp != 0 {
			file.Tokens.AccessTokenExpiresAt = claims.Exp
		}
		if file.Tokens.PlanType == "" {
			file.Tokens.PlanType = claims.PlanType
		}
	}
}
