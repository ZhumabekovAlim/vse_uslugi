package services

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type RobokassaService struct {
	MerchantLogin string

	// Боевые
	Password1 string
	Password2 string

	// Тестовые
	TestPassword1 string
	TestPassword2 string

	BaseURL string // пример: "https://auth.robokassa.ru/Merchant/Index.aspx" или ".kz"
	IsTest  bool   // глобальный флаг тестового режима
}

// вспомогательные геттеры паролей, чтобы не дублировать логику
func (s *RobokassaService) pass1() string {
	if s.IsTest && s.TestPassword1 != "" {
		return s.TestPassword1
	}
	return s.Password1
}
func (s *RobokassaService) pass2(isTest bool) string {
	if isTest && s.TestPassword2 != "" {
		return s.TestPassword2
	}
	return s.Password2
}

// GeneratePayURL — формирование ссылки на оплату
func (s *RobokassaService) GeneratePayURL(invID int, outSum float64, description string) (string, error) {
	// подпись: md5(MerchantLogin:OutSum:InvId:Password1)
	raw := fmt.Sprintf("%s:%.2f:%d:%s", s.MerchantLogin, outSum, invID, s.pass1())
	sig := fmt.Sprintf("%x", md5.Sum([]byte(raw)))

	params := url.Values{}
	params.Set("MerchantLogin", s.MerchantLogin)
	params.Set("OutSum", fmt.Sprintf("%.2f", outSum))
	params.Set("InvId", strconv.Itoa(invID))
	params.Set("Description", description)
	params.Set("SignatureValue", strings.ToUpper(sig))

	// очень важно: для теста обязательно IsTest=1
	if s.IsTest {
		params.Set("IsTest", "1")
	}

	return fmt.Sprintf("%s?%s", s.BaseURL, params.Encode()), nil
}

// VerifyResult — валидация подписи от Robokassa (result URL).
// В тестовом режиме Robokassa шлёт такие же параметры, но хэш считается по TestPassword2.
// Лучше читать IsTest из входящих параметров и выбирать пароль динамически.
func (s *RobokassaService) VerifyResult(outSum, invID, signature string, isTest bool) bool {
	// подпись: md5(OutSum:InvId:Password2)
	raw := fmt.Sprintf("%s:%s:%s", outSum, invID, s.pass2(isTest))
	expected := fmt.Sprintf("%x", md5.Sum([]byte(raw)))
	return strings.EqualFold(expected, signature)
}
