package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
)


type TemperatureResponse struct {
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

type ViaCEPResponse struct {
	CEP         string `json:"cep"`
	Logradouro  string `json:"logradouro"`
	Complemento string `json:"complemento"`
	Bairro      string `json:"bairro"`
	Localidade  string `json:"localidade"`
	UF          string `json:"uf"`
	IBGE        string `json:"ibge"`
	GIA         string `json:"gia"`
	DDD         string `json:"ddd"`
	SIAFI       string `json:"siafi"`
	Erro        bool   `json:"erro"`
}

type WeatherAPIResponse struct {
	Location struct {
		Name    string `json:"name"`
		Region  string `json:"region"`
		Country string `json:"country"`
	} `json:"location"`
	Current struct {
		TempC float64 `json:"temp_c"`
	} `json:"current"`
}

func celsiusToFahrenheit(celsius float64) float64 {
	return celsius*1.8 + 32
}

func celsiusToKelvin(celsius float64) float64 {
	return celsius + 273
}

func isValidCEP(cep string) bool {
	regex := regexp.MustCompile(`^\d{8}$`)
	return regex.MatchString(cep)
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var httpClient HTTPClient = &http.Client{}

func getLocationFromCEP(cep string, client HTTPClient) (*ViaCEPResponse, error) {
	url := fmt.Sprintf("https://viacep.com.br/ws/%s/json/", cep)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var viaCEPResponse ViaCEPResponse
	if err := json.NewDecoder(resp.Body).Decode(&viaCEPResponse); err != nil {
		return nil, err
	}

	if viaCEPResponse.Erro || viaCEPResponse.Localidade == "" {
		return nil, fmt.Errorf("CEP not found")
	}

	return &viaCEPResponse, nil
}

func getTemperatureFromLocation(city string, client HTTPClient) (*WeatherAPIResponse, error) {
	apiKey := os.Getenv("WEATHER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("WEATHER_API_KEY environment variable not set")
	}

	url := fmt.Sprintf("http://api.weatherapi.com/v1/current.json?key=%s&q=%s&aqi=no", apiKey, city)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get weather data: status code %d", resp.StatusCode)
	}

	var weatherResponse WeatherAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&weatherResponse); err != nil {
		return nil, err
	}

	return &weatherResponse, nil
}

func temperatureHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	cep := r.URL.Query().Get("cep")
	if cep == "" {
		responseWithError(w, http.StatusBadRequest, "CEP parameter is required")
		return
	}

	if !isValidCEP(cep) {
		responseWithError(w, http.StatusUnprocessableEntity, "invalid zipcode")
		return
	}

	location, err := getLocationFromCEP(cep, httpClient)
	if err != nil {
		log.Printf("Error getting location from CEP: %v", err)
		responseWithError(w, http.StatusNotFound, "can not find zipcode")
		return
	}

	weather, err := getTemperatureFromLocation(location.Localidade, httpClient)
	if err != nil {
		log.Printf("Error getting temperature: %v", err)
		responseWithError(w, http.StatusInternalServerError, "failed to get temperature data")
		return
	}

	tempC := weather.Current.TempC
	tempF := celsiusToFahrenheit(tempC)
	tempK := celsiusToKelvin(tempC)

	response := TemperatureResponse{
		TempC: tempC,
		TempF: tempF,
		TempK: tempK,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func responseWithError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Message: message})
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/temperature", temperatureHandler)
	http.HandleFunc("/health", healthCheckHandler)

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
