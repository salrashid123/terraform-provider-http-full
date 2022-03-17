terraform {
  required_providers {
    http-full = {
      source = "salrashid123/http-full"
      version = "1.2.2"
    }
  }
}



provider "http-full" {}


data "http" "sts" {
  provider = http-full

  url = "https://stsserver-6w42z6vi3q-uc.a.run.app/token"

  method = "POST"
  request_headers = {
    content-type = "application/json"
  }
  request_body = jsonencode({
    grant_type = "urn:ietf:params:oauth:grant-type:token-exchange",
    resource = "grpcserver-6w42z6vi3q-uc.a.run.app",
    audience = "grpcserver-6w42z6vi3q-uc.a.run.app",
    requested_token_type = "urn:ietf:params:oauth:token-type:access_token",
    subject_token = "iamtheeggman",
    subject_token_type = "urn:ietf:params:oauth:token-type:access_token"
  })
}

output "sts_token" {
  value = jsondecode(data.http.sts.body).access_token
}



data "http" "example_get" {
  provider = http-full
  url = "https://localhost:8081/get"

  method = "GET"

  ca = file("${path.module}/certs/CA_crt.pem")
  client_crt = file("${path.module}/certs/client.crt")
  client_key = file("${path.module}/certs/client.key")  
}

output "data_get" {
  value = data.http.example_get.body
}

data "http" "example_json" {
  provider = http-full
  url = "https://localhost:8081/post"

  method = "POST"
  request_headers = {
    content-type = "application/json"
  }

  request_body = jsonencode(
    {
      foo = "bar",
      bar = "bar"
    })

  ca = file("${path.module}/certs/CA_crt.pem")
  client_crt = file("${path.module}/certs/client.crt")
  client_key = file("${path.module}/certs/client.key")  
}

output "data_json" {
  value = data.http.example_json.body
}

data "http" "example_form" {
  provider = http-full
  url = "https://localhost:8081/post"

  method = "POST"
  request_headers = {
    content-type = "application/x-www-form-urlencoded"
  }

  request_body = "foo=bar&bar=bar"

  ca = file("${path.module}/certs/CA_crt.pem")
  client_crt = file("${path.module}/certs/client.crt")
  client_key = file("${path.module}/certs/client.key")  
}

output "data_form" {
  value = data.http.example_form.body
}
