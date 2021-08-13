terraform {
  required_providers {
    http-full = {
      source  = "registry.terraform.io/salrashid123/http-full"
      version = "~> 5.0.0"
    }
  }
}

provider "http-full" {
}
 
data "http" "example" {
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

output "data" {
  value = jsondecode(data.http.example.body)
}