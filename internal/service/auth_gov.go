package service

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// --- SOAP СТРУКТУРЫ ДЛЯ МЕЖСЕРВИСНОГО КЛИЕНТА ---
type GovSoapEnvelopeRequest struct {
	XMLName xml.Name `xml:"http://xmlsoap.org Envelope"`
	Body    struct {
		FetchData struct {
			Email       string `xml:"email"`
			MasterToken string `xml:"master_token"`
		} `xml:"fetch_secure_employee_data"`
	} `xml:"Body"`
}

type GovSoapEnvelopeResponse struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    struct {
		Response struct {
			Result struct {
				GovStatus    string `xml:"gov_status"`
				Authorized   bool   `xml:"authorized"`
				ErrorMessage string `xml:"error_message"`
			} `xml:"fetch_secure_employee_dataResult"`
		} `xml:"fetch_secure_employee_dataResponse"`
	} `xml:"Body"`
}

// verifyWithGovPlatform осуществляет синхронный SOAP/XML запрос к псевдогосплатформе
func (s *AuthService) verifyWithGovPlatform(ctx context.Context, email string) error {
	soapURL := "http://gov-soap-service:8000/" // Внутреннее имя контейнера в Docker-сети
	masterToken := "gos_master_secure_token_2026"

	// Формируем SOAP Envelope
	var reqEnvelope GovSoapEnvelopeRequest
	reqEnvelope.Body.FetchData.Email = email
	reqEnvelope.Body.FetchData.MasterToken = masterToken

	xmlBytes, err := xml.Marshal(reqEnvelope)
	if err != nil {
		return fmt.Errorf("failed to marshal SOAP request: %w", err)
	}

	// Отправляем POST запрос с таймаутом контекста
	req, err := http.NewRequestWithContext(ctx, "POST", soapURL, strings.NewReader(xml.Header+string(xmlBytes)))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request to GOV: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("governmental verification platform is currently unavailable")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("governmental platform returned bad status: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read GOV response body: %w", err)
	}

	var respEnvelope GovSoapEnvelopeResponse
	if err := xml.Unmarshal(bodyBytes, &respEnvelope); err != nil {
		return fmt.Errorf("failed to unmarshal GOV XML response: %w", err)
	}

	result := respEnvelope.Body.Response.Result

	// ТЕСТ 3: Гражданин отсутствует в реестре
	if result.ErrorMessage == "USER_NOT_FOUND" {
		return errors.New("registration rejected: identity not found in governmental registry")
	}

	// ТЕСТ 2: Профиль заблокирован государством
	if result.GovStatus == "BLOCKED" || !result.Authorized {
		return errors.New("access denied: account is blocked by governmental decree")
	}

	return nil
}
