---
page_title: "HTTP Full Provider"
description: |-
  The HTTP provider allowing GET, POST and mTLS with arbitrary HTTP servers.
---

# HTTP Full Provider

The HTTP-FULL provider is a utility provider for interacting with generic HTTP
servers as part of a Terraform configuration.  Its identical to the terraform default 
[http provider](https://registry.terraform.io/providers/hashicorp/http/latest/docs/data-sources/http) except that this support `mTL`S and `GET|POST|PUT|PATCH`.


## Example Usage

```terraform
terraform {
  required_providers {
    http-full = {
      source = "salrashid123/http-full"
      version = "1.2.2"
    }
  }
}

provider "http-full" { }
 
# HTTP POST 
data "http" "example" {
  provider = http-full
  url = "https://httpbin.org/post"
  request_headers = {
    content-type = "application/json"
  }
  request_body = jsonencode({
    foo = "bar",
    bar = "bar"
  })
}

output "data" {
  value = jsondecode(data.http.example.body)
}
```