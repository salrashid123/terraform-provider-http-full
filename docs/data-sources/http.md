---
page_title: "HTTP-FULL Data Source"
description: |-
  Retrieves the content at an HTTP or HTTPS URL with support for GET, POST and mTLS
---

# `http` Data Source

The `http` data source is a fork of `https://registry.terraform.io/providers/hashicorp/http/latest/docs/data-sources/http`

with additional support for JSON `POST|PUT|PATCH|DELETE` and `mTLS`

Note, the `method` parameter defaults to `GET` but specifying it will allow overriding the verb.

## Example Usage

### GET

```hcl
provider "http-full" {}

data "http" "example_get" {
  provider = http-full
  url = "https://localhost:8081/get"

  method = "GET"

}

output "data_status_code" {
  value = data.http.example_get.status_code
}

output "data_get" {
  value = data.http.example_get.body
}
```


### POST JSON

```hcl
provider "http-full" {}

data "http" "example_json" {
  provider = http-full
  url = "https://localhost:8081/post"

  method = "POST"

  request_headers = {
    content-type = "application/json"
  }

  request_body = jsonencode({foo = "bar",bar = "bar"})

}

output "data_json" {
  value = data.http.example_json.body
}
```

### POST Form

To POST Form data, use the reserved key `body` within `request_body` as shown below:

```hcl
provider "http-full" {}
 
data "http" "example_form" {
  provider = http-full
  url = "https://localhost:8081/post"

  method = "POST"
  request_headers = {
    content-type = "application/x-www-form-urlencoded"
  }

  request_body = "foo=bar&bar=bar"
 
}
```

### mTLS

```hcl
provider "http-full" {}

data "http" "example_json_mtls" {
  provider = http-full
  url = "https://localhost:8081/post"

  method = "POST"
  request_headers = {
    content-type = "application/json"
  }

  request_body = jsonencode({foo = "bar",bar = "bar"})

  ca = file("${path.module}/certs/CA_crt.pem")
  client_crt = file("${path.module}/certs/client.crt")
  client_key = file("${path.module}/certs/client.key")  
}

output "data_json_mtls" {
  value = data.http.example_json_mtls.body
}
```


## Argument Reference

The following arguments are supported:

* `url` - (Required) The URL to request data from. This URL must respond with
  a `200 OK` response and a `text/*` or `application/json` Content-Type.

* `method` - (Optional) String representing the HTTP verb to use in the call;
  (default=`GET`; if `request_body` is set, defaults to `POST`).

* `insecure_skip_verify` - (Optional) Skip server TLS verification (default=`false`).

* `request_timeout_ms` - (Optional) Timeout the request in ms

* `request_headers` - (Optional) A map of strings representing additional HTTP
  headers to include in the request.

* `request_body` - (Optional) String representing the BODY to POST.

* `ca` - (Optional) Certificate Authority in PEM format for the target server.

* `client_crt` - (Optional) Client Certificate to present to the target server.

* `client_key` - (Optional) Client Certificate private Key to use for mTLS.

## Attributes Reference

The following attributes are exported:

* `status_code` - The status_code of the HTTP response if not error

* `body` - The raw body of the HTTP response.

* `response_headers` - A map of strings representing the response HTTP headers.
  Duplicate headers are concatenated with `, ` according to
  [RFC2616](https://www.w3.org/Protocols/rfc2616/rfc2616-sec4.html#sec4.2)



