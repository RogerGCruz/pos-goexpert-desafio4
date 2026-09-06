package bdd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cucumber/godog"
)

// apiContext mantém o estado de um cenário: IP/token usados e resultados
// das requisições feitas contra a API real.
type apiContext struct {
	baseURL     string
	ip          string
	apiKey      string
	lastStatus  int
	lastBody    string
	allStatuses []int
}

func (a *apiContext) reset() {
	a.ip = ""
	a.apiKey = ""
	a.lastStatus = 0
	a.lastBody = ""
	a.allStatuses = nil
}

func (a *apiContext) apiIsAvailableAt(url string) error {
	a.baseURL = url
	return nil
}

// uniqueIP gera um IP falso exclusivo por cenário para evitar interferência
// de estado (contagem/bloqueio) entre cenários e execuções consecutivas.
func (a *apiContext) uniqueIP() error {
	now := time.Now().UnixNano()
	a.ip = fmt.Sprintf("10.99.%d.%d", (now/1e6)%250, now%250)
	return nil
}

func (a *apiContext) withHeader(name, value string) error {
	if name == "API_KEY" {
		a.apiKey = value
	}
	return nil
}

func (a *apiContext) sendRequests(count int, path string) error {
	client := &http.Client{Timeout: 3 * time.Second}
	for i := 0; i < count; i++ {
		req, err := http.NewRequest(http.MethodGet, a.baseURL+path, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Forwarded-For", a.ip)
		if a.apiKey != "" {
			req.Header.Set("API_KEY", a.apiKey)
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("request %d failed: %w", i+1, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}

		a.lastStatus = resp.StatusCode
		a.lastBody = string(body)
		a.allStatuses = append(a.allStatuses, resp.StatusCode)
	}
	return nil
}

func (a *apiContext) allResponsesShouldHaveStatus(expected int) error {
	for i, s := range a.allStatuses {
		if s != expected {
			return fmt.Errorf("request %d: expected status %d, got %d", i+1, expected, s)
		}
	}
	return nil
}

func (a *apiContext) lastResponseShouldHaveStatus(expected int) error {
	if a.lastStatus != expected {
		return fmt.Errorf("expected last status %d, got %d (body: %q)", expected, a.lastStatus, a.lastBody)
	}
	return nil
}

func (a *apiContext) lastResponseBodyShouldBe(expected string) error {
	if a.lastBody != expected {
		return fmt.Errorf("expected body %q, got %q", expected, a.lastBody)
	}
	return nil
}

// InitializeScenario registra os steps Gherkin e garante estado isolado por cenário.
func InitializeScenario(sc *godog.ScenarioContext) {
	api := &apiContext{}

	sc.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		api.reset()
		return ctx, nil
	})

	sc.Given(`^a API do rate limiter está disponível em "([^"]*)"$`, api.apiIsAvailableAt)
	sc.Given(`^um IP de origem exclusivo para o cenário$`, api.uniqueIP)
	sc.Given(`^o header "([^"]*)" com valor "([^"]*)"$`, api.withHeader)
	sc.When(`^eu envio (\d+) requisições para "([^"]*)"$`, api.sendRequests)
	sc.Then(`^todas as respostas devem ter status (\d+)$`, api.allResponsesShouldHaveStatus)
	sc.Then(`^a última resposta deve ter status (\d+)$`, api.lastResponseShouldHaveStatus)
	sc.Then(`^o corpo da última resposta deve ser "([^"]*)"$`, api.lastResponseBodyShouldBe)
}
