package services

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// RobokassaService provides helpers to work with Robokassa payment gateway.
type RobokassaService struct {
	MerchantLogin string
	Password1     string
	Password2     string
	BaseURL       string
	IsTest        bool
}

// GeneratePayURL builds a payment URL that the client should be redirected to.
func (s *RobokassaService) GeneratePayURL(invID int, outSum float64, description string) (string, error) {
	sig := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s:%.2f:%d:%s", s.MerchantLogin, outSum, invID, s.Password1))))
	params := url.Values{}
	params.Set("MerchantLogin", s.MerchantLogin)
	params.Set("OutSum", fmt.Sprintf("%.2f", outSum))
	params.Set("InvId", strconv.Itoa(invID))
	params.Set("Description", description)
	params.Set("SignatureValue", strings.ToUpper(sig))
	params.Set("IsTest", "1")

	return fmt.Sprintf("%s?%s", s.BaseURL, params.Encode()), nil
}

// VerifyResult validates callback signature from Robokassa.
func (s *RobokassaService) VerifyResult(outSum, invID, signature string) bool {
	expected := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", outSum, invID, s.Password2))))
	return strings.EqualFold(expected, signature)
}
