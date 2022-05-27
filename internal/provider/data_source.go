package provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func validateVerb(val interface{}, key string) (warns []string, errs []error) {
	if v, ok := val.(string); ok {
		switch v {
		case
			http.MethodGet,
			http.MethodPost,
			http.MethodHead,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodPut:
			break
		default:
			errs = append(errs, fmt.Errorf("%s must be GET|POST|HEAD|DELETE|PATCH|PUT, got: %s", key, v))
		}
	} else {
		errs = append(errs, fmt.Errorf("error parsing method"))
	}
	return
}

func dataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceRead,

		Schema: map[string]*schema.Schema{
			"url": {
				Type:     schema.TypeString,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"method": {
				Type:     schema.TypeString,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Default:      http.MethodGet,
				ValidateFunc: validateVerb,
			},

			"request_headers": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"request_body": {
				Type:     schema.TypeString,
				Computed: false,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"body": {
				Type:     schema.TypeString,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"response_headers": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"status_code": {
				Type:     schema.TypeInt,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
			"ca": {
				Type:     schema.TypeString,
				Required: false,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"client_crt": {
				Type:     schema.TypeString,
				Required: false,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"client_key": {
				Type:      schema.TypeString,
				Required:  false,
				Optional:  true,
				Sensitive: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
	url := d.Get("url").(string)
	headers := d.Get("request_headers").(map[string]interface{})

	tlsConfig := &tls.Config{}

	castr, ok := d.GetOk("ca")
	if ok {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(castr.(string)))
		tlsConfig.RootCAs = caCertPool
	}

	client_crt, ok := d.GetOk("client_crt")
	if ok {
		client_key, ok := d.GetOk("client_key")
		if !ok {
			return append(diags, diag.Errorf("Both client_crt and client_key must be specified")...)
		}
		clientCerts, err := tls.X509KeyPair(
			[]byte(client_crt.(string)),
			[]byte(client_key.(string)),
		)
		if err != nil {
			return append(diags, diag.Errorf("Error loading client certificates: %s", err)...)
		}
		tlsConfig.Certificates = []tls.Certificate{clientCerts}
	}

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	client := &http.Client{Transport: tr}

	verb := http.MethodGet

	method_override, ok := d.GetOk("method")
	if ok {
		if verb, ok = method_override.(string); !ok {
			return append(diags, diag.Errorf("Error overring verb")...)
		}
	}

	var body io.Reader
	b, ok := d.GetOk("request_body")
	if ok {
		verb = http.MethodPost
		if method_override != nil {
			if verb, ok = method_override.(string); !ok {
				return append(diags, diag.Errorf("Error overring verb")...)
			}
		}
		body = bytes.NewReader([]byte(b.(string)))
	}

	req, err := http.NewRequestWithContext(ctx, verb, url, body)
	if err != nil {
		return append(diags, diag.Errorf("Error creating request: %s", err)...)
	}

	for name, value := range headers {
		req.Header.Set(name, value.(string))
	}

	resp, err := client.Do(req)
	if err != nil {
		return append(diags, diag.Errorf("Error making request: %s", err)...)
	}

	defer resp.Body.Close()

	// TODO, check if the response code is valid for the verb sent in...

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {

		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return append(diags, diag.Errorf("HTTP request error. Response code: %d", resp.StatusCode)...)
		}
		return append(diags, diag.Errorf("HTTP request error. Response code: %d,  Error Response body: %s", resp.StatusCode, string(bytes))...)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" || isContentTypeText(contentType) == false {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("Content-Type is not recognized as a text type, got %q", contentType),
			Detail:   "If the content is binary data, Terraform may not properly handle the contents of the response.",
		})
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return append(diags, diag.FromErr(err)...)
	}

	responseHeaders := make(map[string]string)
	for k, v := range resp.Header {
		// Concatenate according to RFC2616
		// cf. https://www.w3.org/Protocols/rfc2616/rfc2616-sec4.html#sec4.2
		responseHeaders[k] = strings.Join(v, ", ")
	}

	if err = d.Set("status_code", resp.StatusCode); err != nil {
		return append(diags, diag.Errorf("Error setting HTTP status_code: %s", err)...)
	}

	if err = d.Set("response_headers", responseHeaders); err != nil {
		return append(diags, diag.Errorf("Error setting HTTP response headers: %s", err)...)
	}

	if err = d.Set("body", string(bytes)); err != nil {
		return append(diags, diag.Errorf("Error setting HTTP response body: %s", err)...)
	}

	// set ID as something more stable than time
	d.SetId(url)

	return diags
}

// This is to prevent potential issues w/ binary files
// and generally unprintable characters
// See https://github.com/hashicorp/terraform/pull/3858#issuecomment-156856738
func isContentTypeText(contentType string) bool {

	parsedType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	allowedContentTypes := []*regexp.Regexp{
		regexp.MustCompile("^text/.+"),
		regexp.MustCompile("^application/json$"),
		regexp.MustCompile("^application/samlmetadata\\+xml"),
	}

	for _, r := range allowedContentTypes {
		if r.MatchString(parsedType) {
			charset := strings.ToLower(params["charset"])
			return charset == "" || charset == "utf-8" || charset == "us-ascii"
		}
	}

	return false
}
