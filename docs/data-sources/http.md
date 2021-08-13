---
page_title: "HTTP Data Source"
description: |-
  Retrieves the content at an HTTP or HTTPS URL.
---

# `http` Data Source

The `http` data source is a fork of `https://registry.terraform.io/providers/hashicorp/http/latest/docs/data-sources/http`

with additional support for JSON `POST` and `mTLS`

## Example Usage

### GET

```hcl
provider "http-full" {
}
 
data "http" "example" {
  provider = http-full
  url = "https://httpbin.org/get"
}


output "data" {
  value = jsondecode(data.http.example.body)
}
```


### POST 

```hcl
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
```

### mTLS

```hcl
provider "http-full" {
}

data "http" "example" {
  provider = http-full
  url = "https://localhost:8081/get"

  ca = file("${path.module}/../certs/CA_crt.pem")
  client_crt = file("${path.module}/../certs/client.crt")
  client_key = file("${path.module}/../certs/client.key")  
}

output "data" {
  value = data.http.example.body
}
```


## Argument Reference

The following arguments are supported:

* `url` - (Required) The URL to request data from. This URL must respond with
  a `200 OK` response and a `text/*` or `application/json` Content-Type.

* `request_headers` - (Optional) A map of strings representing additional HTTP
  headers to include in the request.

* `request_body` - (Optional) A map of strings representing the JSON BODY to POST.

* `ca` - (Optional) Certificate Authority in PEM format for the target server.

* `client_crt` - (Optional) Client Certificate to present to the target server.

* `client_key` - (Optional) Client Certificate private Key to use for mTLS.

## Attributes Reference

The following attributes are exported:

* `body` - The raw body of the HTTP response.

* `response_headers` - A map of strings representing the response HTTP headers.
  Duplicate headers are contatenated with `, ` according to
  [RFC2616](https://www.w3.org/Protocols/rfc2616/rfc2616-sec4.html#sec4.2)



