package provider

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

type TestHttpMock struct {
	server *httptest.Server
}

const testDataSourceConfig_basic = `
data "http" "http_test" {
  url = "%s/meta_%d.txt"
}

output "body" {
  value = data.http.http_test.body
}

output "response_body" {
  value = data.http.http_test.response_body
}

output "response_headers" {
  value = data.http.http_test.response_headers
}
`

func TestDataSource_http200(t *testing.T) {
	testHttpMock := setUpMockHttpServer()

	defer testHttpMock.server.Close()

	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testDataSourceConfig_basic, testHttpMock.server.URL, 200),
				Check: func(s *terraform.State) error {
					_, ok := s.RootModule().Resources["data.http.http_test"]
					if !ok {
						return fmt.Errorf("missing data resource")
					}

					outputs := s.RootModule().Outputs

					if outputs["body"].Value != "1.0.0" {
						return fmt.Errorf(
							`'body' output is %s; want '1.0.0'`,
							outputs["body"].Value,
						)
					}

					if outputs["response_body"].Value != "1.0.0" {
						return fmt.Errorf(
							`'response_body' output is %s; want '1.0.0'`,
							outputs["response_body"].Value,
						)
					}

					response_headers := outputs["response_headers"].Value.(map[string]interface{})

					if response_headers["X-Single"].(string) != "foobar" {
						return fmt.Errorf(
							`'X-Single' response header is %s; want 'foobar'`,
							response_headers["X-Single"].(string),
						)
					}

					if response_headers["X-Double"].(string) != "1, 2" {
						return fmt.Errorf(
							`'X-Double' response header is %s; want '1, 2'`,
							response_headers["X-Double"].(string),
						)
					}

					return nil
				},
			},
		},
	})
}

const testDataSourceConfig_timeout = `
data "http" "http_test" {
  url = "%s/timeout"
  request_timeout_ms = 100
}
`

func TestDataSource_http_timeout(t *testing.T) {
	testHttpMock := setUpMockHttpServer()

	defer testHttpMock.server.Close()

	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testDataSourceConfig_timeout, testHttpMock.server.URL),
				ExpectError: regexp.MustCompile("context deadline exceeded"),
			},
		},
	})
}

func TestDataSource_http404(t *testing.T) {
	testHttpMock := setUpMockHttpServer()

	defer testHttpMock.server.Close()

	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testDataSourceConfig_basic, testHttpMock.server.URL, 404),
				ExpectError: regexp.MustCompile("HTTP request error. Response code: 404"),
			},
		},
	})
}

const testDataSourceConfig_basic_error_with_body = `
data "http" "http_test" {
  url = "%s/errorwithbody"
}

output "response_body" {
  value = data.http.http_test.response_body
}

output "response_headers" {
  value = data.http.http_test.response_headers
}
`

func TestDataSource_httperrorwithbody(t *testing.T) {
	testHttpMock := setUpMockHttpServer()

	defer testHttpMock.server.Close()

	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testDataSourceConfig_basic_error_with_body, testHttpMock.server.URL),
				ExpectError: regexp.MustCompile("HTTP request error. Response code: 500,  Error Response body: ruh-roh"),
			},
		},
	})
}

const testDataSourceConfig_withHeaders = `
data "http" "http_test" {
  url = "%s/restricted/meta_%d.txt"

  request_headers = {
    "Authorization" = "Zm9vOmJhcg=="
  }
}

output "response_body" {
  value = data.http.http_test.response_body
}
`

func TestDataSource_withHeaders200(t *testing.T) {
	testHttpMock := setUpMockHttpServer()

	defer testHttpMock.server.Close()

	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testDataSourceConfig_withHeaders, testHttpMock.server.URL, 200),
				Check: func(s *terraform.State) error {
					_, ok := s.RootModule().Resources["data.http.http_test"]
					if !ok {
						return fmt.Errorf("missing data resource")
					}

					outputs := s.RootModule().Outputs

					if outputs["response_body"].Value != "1.0.0" {
						return fmt.Errorf(
							`'response_body' output is %s; want '1.0.0'`,
							outputs["response_body"].Value,
						)
					}

					return nil
				},
			},
		},
	})
}

const testDataSourceConfig_utf8 = `
data "http" "http_test" {
  url = "%s/utf-8/meta_%d.txt"
}

output "status_code" {
	value = data.http.http_test.status_code
}

output "response_body" {
  value = "${data.http.http_test.response_body}"
}
`

func TestDataSource_utf8(t *testing.T) {
	testHttpMock := setUpMockHttpServer()

	defer testHttpMock.server.Close()

	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testDataSourceConfig_utf8, testHttpMock.server.URL, 200),
				Check: func(s *terraform.State) error {
					_, ok := s.RootModule().Resources["data.http.http_test"]
					if !ok {
						return fmt.Errorf("missing data resource")
					}

					outputs := s.RootModule().Outputs

					v, ok := outputs["status_code"].Value.(string)
					if !ok {
						return fmt.Errorf("missing status code resource")
					}
					status_code, err := strconv.Atoi(v)
					if err != nil {
						return fmt.Errorf("error converting status code resource")
					}
					if status_code != http.StatusOK {
						return fmt.Errorf(
							`'status_code' output is %d; want '200'`,
							status_code,
						)
					}

					if outputs["response_body"].Value != "1.0.0" {
						return fmt.Errorf(
							`'response_body' output is %s; want '1.0.0'`,
							outputs["response_body"].Value,
						)
					}

					return nil
				},
			},
		},
	})
}

const testDataSourceConfig_utf16 = `
data "http" "http_test" {
  url = "%s/utf-16/meta_%d.txt"
}

output "response_body" {
  value = "${data.http.http_test.response_body}"
}
`

func TestDataSource_utf16(t *testing.T) {
	testHttpMock := setUpMockHttpServer()

	defer testHttpMock.server.Close()

	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testDataSourceConfig_utf16, testHttpMock.server.URL, 200),
				// This should now be a warning, but unsure how to test for it...
				//ExpectWarning: regexp.MustCompile("Content-Type is not a text type. Got: application/json; charset=UTF-16"),
			},
		},
	})
}

const testDataSourceConfig_verb = `
data "http" "http_test" {
  url = "%s/post"
  method = "GET"
  request_headers = {
    content-type = "application/json"
  }  
  request_body = jsonencode({
    foo = "bar",
    bar = "bar"
  })  
}

output "response_body" {
  value = "${data.http.http_test.response_body}"
}
`

func TestDataSource_verb(t *testing.T) {
	testHttpMock := setUpMockHttpServer()

	defer testHttpMock.server.Close()

	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testDataSourceConfig_verb, testHttpMock.server.URL),
				ExpectError: regexp.MustCompile("HTTP request error. Response code: 405"),
			},
		},
	})
}

const testDataSourceConfig_post = `
data "http" "http_test" {
  url = "%s/post"
  method = "POST"
  request_headers = {
    content-type = "application/json"
  }  
  request_body = jsonencode({
    foo = "bar",
    bar = "bar"
  })  
}

output "response_body" {
  value = "${data.http.http_test.response_body}"
}
`

func TestDataSource_post(t *testing.T) {
	testHttpMock := setUpMockHttpServer()

	defer testHttpMock.server.Close()

	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testDataSourceConfig_post, testHttpMock.server.URL),
				// This should now be a warning, but unsure how to test for it...
				//ExpectWarning: regexp.MustCompile("Content-Type is not a text type. Got: application/json; charset=UTF-16"),

				Check: func(s *terraform.State) error {
					_, ok := s.RootModule().Resources["data.http.http_test"]
					if !ok {
						return fmt.Errorf("missing data resource")
					}

					outputs := s.RootModule().Outputs

					if outputs["response_body"].Value != "1.0.0" {
						return fmt.Errorf(
							`'response_body' output is %s; want '1.0.0'`,
							outputs["response_body"].Value,
						)
					}

					return nil
				},
			},
		},
	})
}

const testDataSourceConfig_form_post = `
data "http" "http_test" {
  url = "%s/formpost"

  method = "POST"
  request_headers = {
    content-type = "application/x-www-form-urlencoded"
  }  
  request_body = "foo=bar&bar=bar"
}

output "response_body" {
  value = "${data.http.http_test.response_body}"
}
`

func TestDataSource_form_post(t *testing.T) {
	testHttpMock := setUpMockHttpServer()

	defer testHttpMock.server.Close()

	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testDataSourceConfig_form_post, testHttpMock.server.URL),
				// This should now be a warning, but unsure how to test for it...
				//ExpectWarning: regexp.MustCompile("Content-Type is not a text type. Got: application/json; charset=UTF-16"),

				Check: func(s *terraform.State) error {
					_, ok := s.RootModule().Resources["data.http.http_test"]
					if !ok {
						return fmt.Errorf("missing data resource")
					}

					outputs := s.RootModule().Outputs

					if outputs["response_body"].Value != "1.0.0" {
						return fmt.Errorf(
							`'response_body' output is %s; want '1.0.0'`,
							outputs["response_body"].Value,
						)
					}

					return nil
				},
			},
		},
	})
}

const (

	// X509v3 extensions:
	// X509v3 Key Usage: critical
	// 	Certificate Sign, CRL Sign
	// X509v3 Basic Constraints: critical
	// 	CA:TRUE, pathlen:0
	// X509v3 Subject Key Identifier:
	// 	B7:BA:B0:02:A1:E7:BE:34:C6:C1:05:5C:66:78:E5:BB:53:5D:A1:54
	caCert = `-----BEGIN CERTIFICATE-----\nMIIDfjCCAmagAwIBAgIBATANBgkqhkiG9w0BAQsFADBQMQswCQYDVQQGEwJVUzEP\nMA0GA1UECgwGR29vZ2xlMRMwEQYDVQQLDApFbnRlcnByaXNlMRswGQYDVQQDDBJF\nbnRlcnByaXNlIFJvb3QgQ0EwHhcNMjIwNTI2MjI1NjI1WhcNMzIwNTI1MjI1NjI1\nWjBQMQswCQYDVQQGEwJVUzEPMA0GA1UECgwGR29vZ2xlMRMwEQYDVQQLDApFbnRl\ncnByaXNlMRswGQYDVQQDDBJFbnRlcnByaXNlIFJvb3QgQ0EwggEiMA0GCSqGSIb3\nDQEBAQUAA4IBDwAwggEKAoIBAQDQ+bpQHaJQWggUoPXVf/7xqLsOPH5D83MDU8l1\ndamAGe7yhZp4leU5hC6KUs8hqA9NQ67WUEOmzS00D01DfsKHsJo9mbufaHN3ij4l\nIDMqJJOgOTvdz3cEfAFhq2syEjqk1ghEwGJhZ2tdh0LORwLUYfoXgYs0w6m6++z2\nkvLZ4G0EgraqsmpjfFXBRDN/OsBdy68jmZBS9LFo/KZu0KH3/ZKAih39SFNOtKNx\n9gXvF7PJ+KOnWEAjuXpQJDNBF7S9WBDEBaIR+qdY5B5oGzzkcGuOlWbqUWfAXMyb\n7WrWODMf8FS8JHVTAN0eLVmnP0Ibqzvtk48oc7NgTg24O5ZzAgMBAAGjYzBhMA4G\nA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBS7tGUlTrMJ\nafcmmZwFqGq5ktD4ZTAfBgNVHSMEGDAWgBS7tGUlTrMJafcmmZwFqGq5ktD4ZTAN\nBgkqhkiG9w0BAQsFAAOCAQEAnE5jWnXIa6hGJKrIUVHhCxdJ4CDpayKiULhjPipR\nTZxzOlbhJHM/eYfH8VtbHRLkZrG/u3uiGWinLliXWHR9cB+BRgdVOMeehDMKP6o0\nWoACUpyLsbiPKdTUEXzXg4MwLwv23vf2xWvp4TousLA8++rIk1qeFW0NSAUGzYfs\nsKpBP2BdJVXcveAEpfwmbnQTZ0OzceA4RFdu4hMZhOwXgK2WZh4fMhyRBh67ueFh\nkVEGN4UUVAP4r/pJEtf4lLE468yPdD+w0yM0xDVAb9DrMyr3h4FwxHalZdgOeRSq\nATCK3GKv5lwmr/NPdg/cPdG5p/lfWQACwi47XgGi59nYIw==\n-----END CERTIFICATE-----`
	// 	caKey = `-----BEGIN PRIVATE KEY-----
	// MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQDQ+bpQHaJQWggU
	// oPXVf/7xqLsOPH5D83MDU8l1damAGe7yhZp4leU5hC6KUs8hqA9NQ67WUEOmzS00
	// D01DfsKHsJo9mbufaHN3ij4lIDMqJJOgOTvdz3cEfAFhq2syEjqk1ghEwGJhZ2td
	// h0LORwLUYfoXgYs0w6m6++z2kvLZ4G0EgraqsmpjfFXBRDN/OsBdy68jmZBS9LFo
	// /KZu0KH3/ZKAih39SFNOtKNx9gXvF7PJ+KOnWEAjuXpQJDNBF7S9WBDEBaIR+qdY
	// 5B5oGzzkcGuOlWbqUWfAXMyb7WrWODMf8FS8JHVTAN0eLVmnP0Ibqzvtk48oc7Ng
	// Tg24O5ZzAgMBAAECggEAMR5BeIs+l3xR4edjYOdQ2SQ7s0DsvLQAGIwdEgqx6HYv
	// /7j/cdBprHcxKToFjXefAR4jfiQngpE/Srk+A9tLhfEwj8IOo409dp97s+Y5oHIw
	// cLyDIcOdyeQLvxU3gPFf71aPYvmFJjfUuIsOXMW8GIde7R95xNEol9aW/+3SPvtf
	// 3b86gVugWvWbUhGWKWBTW2VQnQVo+MZEy4R3OybFuwxnMasOeLUYQb5RceOEWO3Z
	// m6UVLPd+vrt5uKtCJIPhU+39Vw8WYiSEysvmIT/p9yaC0Nlydaa5sBQ7CwqRLIqu
	// Fw+I6SriwoABXCwRoRbOql+HP6+uXJSBm/1j4IGIoQKBgQD/qA11sonVxCanLgXG
	// GOGFbkvhtJ3IYucTkXh1pI+ZHHcK/frvtL+dpukEOFMNTEmOfJnBA/VVpDDuMmIk
	// pF+5qSYqnAxXn7oOynYVsTc6QC+F59UjA+wdwag92GHH7brWJjsa9vyg8rDtEAOk
	// jVZhbf8lBHMjl3B2PVBZqz/k1QKBgQDRQZ3d1/LW2zJ6ZY9CNR1jsKsDCoEZZrnU
	// lq5beWK5MFBQXeS7v6qptYNgM8VNYsCK2mN1YjqA1JYI0sGtj4XTumWYsaOUNuhE
	// Rp++LOsi5eySkRNOOAJAHB9VRiT+U+rwUwZkxMKWZbekmhJYJe9DcUZ8oeHD+btG
	// b2OMESXSJwKBgCEyczz7SAaoB9ThlwJYLMCkx9mxGGPy48qYsymjirn5BkQ5IqKJ
	// t/ACwnM31SD+7PZBm72ChBLw1SG5DSFw7rUvD7Osu7WNGh3dkGPUtTUtLH6Y0gZP
	// 9hMPGIefV2McrYwtPrOLqtZDbVH7KF3vtG3GWME3yLOwcHwKDir2n79ZAoGAR1pT
	// hVDcekz2EmxNBCtuYQ7d0USkrs+rcAUNYR2r/y+tQyoxE6AQhpvhN02P6opQ00gS
	// f/VFs6ZJnqqW5iK5ZG/7sqxn9eMfIiDe2Y8hgp3aJEQZzCMnCUtNl9s6RArDYr08
	// weGh5Hy8uQDcXnhY9KtMeLUOca/XHvZegGVceyMCgYBPW0VHLEULeHijCOI1qcmg
	// 01z6fU3IEBNle80VO5e3+QRjqZnCdOAj0PHqRhpceFkr+QDqlC+9o2eGlicVSE/s
	// 7Mc467clydBvvkyo7DVtQgivcGdOuVd0kEJVMbC9mcrF4pSfiuhxTqQVmJ+B2QO6
	// hiAA1onD1iT72j0DOayXbw==
	// -----END PRIVATE KEY-----`

	// X509v3 extensions:
	// X509v3 Extended Key Usage:
	// 	TLS Web Server Authentication
	// X509v3 Subject Key Identifier:
	// 	5D:18:9B:10:D0:22:40:96:2B:A2:12:46:BC:C0:99:60:FA:18:F9:0A
	// X509v3 Authority Key Identifier:
	// 	keyid:BB:B4:65:25:4E:B3:09:69:F7:26:99:9C:05:A8:6A:B9:92:D0:F8:65
	// X509v3 Subject Alternative Name:
	// 	IP Address:127.0.0.1, DNS:localhost

	localhostCert = `-----BEGIN CERTIFICATE-----
MIIELzCCAxegAwIBAgIBAjANBgkqhkiG9w0BAQsFADBQMQswCQYDVQQGEwJVUzEP
MA0GA1UECgwGR29vZ2xlMRMwEQYDVQQLDApFbnRlcnByaXNlMRswGQYDVQQDDBJF
bnRlcnByaXNlIFJvb3QgQ0EwHhcNMjIwNTI2MjMwNjIzWhcNMzIwNTI1MjMwNjIz
WjBPMQswCQYDVQQGEwJVUzEPMA0GA1UECgwGR29vZ2xlMRMwEQYDVQQLDApFbnRl
cnByaXNlMRowGAYDVQQDDBFzZXJ2ZXIuZG9tYWluLmNvbTCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBALk1Zr85ztqUagPPBJl/m7g+GBcend+JEdmVa9J3
zP7/MBV+kJymdZ1DWeKdXK3CEOqrH3/vHTsCMyDX6H671LlTnBls6ZdDP10ujCds
AHbTrFUfD9U4QPtYkL0J0PIHjYGHnHdOkeRQuE8tBx1bRgVJMSsYSFiaZDVI5B3A
050I41YlEZc6Fq8NIcLig3j2ycqC9eLaDmrKNayRhXBm+N31S26ni3uJUH3sFn7l
Vt63BGv1o3xbcRv8TRCrLzZb18GbpAG3x5hSbQBn5GJhXDXzeNhdVUE1NPWKtlMs
sJ6XBiARqrgoHtanae9f1xCbMkMn+wdjhviIuk7S4t0yGcsCAwEAAaOCARMwggEP
MA4GA1UdDwEB/wQEAwIHgDAJBgNVHRMEAjAAMBMGA1UdJQQMMAoGCCsGAQUFBwMB
MB0GA1UdDgQWBBRdGJsQ0CJAliuiEka8wJlg+hj5CjAfBgNVHSMEGDAWgBS7tGUl
TrMJafcmmZwFqGq5ktD4ZTBFBggrBgEFBQcBAQQ5MDcwNQYIKwYBBQUHMAKGKWh0
dHA6Ly9wa2kuZXNvZGVtb2FwcDIuY29tL2NhL3Jvb3QtY2EuY2VyMDoGA1UdHwQz
MDEwL6AtoCuGKWh0dHA6Ly9wa2kuZXNvZGVtb2FwcDIuY29tL2NhL3Jvb3QtY2Eu
Y3JsMBoGA1UdEQQTMBGHBH8AAAGCCWxvY2FsaG9zdDANBgkqhkiG9w0BAQsFAAOC
AQEAqzmJ1CnygTMKvxkIbQKiYtBLlDAA7tJO+55mcivtk9RxT+PBYEnEV1e+IqHC
4x+YLESVCHQk1Eia3dJy3fEAylRxzICxMrg+4EA2nuDSgVH8CeD74kUEsEzSw8eY
SQH2RoOxly+32+lkw2oF1a38+elMvU/0Z2w1F2CW5sVj1kieG4vzn0rqvmNauU04
r1m6rnN1yq6rtuQ16Y9SQb1VGXs9ijNKMICGcBONqebYCV7nGPCitH5yQsIXy8ns
DoHHMPWPLdj8n6w9drtKeBN4IHooizAuv43HbWapVgVAKsLxffo1B7DcgQDB0MYn
Jh1J6CU1KiOziz1rQrambz66Jw==
-----END CERTIFICATE-----`

	localhostKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAuTVmvznO2pRqA88EmX+buD4YFx6d34kR2ZVr0nfM/v8wFX6Q
nKZ1nUNZ4p1crcIQ6qsff+8dOwIzINfofrvUuVOcGWzpl0M/XS6MJ2wAdtOsVR8P
1ThA+1iQvQnQ8geNgYecd06R5FC4Ty0HHVtGBUkxKxhIWJpkNUjkHcDTnQjjViUR
lzoWrw0hwuKDePbJyoL14toOaso1rJGFcGb43fVLbqeLe4lQfewWfuVW3rcEa/Wj
fFtxG/xNEKsvNlvXwZukAbfHmFJtAGfkYmFcNfN42F1VQTU09Yq2UyywnpcGIBGq
uCge1qdp71/XEJsyQyf7B2OG+Ii6TtLi3TIZywIDAQABAoIBAD05Cd3snhRjOyhH
Jp4XMMKWxB/gXw+ln+DtI9dPAtTIRnzUeblOzVJPEUd3/Ury++SW7LK9uEvpTj1t
Ic3DCW651MAS4KS/9hI3cN0XNpARKMZ6niE9lz1+6VmUBR38oSpQScimkFOI22RQ
3ik2Is9cgoRcYo3ne3ihv8aWF12xIbcP9MjEGh/uTaRZP/PMv3EmDL/5VCN91laB
9ZtTgeaMjTt53fNrz/PKLFA4qTuXkHlnbxMUp1uW+kHpwH8XY/lhYjjlSjOPQg5y
QsrObOqOExnfHa7Rds0FT05XsbV5TmRszF2I9fIcnUp0uwIyQ6iTTQzJIufkADQ7
Jg7HgwECgYEA8a6yG66GvA40JAFp+UK5cl7qKo7YJ2zLPxtRbFW7nZ22jW26u3Fq
FPhBxSHtn4UfsQTqqpXp04KASS7UyUt6ri2raEzUdSaX/uCMtB9fm+5ixpUJOLO9
Q9HCkJNFchktCeYlet89MJ/zkl2JSyTGez32JqHSfNRPQ5ds6tIr4QUCgYEAxC48
Xlx+wjDRpW3SG6QgwPImPjYQCK8hcltEfhltX9veFUJCWYZqEGtD10phS1dQF9WO
FhTpLFgWCaNlrywD6jsDMS+xv8+VfsooIAlThuKpTZWMLQasDwfYc2n7qcHJOwYs
pnX4sQNvN4iMvrL0CADDiSVmH99SU0NdVtauSI8CgYEAjf7vBFaZMNpDhjgSdHHg
lTLw8Ao3M4q3K5+4Sidg8O0duaCTyteK1UE7G0Cg5U2I3i+eVJV56VxOVTEfshkX
vkh04fXqCd6gBQ8XfCjGus3n2PbtkRQBilwurVTpw2zJSnye3r9Uq0H/EKrGJJE5
0GUKP45qJg9zdqn8Q0cyoqUCgYA8Yi7arIWnp/cfgCoHsAEU4nO6+lD9G0qkNEtk
tNbhhn9Y88gQXjsPSrTa8133HqzcaTMOwOj0aTh/RvfpbxbVZcyZuyBu9aoCGJ85
HSXEgsexxbIbuc4D4lpRS/HWUntp24Cqy+z8Lx5wbWtE1zgdrn6BHC3O6aIhVr7I
F9QVKQKBgQDBAbYkyc41baRJm6oKIfDUfjNiYdggsKm3OmL4qs+snqKUHv0WF/aI
U2gGbAvI7b/PKnMs4zct0JoFkZ2MN369PEhcpKnQ6iM8Le+42r8OPhpOs/M0uCeP
x1mpmqYktTV/Y7QcHQsrVaZ+4WWYO0+Erp1mYrm7EUOhvGOGrp/6mw==
-----END RSA PRIVATE KEY-----`

	// X509v3 extensions:
	// X509v3 Extended Key Usage:
	// 	TLS Web Client Authentication
	// X509v3 Authority Key Identifier:
	// 	keyid:B7:BA:B0:02:A1:E7:BE:34:C6:C1:05:5C:66:78:E5:BB:53:5D:A1:54
	clientCert = `-----BEGIN CERTIFICATE-----\nMIIEDzCCAvegAwIBAgIBAzANBgkqhkiG9w0BAQsFADBQMQswCQYDVQQGEwJVUzEP\nMA0GA1UECgwGR29vZ2xlMRMwEQYDVQQLDApFbnRlcnByaXNlMRswGQYDVQQDDBJF\nbnRlcnByaXNlIFJvb3QgQ0EwHhcNMjIwNTI2MjMxNDE5WhcNMzIwNTI1MjMxNDE5\nWjBNMQswCQYDVQQHDAJVUzEPMA0GA1UECgwGR29vZ2xlMRMwEQYDVQQLDApFbnRl\ncnByaXNlMRgwFgYDVQQDDA91c2VyQGRvbWFpbi5jb20wggEiMA0GCSqGSIb3DQEB\nAQUAA4IBDwAwggEKAoIBAQC87w2DG1FqxHEidfPmhXsnqBNmgp3Rntyo7lJNtL2p\n1N49R88TiOKDNHsxAW4pT8E/cwWKB18SGMgpPEhC6vT7KOVzwUb/ozslfV3JiA4l\n8JU2jYkwXcgUCo1vZGlAcz3ciqfk+pQN1NFy6UuYNN45HNvoFcPgr+3mso+ODGXr\n1rkg/RCfGiMUK8qiyeGq0P7VkavFNsr09Mcx4cxrA7j9TOtTHQg2PReGKihCAlpE\nJHHtmrMRGUun/4i3E9tv53qyv85M9QXXbVN4kZrAH4jCljV8M1StPX+9e0C9A/J1\nvi9dtJ274+NL6dSOOvHv6FH+9bbHaTlmqM8MpyRa6Cl5AgMBAAGjgfYwgfMwDgYD\nVR0PAQH/BAQDAgeAMAkGA1UdEwQCMAAwEwYDVR0lBAwwCgYIKwYBBQUHAwIwHQYD\nVR0OBBYEFDg0Le8zwIgDv537avLRXuIQKTTZMB8GA1UdIwQYMBaAFLu0ZSVOswlp\n9yaZnAWoarmS0PhlMEUGCCsGAQUFBwEBBDkwNzA1BggrBgEFBQcwAoYpaHR0cDov\nL3BraS5lc29kZW1vYXBwMi5jb20vY2Evcm9vdC1jYS5jZXIwOgYDVR0fBDMwMTAv\noC2gK4YpaHR0cDovL3BraS5lc29kZW1vYXBwMi5jb20vY2Evcm9vdC1jYS5jcmww\nDQYJKoZIhvcNAQELBQADggEBADsc0BRMlI2wm4RcAOxK3GKrROAY9Lk/LglGqC63\nbGq0fVq+yu8H9fkQSSFVSaIaXtSYDl+fj7bOUkrJqzRy9hWHDoTTCUF+CNtiP7Lw\nE4jPSp2MllDh5S09/vQgd2k0ahejySSVgBU40klnwQovTWrA7sVG07eBxJph8IXc\nd2iqLbLLh3pnYUwE6VMDmspPVT8LsdNQuHoHsLVIb/zK+OkQM6NX/30Ri/XE41G1\n+h7c49t3eJ4YdYMnuXPS8QDuuRvFsunh00sejZtfvliJcFuCcRLPNw6hrbgIgtB9\nVeoUnh6o5hugfzXc/YqRLYW4zLgEoDnGUv+rf0D8CoYiwDQ=\n-----END CERTIFICATE-----`
	clientKey  = `-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC87w2DG1FqxHEi\ndfPmhXsnqBNmgp3Rntyo7lJNtL2p1N49R88TiOKDNHsxAW4pT8E/cwWKB18SGMgp\nPEhC6vT7KOVzwUb/ozslfV3JiA4l8JU2jYkwXcgUCo1vZGlAcz3ciqfk+pQN1NFy\n6UuYNN45HNvoFcPgr+3mso+ODGXr1rkg/RCfGiMUK8qiyeGq0P7VkavFNsr09Mcx\n4cxrA7j9TOtTHQg2PReGKihCAlpEJHHtmrMRGUun/4i3E9tv53qyv85M9QXXbVN4\nkZrAH4jCljV8M1StPX+9e0C9A/J1vi9dtJ274+NL6dSOOvHv6FH+9bbHaTlmqM8M\npyRa6Cl5AgMBAAECggEAZFUCtPQl6XAGsIk5C9soyqd8Hf0ROEeH4QImnPN1oSHV\nH2/p7PLNb2XIYf7jdHbRJhO8Bk/h0edtLFDCAx9pF5PhPfaO8KTLfR41Vxe0g7te\nUgkZqKC05sev0k7dggdw+5R6kqPrSekRjVeM+Hhi5quHsJkWW1SyHsgGaiX1XibP\n5lR3xQKQRKkoHBGbxx2RxQ9grMg9vFzi8EGM63EPIyF4x7rV+pXVl+pHnCFTceV/\nz8a0ZoqCbGxMiUPVlwwIdT2ZRMZC6ChIbgfd+ZKsZU3dOaqVA+qaKbu63AJK1Qon\n/m1NgMYhOkIaiMEt2OnkdmxdzlSO/PV/Bx2Hue4KAQKBgQDvCNpncfMk8DACtwFC\nqnX1e6xOa1nVGFGjoh7whRZXrhMZcfTn9pE3SoCxKb4fxsYngoVlY4zwhyn10XqC\nhH1ydU+cl7HV4kp8rnjGLgtcc759G3NBU2HrfKHED/HLGTMwMukT/L772C6RpQtY\ncGfc9eEo8029Sb3eOwpQ0UKwQQKBgQDKV+RMcJCTANmBT+dth3C1DONfirZ+YudF\nvg3knz8iYzTvsq+zZ0xAKvNmleq2f2Q8eixcudF2gbIttTfKOFmgEvdSXzYzFKV+\naJZ67HIt0stfnOn0MEsmTEu5TbXt513vgKuotqpukNuKJtnMZhM48c0A6RcoRsDZ\n/zibMnQrOQKBgQCh/CHlkDbxhUND07iq8NFXNiQiUGVkH0LT3P2SiN4HNRQEXlFV\nEKaADaEAbgVFi3KlO7Iib0AHj9FDoF2hLR/F/PGicLo2807/B00ZIALa+CTSq1OD\npXnqF1+YeiWlOMKTmyyQOutBx9JnKK1zlVkNSCL5mUfJSru8ac4nzmefAQKBgQDH\nSvwkQbZT48FW+QFjQsRCvpfwUWpfX0CU05VReXuwfe/0qpUdaX+Tr/oeL0iHST/L\nxTWOesKRKzr4hAWYGhpEbInGStrSQuKhd5fHKL1o3rbKzH0tsqdB6GGo+J5Y3MoL\njDsGqCuDTQ++qXdZN6x1KMuWuv3BALcPv63cRjxfGQKBgEMTcTiq3uUNJtx6idK8\n4M+Qe53z1oz1tz2/a6YykH1Wh6uwYYiL2gqIlTYKUXcrCvUn7fQcumm2OPy/hXXP\nUN5qfcT1wiP1cPDKItsKfcu39Z8lVqUglW0MOIFDST30NYuDAk0ni+ldhi3wfD5V\nmS8QVY9xAlaeLLXsZuQIoOpL\n-----END PRIVATE KEY-----`
)
const testDataSourceConfig_mtls = `
data "http" "http_test" {
  url = "%s/get"
  ca = "%s"
  client_crt = "%s"
  client_key = "%s" 
}

output "response_body" {
  value = "${data.http.http_test.response_body}"
}
`

func TestDataSource_mtls(t *testing.T) {
	server := httptest.NewUnstartedServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				if r.URL.Path == "/get" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("1.0.0"))
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			},
		),
	)

	formatCaCert := strings.Replace(caCert, `\n`, "\n", -1)
	clientCaCertPool := x509.NewCertPool()
	ok := clientCaCertPool.AppendCertsFromPEM([]byte(formatCaCert))
	if !ok {
		panic(errors.New("Error loading root cert: "))
	}
	privBlock, _ := pem.Decode([]byte(localhostKey))
	key, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	if err != nil {
		panic(fmt.Errorf("Error getting server private key : %v", err))
	}

	pubBlock, _ := pem.Decode([]byte(localhostCert))
	cert, err := x509.ParseCertificate(pubBlock.Bytes)
	if err != nil {
		panic(fmt.Errorf("Error getting server public cert : %v", err))
	}

	tlsConfig := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  clientCaCertPool,
		Certificates: []tls.Certificate{
			{
				PrivateKey:  key,
				Certificate: [][]byte{cert.Raw},
			},
		},
	}

	tlsConfig.BuildNameToCertificate()

	server.TLS = tlsConfig
	server.StartTLS()

	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testDataSourceConfig_mtls, server.URL, caCert, clientCert, clientKey),
				Check: func(s *terraform.State) error {
					_, ok := s.RootModule().Resources["data.http.http_test"]
					if !ok {
						return fmt.Errorf("missing data resource")
					}

					outputs := s.RootModule().Outputs

					if outputs["response_body"].Value != "1.0.0" {
						return fmt.Errorf(
							`'response_body' output is %s; want '1.0.0'`,
							outputs["response_body"].Value,
						)
					}

					return nil
				},
			},
		},
	})
}

func setUpMockHttpServer() *TestHttpMock {
	Server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			w.Header().Set("Content-Type", "text/plain")
			w.Header().Add("X-Single", "foobar")
			w.Header().Add("X-Double", "1")
			w.Header().Add("X-Double", "2")
			if r.URL.Path == "/meta_200.txt" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("1.0.0"))
			} else if r.URL.Path == "/restricted/meta_200.txt" {
				if r.Header.Get("Authorization") == "Zm9vOmJhcg==" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("1.0.0"))
				} else {
					w.WriteHeader(http.StatusForbidden)
				}
			} else if r.URL.Path == "/utf-8/meta_200.txt" {
				w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("1.0.0"))
			} else if r.URL.Path == "/timeout" {
				w.WriteHeader(http.StatusOK)
				time.Sleep(time.Duration(200) * time.Millisecond)
				w.Write([]byte("1.0.0"))
			} else if r.URL.Path == "/utf-16/meta_200.txt" {
				w.Header().Set("Content-Type", "application/json; charset=UTF-16")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("\"1.0.0\""))
			} else if r.URL.Path == "/x509/cert.pem" {
				w.Header().Set("Content-Type", "application/x-x509-ca-cert")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("pem"))
			} else if r.URL.Path == "/meta_404.txt" {
				w.WriteHeader(http.StatusNotFound)
			} else if r.URL.Path == "/formpost" && r.Method == http.MethodPost {
				defer r.Body.Close()
				err := r.ParseForm()
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
				}
				if r.FormValue("foo") != "bar" || r.FormValue("bar") != "bar" {
					w.WriteHeader(http.StatusInternalServerError)
				}

				w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("1.0.0"))
			} else if r.URL.Path == "/post" && r.Method == http.MethodPost {
				defer r.Body.Close()
				jsonMap := make(map[string](string))
				err := json.NewDecoder(r.Body).Decode(&jsonMap)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
				}
				if jsonMap["foo"] != "bar" || jsonMap["bar"] != "bar" {
					w.WriteHeader(http.StatusInternalServerError)
				}
				w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("1.0.0"))
			} else if r.URL.Path == "/post" && r.Method == http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else if r.URL.Path == "/errorwithbody" {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("ruh-roh"))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}),
	)

	return &TestHttpMock{
		server: Server,
	}
}

const testDataSourceConfig_skip_verify_tls_fail = `
data "http" "http_test" {
  url = "%s/get"
  ca = "%s"
}

output "response_body" {
  value = "${data.http.http_test.response_body}"
}
`

func TestDataSource_skip_tls_verify_fail(t *testing.T) {
	testHttpMock := setUpMockTLSHttpServer()
	defer testHttpMock.server.Close()
	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testDataSourceConfig_skip_verify_tls_fail, testHttpMock.server.URL, caCert),
				ExpectError: regexp.MustCompile("x509: certificate signed by unknown authority"),
			},
		},
	})
}

const testDataSourceConfig_skip_verify_tls_success = `
data "http" "http_test" {
  url = "%s/get"
  ca = "%s"
  insecure_skip_verify = true
}

output "response_body" {
  value = "${data.http.http_test.response_body}"
}
`

func TestDataSource_skip_tls_verify_success(t *testing.T) {
	testHttpMock := setUpMockTLSHttpServer()

	defer testHttpMock.server.Close()

	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testDataSourceConfig_skip_verify_tls_success, testHttpMock.server.URL, caCert),
				Check: func(s *terraform.State) error {
					_, ok := s.RootModule().Resources["data.http.http_test"]
					if !ok {
						return fmt.Errorf("missing data resource")
					}

					outputs := s.RootModule().Outputs

					if outputs["response_body"].Value != "1.0.0" {
						return fmt.Errorf(
							`'response_body' output is %s; want '1.0.0'`,
							outputs["response_body"].Value,
						)
					}

					return nil
				},
			},
		},
	})
}

const testDataSourceConfig_sni_fail = `
data "http" "http_test" {
  url = "%s/get"
  ca = "%s"
  sni = "foo"
}

output "response_body" {
  value = "${data.http.http_test.response_body}"
}
`

func TestDataSource_sni_fail(t *testing.T) {
	server := httptest.NewUnstartedServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				if r.URL.Path == "/get" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("1.0.0"))
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			},
		),
	)

	privBlock, _ := pem.Decode([]byte(localhostKey))
	key, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	if err != nil {
		panic(fmt.Errorf("Error getting server private key : %v", err))
	}

	pubBlock, _ := pem.Decode([]byte(localhostCert))
	cert, err := x509.ParseCertificate(pubBlock.Bytes)
	if err != nil {
		panic(fmt.Errorf("Error getting server public cert : %v", err))
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{
			{
				PrivateKey:  key,
				Certificate: [][]byte{cert.Raw},
			},
		},
	}

	tlsConfig.BuildNameToCertificate()

	server.TLS = tlsConfig
	server.StartTLS()

	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testDataSourceConfig_sni_fail, server.URL, caCert),
				ExpectError: regexp.MustCompile("x509: certificate is valid for localhost, not foo"),
			},
		},
	})
}

const testDataSourceConfig_sni_success = `
data "http" "http_test" {
  url = "%s/get"
  ca = "%s"
  sni = "localhost"
}

output "response_body" {
  value = "${data.http.http_test.response_body}"
}
`

func TestDataSource_sni_success(t *testing.T) {
	server := httptest.NewUnstartedServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				if r.URL.Path == "/get" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("1.0.0"))
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			},
		),
	)

	privBlock, _ := pem.Decode([]byte(localhostKey))
	key, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	if err != nil {
		panic(fmt.Errorf("Error getting server private key : %v", err))
	}

	pubBlock, _ := pem.Decode([]byte(localhostCert))
	cert, err := x509.ParseCertificate(pubBlock.Bytes)
	if err != nil {
		panic(fmt.Errorf("Error getting server public cert : %v", err))
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{
			{
				PrivateKey:  key,
				Certificate: [][]byte{cert.Raw},
			},
		},
	}

	tlsConfig.BuildNameToCertificate()

	server.TLS = tlsConfig
	server.StartTLS()

	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testDataSourceConfig_sni_success, server.URL, caCert),
				Check: func(s *terraform.State) error {
					_, ok := s.RootModule().Resources["data.http.http_test"]
					if !ok {
						return fmt.Errorf("missing data resource")
					}

					outputs := s.RootModule().Outputs

					if outputs["response_body"].Value != "1.0.0" {
						return fmt.Errorf(
							`'response_body' output is %s; want '1.0.0'`,
							outputs["response_body"].Value,
						)
					}

					return nil
				},
			},
		},
	})
}

// the default httptest.NewTLSServer uses go's built in ca and certs..
//
//	we're only using this for simple TLS testing.
//	The tests that verify SNI uses the ca specified in this file and the tls server
//	is started from scratch using httptest.NewUnstartedServer
func setUpMockTLSHttpServer() *TestHttpMock {
	Server := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			w.Header().Set("Content-Type", "text/plain")
			if r.URL.Path == "/get" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("1.0.0"))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}),
	)

	return &TestHttpMock{
		server: Server,
	}
}
