## Test cases for mTLS


### Start Server

```
go run src/server/main.go
```

```
# test standalone client

go run src/client/main.go
```

test provider

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