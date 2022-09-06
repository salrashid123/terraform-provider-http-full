---
page_title: "HTTP Full Provider"
description: |-
  The HTTP provider allowing GET, POST and mTLS with arbitrary HTTP servers.
---

# HTTP Full Provider

The HTTP-FULL provider is a utility provider for interacting with generic HTTP
servers as part of a Terraform configuration.  Its identical to the terraform default 
[http provider](https://registry.terraform.io/providers/hashicorp/http/latest/docs/data-sources/http) except that this supports `mTLS`, `GET|POST|PUT|PATCH` verbs and `HTTPS_PROXY`.


## Example Usage

```terraform
terraform {
  required_providers {
    http-full = {
      source = "salrashid123/http-full"
    }
  }
}

provider "http-full" { }
 
# HTTP POST 
data "http" "example" {
  provider = http-full
  url = "https://httpbin.org/post"
  method = "POST"
  request_headers = {
    content-type = "application/json"
  }
  request_body = jsonencode({
    foo = "bar",
    bar = "bar"
  })
}

output "data" {
  value = jsondecode(data.http.example.response_body)
}
```