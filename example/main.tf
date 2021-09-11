terraform {
  required_providers {
    http-full = {
      source = "salrashid123/http-full"
      version = "1.1.0"
    }
  }
}


provider "http-full" {
}
 
data "http" "example_json" {
  provider = http-full
  url = "https://httpbin.org/post"
  request_headers = {
    content-type = "application/json"
  }
  request_body = {
    foo = "bar"
    bar = "bar"
  }
}

data "http" "example_form" {
  provider = http-full
  url = "https://httpbin.org/post"
  request_headers = {
    content-type = "application/x-www-form-urlencoded"
  }
  request_body = {
    body = "foo=bar&bar=bar"
  }
}


data "http" "sts" {
  provider = http-full
  url = "https://stsserver-6w42z6vi3q-uc.a.run.app/token"
  request_headers = {
    content-type = "application/json"
  }
  request_body = {
    grant_type = "urn:ietf:params:oauth:grant-type:token-exchange"
    resource = "grpcserver-6w42z6vi3q-uc.a.run.app"
    audience = "grpcserver-6w42z6vi3q-uc.a.run.app"
    requested_token_type = "urn:ietf:params:oauth:token-type:access_token"
    subject_token = "iamtheeggman"
    subject_token_type = "urn:ietf:params:oauth:token-type:access_token"
  }
}


output "data_json" {
  value = jsondecode(data.http.example_json.body)
}

output "data_form" {
  value = jsondecode(data.http.example_form.body)
}


output "sts_token" {
  value = jsondecode(data.http.sts.body).access_token
}