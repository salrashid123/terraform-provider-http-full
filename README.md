
Terraform Provider for HTTP mTLS and POST Dataources
=======================================

This provider is a copy/fork of Terraforms [http Provider](https://registry.terraform.io/providers/hashicorp/http/latest/docs/data-sources/http) except that this support HTTP JSON POST and mTLS.

thats all.


- Website: https://www.terraform.io
- [![Gitter chat](https://badges.gitter.im/hashicorp-terraform/Lobby.png)](https://gitter.im/hashicorp-terraform/Lobby)

<img src="https://cdn.rawgit.com/hashicorp/terraform-website/master/content/source/assets/images/logo-hashicorp.svg" width="600px">

Maintainers
-----------

This provider plugin is maintained by the sal, just sal for now.

Requirements
------------

- [Terraform](https://www.terraform.io/downloads.html) 0.14.x
- [Go](https://golang.org/doc/install) 1.15 (to build the provider plugin)

Usage
---------------------



```hcl
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
 
# HTTP POST 
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


# mTLS
data "http" "example" {
  provider = http-full
  url = "https://localhost:8081/get"

  ca = file("${path.module}/../certs/CA_crt.pem")
  client_crt = file("${path.module}/../certs/client.crt")
  client_key = file("${path.module}/../certs/client.key")  
}
```


You can also use this to interact with an [STS server](https://github.com/salrashid123/sts_server) to get any auth token.

```hcl
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

output "sts_token" {
  value = jsondecode(data.http.sts.body).access_token
}
```


For mTLS and other configurations, see [example/index.md](blob/main/docs/index.md)

Building The Provider
---------------------

Clone repository to: `$GOPATH/src/github.com/salrashid123/terraform-provider-http-full

```sh
mkdir -p $GOPATH/src/github.com/terraform-providers
cd $GOPATH/src/github.com/terraform-providers
git clone https://github.com/salrashid123/terraform-provider-http-full.git
```

Enter the provider directory and build the provider

```sh
cd $GOPATH/src/github.com/terraform-providers/terraform-provider-http-full
make fmt
make build
```

Using the provider
----------------------

Copy the provider to your directory

```bash
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/salrashid123/http-full/5.0.0/linux_amd64/
cp $GOBIN/terraform-provider-http-full ~/.terraform.d/plugins/registry.terraform.io/salrashid123/http-full/5.0.0/linux_amd64/terraform-provider-http-full_v5.0.0
```

Then

```bash
cd example
terraform init

terraform apply
```

with

```hcl
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
```


...

In order to test the provider, you can simply run `make test`.


### TEST

```sh
$ make test
```
