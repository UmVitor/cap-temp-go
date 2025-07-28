# CEP Temperature API

Este sistema em Go recebe um CEP, identifica a cidade e retorna o clima atual (temperatura em graus Celsius, Fahrenheit e Kelvin).

## Funcionalidades

- Recebe um CEP válido de 8 dígitos
- Realiza a pesquisa do CEP usando a API ViaCEP
- Obtém a temperatura atual usando a API WeatherAPI
- Retorna as temperaturas em Celsius, Fahrenheit e Kelvin

## Requisitos

- Go 1.24+
- Docker e Docker Compose (para execução em contêiner)
- Chave de API da WeatherAPI (https://www.weatherapi.com/)

## Como executar

### Localmente

1. Clone o repositório
2. Configure a variável de ambiente com sua chave da WeatherAPI:
   ```
   export WEATHER_API_KEY=sua_chave_api
   ```
3. Execute a aplicação:
   ```
   go run main.go
   ```

### Com Docker Compose

1. Configure a variável de ambiente com sua chave da WeatherAPI:
   ```
   export WEATHER_API_KEY=sua_chave_api
   ```
2. Execute com Docker Compose:
   ```
   docker-compose up
   ```

## Endpoints

### GET /temperature?cep={cep}

Retorna a temperatura atual para a localidade do CEP informado.

#### Parâmetros

- `cep`: CEP válido de 8 dígitos (apenas números)

#### Respostas

- **200 OK**: Temperatura obtida com sucesso
  ```json
  {
    "temp_C": 28.5,
    "temp_F": 83.3,
    "temp_K": 301.5
  }
  ```

- **422 Unprocessable Entity**: CEP com formato inválido
  ```json
  {
    "message": "invalid zipcode"
  }
  ```

- **404 Not Found**: CEP não encontrado
  ```json
  {
    "message": "can not find zipcode"
  }
  ```

### GET /health

Endpoint para verificação de saúde da aplicação.

## Deploy no Google Cloud Run

1. Rota:
   ```
   https://go-lab-cap-temp-437097252417.us-central1.run.app/
   ```

2. Exemplo usando query param para obter a temperatura
   ```
   https://go-lab-cap-temp-437097252417.us-central1.run.app/temperature?cep=06213070
   ```
