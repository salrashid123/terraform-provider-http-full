package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

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

output "body" {
  value = data.http.http_test.body
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

output "body" {
  value = data.http.http_test.body
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

					if outputs["body"].Value != "1.0.0" {
						return fmt.Errorf(
							`'body' output is %s; want '1.0.0'`,
							outputs["body"].Value,
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

output "body" {
  value = "${data.http.http_test.body}"
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

					if outputs["body"].Value != "1.0.0" {
						return fmt.Errorf(
							`'body' output is %s; want '1.0.0'`,
							outputs["body"].Value,
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

output "body" {
  value = "${data.http.http_test.body}"
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

const testDataSourceConfig_post = `
data "http" "http_test" {
  url = "%s/post"
  request_headers = {
    content-type = "application/json"
  }  
  request_body = {
    foo = "bar"
    bar = "bar"
  }  
}

output "body" {
  value = "${data.http.http_test.body}"
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

					if outputs["body"].Value != "1.0.0" {
						return fmt.Errorf(
							`'body' output is %s; want '1.0.0'`,
							outputs["body"].Value,
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
  request_headers = {
    content-type = "application/x-www-form-urlencoded"
  }  
  request_body = {
    body = "foo=bar&bar=bar"
  }  
}

output "body" {
  value = "${data.http.http_test.body}"
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

					if outputs["body"].Value != "1.0.0" {
						return fmt.Errorf(
							`'body' output is %s; want '1.0.0'`,
							outputs["body"].Value,
						)
					}

					return nil
				},
			},
		},
	})
}

// TODO:  i don't know how to do mTLS with https://pkg.go.dev/net/http/httptest#NewTLSServer
// The following only does TLS even with the client_certs set
// net/http/internal/testcert.go
const localhostCert = `-----BEGIN CERTIFICATE-----\nMIICEzCCAXygAwIBAgIQMIMChMLGrR+QvmQvpwAU6zANBgkqhkiG9w0BAQsFADAS\nMRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw\nMDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCB\niQKBgQDuLnQAI3mDgey3VBzWnB2L39JUU4txjeVE6myuDqkM/uGlfjb9SjY1bIw4\niA5sBBZzHi3z0h1YV8QPuxEbi4nW91IJm2gsvvZhIrCHS3l6afab4pZBl2+XsDul\nrKBxKKtD1rGxlG4LjncdabFn9gvLZad2bSysqz/qTAUStTvqJQIDAQABo2gwZjAO\nBgNVHQ8BAf8EBAMCAqQwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDwYDVR0TAQH/BAUw\nAwEB/zAuBgNVHREEJzAlggtleGFtcGxlLmNvbYcEfwAAAYcQAAAAAAAAAAAAAAAA\nAAAAATANBgkqhkiG9w0BAQsFAAOBgQCEcetwO59EWk7WiJsG4x8SY+UIAA+flUI9\ntyC4lNhbcF2Idq9greZwbYCqTTTr2XiRNSMLCOjKyI7ukPoPjo16ocHj+P3vZGfs\nh1fIw3cSS2OolhloGw/XM6RWPWtPAlGykKLciQrBru5NAPvCMsb/I1DAceTiotQM\nfblo6RBxUQ==\n-----END CERTIFICATE-----`
const localhostKey = `-----BEGIN PRIVATE KEY-----\nMIICXgIBAAKBgQDuLnQAI3mDgey3VBzWnB2L39JUU4txjeVE6myuDqkM/uGlfjb9\nSjY1bIw4iA5sBBZzHi3z0h1YV8QPuxEbi4nW91IJm2gsvvZhIrCHS3l6afab4pZB\nl2+XsDulrKBxKKtD1rGxlG4LjncdabFn9gvLZad2bSysqz/qTAUStTvqJQIDAQAB\nAoGAGRzwwir7XvBOAy5tM/uV6e+Zf6anZzus1s1Y1ClbjbE6HXbnWWF/wbZGOpet\n3Zm4vD6MXc7jpTLryzTQIvVdfQbRc6+MUVeLKwZatTXtdZrhu+Jk7hx0nTPy8Jcb\nuJqFk541aEw+mMogY/xEcfbWd6IOkp+4xqjlFLBEDytgbIECQQDvH/E6nk+hgN4H\nqzzVtxxr397vWrjrIgPbJpQvBsafG7b0dA4AFjwVbFLmQcj2PprIMmPcQrooz8vp\njy4SHEg1AkEA/v13/5M47K9vCxmb8QeD/asydfsgS5TeuNi8DoUBEmiSJwma7FXY\nfFUtxuvL7XvjwjN5B30pNEbc6Iuyt7y4MQJBAIt21su4b3sjXNueLKH85Q+phy2U\nfQtuUE9txblTu14q3N7gHRZB4ZMhFYyDy8CKrN2cPg/Fvyt0Xlp/DoCzjA0CQQDU\ny2ptGsuSmgUtWj3NM9xuwYPm+Z/F84K6+ARYiZ6PYj013sovGKUFfYAqVXVlxtIX\nqyUBnu3X9ps8ZfjLZO7BAkEAlT4R5Yl6cGhaJQYZHOde3JEMhNRcVFMO8dJDaFeo\nf9Oeos0UUothgiDktdQHxdNEwLjQf7lJJBzV+5OtwswCWA==\n-----END PRIVATE KEY-----`

const testDataSourceConfig_tls = `
data "http" "http_test" {
  url = "%s/get"
  ca = "%s"
}

output "body" {
  value = "${data.http.http_test.body}"
}
`

func TestDataSource_mtls(t *testing.T) {
	testHttpMock := setUpMockTLSHttpServer()

	defer testHttpMock.server.Close()

	resource.UnitTest(t, resource.TestCase{
		Providers: testProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testDataSourceConfig_tls, testHttpMock.server.URL, localhostCert),
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
				var data map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&data)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
				}
				if data["foo"] != "bar" || data["bar"] != "bar" {
					w.WriteHeader(http.StatusInternalServerError)
				}
				w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("1.0.0"))
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
