---
page_title: "HTTP Full Provider"
description: |-
  The HTTP provider interacts with STS servers.
---

# HTTP Full Provider

The HTTP-FUL provider is a utility provider for interacting with generic HTTP
servers as part of a Terraform configuration.  Its identical to the terraform default 
[http provider](https://registry.terraform.io/providers/hashicorp/http/latest/docs/data-sources/http) except that this support mTLS and POST.

>> note, this provider only supports POST for JSON data.

