Feature: Rate Limiter
  Como consumidor da API
  Quero que requisições excedentes sejam bloqueadas
  Para proteger o serviço de sobrecarga

  Background:
    Given a API do rate limiter está disponível em "http://localhost:8080"
    And um IP de origem exclusivo para o cenário

  Scenario: Requisições dentro do limite por IP são aceitas
    When eu envio 5 requisições para "/"
    Then todas as respostas devem ter status 200

  Scenario: Exceder o limite por IP retorna 429 com a mensagem esperada
    When eu envio 11 requisições para "/"
    Then a última resposta deve ter status 429
    And o corpo da última resposta deve ser "you have reached the maximum number of requests or actions allowed within a certain time frame"

  Scenario: Token com limite maior tem precedência sobre o limite de IP
    Given o header "API_KEY" com valor "abc123"
    When eu envio 20 requisições para "/"
    Then todas as respostas devem ter status 200
