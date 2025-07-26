package main

import (
  "encoding/json"
  "io"
  "net/http"
  "net/http/httptest"
  "strings"
  "testing"
)

func TestCelsiusToFahrenheit(t *testing.T) {
  tests := []struct {
    name     string
    celsius  float64
    expected float64
  }{
    {"Zero", 0, 32},
    {"Positive", 25, 77},
    {"Negative", -10, 14},
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      result := celsiusToFahrenheit(tt.celsius)
      if result != tt.expected {
        t.Errorf("celsiusToFahrenheit(%f) = %f; want %f", tt.celsius, result, tt.expected)
      }
    })
  }
}

func TestCelsiusToKelvin(t *testing.T) {
  tests := []struct {
    name     string
    celsius  float64
    expected float64
  }{
    {"Zero", 0, 273},
    {"Positive", 25, 298},
    {"Negative", -10, 263},
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      result := celsiusToKelvin(tt.celsius)
      if result != tt.expected {
        t.Errorf("celsiusToKelvin(%f) = %f; want %f", tt.celsius, result, tt.expected)
      }
    })
  }
}

func TestIsValidCEP(t *testing.T) {
  tests := []struct {
    name     string
    cep      string
    expected bool
  }{
    {"Valid CEP", "12345678", true},
    {"Invalid CEP - Letters", "1234567a", false},
    {"Invalid CEP - Too Short", "1234567", false},
    {"Invalid CEP - Too Long", "123456789", false},
    {"Invalid CEP - Empty", "", false},
    {"Invalid CEP - With Hyphen", "12345-678", false},
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      result := isValidCEP(tt.cep)
      if result != tt.expected {
        t.Errorf("isValidCEP(%s) = %v; want %v", tt.cep, result, tt.expected)
      }
    })
  }
}

// Mock HTTP client for testing
type MockHTTPClient struct {
  DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
  return m.DoFunc(req)
}

// Save the original HTTP client
var originalHTTPClient = http.DefaultClient

// Helper function to setup mock HTTP client
func setupMockHTTPClient(doFunc func(req *http.Request) (*http.Response, error)) *MockHTTPClient {
  return &MockHTTPClient{DoFunc: doFunc}
}

// Helper function to create mock response
func mockResponse(statusCode int, body string) *http.Response {
  return &http.Response{
    StatusCode: statusCode,
    Body:       io.NopCloser(strings.NewReader(body)),
    Header:     make(http.Header),
  }
}

func TestGetLocationFromCEP(t *testing.T) {
  // Test case 1: Valid CEP
  mockClient := setupMockHTTPClient(func(req *http.Request) (*http.Response, error) {
    validResponse := `{
      "cep": "01001000",
      "logradouro": "Praça da Sé",
      "complemento": "lado ímpar",
      "bairro": "Sé",
      "localidade": "São Paulo",
      "uf": "SP",
      "ibge": "3550308",
      "gia": "1004",
      "ddd": "11",
      "siafi": "7107"
    }`
    return mockResponse(http.StatusOK, validResponse), nil
  })

  location, err := getLocationFromCEP("01001000", mockClient)
  if err != nil {
    t.Errorf("Expected no error, got %v", err)
  }

  if location.Localidade != "São Paulo" {
    t.Errorf("Expected location to be 'São Paulo', got '%s'", location.Localidade)
  }

  // Test case 2: Invalid CEP format (this doesn't use the mock as it's validated before API call)
  if isValidCEP("123456") {
    t.Errorf("Expected CEP '123456' to be invalid")
  }

  // Test case 3: CEP not found
  mockClient = setupMockHTTPClient(func(req *http.Request) (*http.Response, error) {
    notFoundResponse := `{"erro": true}`
    return mockResponse(http.StatusOK, notFoundResponse), nil
  })

  _, err = getLocationFromCEP("99999999", mockClient)
  if err == nil {
    t.Errorf("Expected error for CEP not found, got nil")
  }
}

func TestGetTemperatureFromLocation(t *testing.T) {
  // Set environment variable for testing
  t.Setenv("WEATHER_API_KEY", "test-api-key")

  // Test case 1: Valid location
  mockClient := setupMockHTTPClient(func(req *http.Request) (*http.Response, error) {
    validResponse := `{
      "location": {
        "name": "São Paulo",
        "region": "Sao Paulo",
        "country": "Brazil"
      },
      "current": {
        "temp_c": 25.0
      }
    }`
    return mockResponse(http.StatusOK, validResponse), nil
  })

  weather, err := getTemperatureFromLocation("São Paulo", mockClient)
  if err != nil {
    t.Errorf("Expected no error, got %v", err)
  }

  if weather.Current.TempC != 25.0 {
    t.Errorf("Expected temperature to be 25.0, got %f", weather.Current.TempC)
  }

  // Test case 2: API error
  mockClient = setupMockHTTPClient(func(req *http.Request) (*http.Response, error) {
    return mockResponse(http.StatusBadRequest, `{"error":{"code":1006,"message":"No matching location found."}}`), nil
  })

  _, err = getTemperatureFromLocation("NonExistentCity", mockClient)
  if err == nil {
    t.Errorf("Expected error for invalid location, got nil")
  }
}

func TestTemperatureHandlerInvalidCEP(t *testing.T) {
  // Create a request with an invalid CEP
  req, err := http.NewRequest("GET", "/temperature?cep=1234567", nil)
  if err != nil {
    t.Fatal(err)
  }

  // Create a ResponseRecorder to record the response
  rr := httptest.NewRecorder()
  handler := http.HandlerFunc(temperatureHandler)

  // Call the handler
  handler.ServeHTTP(rr, req)

  // Check the status code
  if status := rr.Code; status != http.StatusUnprocessableEntity {
    t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnprocessableEntity)
  }

  // Check the response body
  var response ErrorResponse
  if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
    t.Errorf("Failed to parse response body: %v", err)
  }

  expectedMessage := "invalid zipcode"
  if response.Message != expectedMessage {
    t.Errorf("handler returned unexpected body: got %v want %v", response.Message, expectedMessage)
  }
}

func TestTemperatureHandlerSuccess(t *testing.T) {
  // Save original HTTP client and restore it after test
  originalClient := httpClient
  defer func() { httpClient = originalClient }()

  // Set environment variable for testing
  t.Setenv("WEATHER_API_KEY", "test-api-key")

  // Create a mock client that handles both API calls
  mockClient := &MockHTTPClient{
    DoFunc: func(req *http.Request) (*http.Response, error) {
      // Check which API is being called based on the URL
      if strings.Contains(req.URL.String(), "viacep.com.br") {
        // Mock ViaCEP response
        viaCEPResponse := `{
          "cep": "01001000",
          "logradouro": "Praça da Sé",
          "complemento": "lado ímpar",
          "bairro": "Sé",
          "localidade": "São Paulo",
          "uf": "SP",
          "ibge": "3550308",
          "gia": "1004",
          "ddd": "11",
          "siafi": "7107"
        }`
        return mockResponse(http.StatusOK, viaCEPResponse), nil
      } else if strings.Contains(req.URL.String(), "weatherapi.com") {
        // Mock WeatherAPI response
        weatherResponse := `{
          "location": {
            "name": "São Paulo",
            "region": "Sao Paulo",
            "country": "Brazil"
          },
          "current": {
            "temp_c": 25.0
          }
        }`
        return mockResponse(http.StatusOK, weatherResponse), nil
      }

      // Default response for unexpected URLs
      return mockResponse(http.StatusInternalServerError, "{}"), nil
    },
  }

  // Set our mock client as the default client
  httpClient = mockClient

  // Create a request with a valid CEP
  req, err := http.NewRequest("GET", "/temperature?cep=01001000", nil)
  if err != nil {
    t.Fatal(err)
  }

  // Create a ResponseRecorder to record the response
  rr := httptest.NewRecorder()
  handler := http.HandlerFunc(temperatureHandler)

  // Call the handler
  handler.ServeHTTP(rr, req)

  // Check the status code
  if status := rr.Code; status != http.StatusOK {
    t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
  }

  // Check the response body
  var response TemperatureResponse
  if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
    t.Errorf("Failed to parse response body: %v", err)
  }

  // Check temperature values
  expectedTempC := 25.0
  expectedTempF := celsiusToFahrenheit(expectedTempC)
  expectedTempK := celsiusToKelvin(expectedTempC)

  if response.TempC != expectedTempC {
    t.Errorf("Expected temp_C to be %f, got %f", expectedTempC, response.TempC)
  }

  if response.TempF != expectedTempF {
    t.Errorf("Expected temp_F to be %f, got %f", expectedTempF, response.TempF)
  }

  if response.TempK != expectedTempK {
    t.Errorf("Expected temp_K to be %f, got %f", expectedTempK, response.TempK)
  }
}

func TestTemperatureHandlerCEPNotFound(t *testing.T) {
  // Save original HTTP client and restore it after test
  originalClient := httpClient
  defer func() { httpClient = originalClient }()

  // Create a mock client that returns a CEP not found error
  mockClient := &MockHTTPClient{
    DoFunc: func(req *http.Request) (*http.Response, error) {
      // Mock ViaCEP response for not found CEP
      return mockResponse(http.StatusOK, `{"erro": true}`), nil
    },
  }

  // Set our mock client as the default client
  httpClient = mockClient

  // Create a request with a valid but non-existent CEP
  req, err := http.NewRequest("GET", "/temperature?cep=99999999", nil)
  if err != nil {
    t.Fatal(err)
  }

  // Create a ResponseRecorder to record the response
  rr := httptest.NewRecorder()
  handler := http.HandlerFunc(temperatureHandler)

  // Call the handler
  handler.ServeHTTP(rr, req)

  // Check the status code
  if status := rr.Code; status != http.StatusNotFound {
    t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
  }

  // Check the response body
  var response ErrorResponse
  if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
    t.Errorf("Failed to parse response body: %v", err)
  }

  expectedMessage := "can not find zipcode"
  if response.Message != expectedMessage {
    t.Errorf("handler returned unexpected body: got %v want %v", response.Message, expectedMessage)
  }
}

func TestHealthCheckHandler(t *testing.T) {
  // Create a request
  req, err := http.NewRequest("GET", "/health", nil)
  if err != nil {
    t.Fatal(err)
  }

  // Create a ResponseRecorder to record the response
  rr := httptest.NewRecorder()
  handler := http.HandlerFunc(healthCheckHandler)

  // Call the handler
  handler.ServeHTTP(rr, req)

  // Check the status code
  if status := rr.Code; status != http.StatusOK {
    t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
  }

  // Check the response body
  expected := "OK"
  if rr.Body.String() != expected {
    t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
  }
}
