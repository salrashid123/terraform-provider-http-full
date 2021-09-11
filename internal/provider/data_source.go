package provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
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

			"request_headers": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"request_body": {
				Type:     schema.TypeMap,
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
	var body io.Reader
	b, ok := d.GetOk("request_body")
	if ok {
		verb = http.MethodPost
		if contentType, ok := headers["content-type"]; ok {
			if strings.Contains(strings.ToLower(contentType.(string)), strings.ToLower("application/x-www-form-urlencoded")) {
				var hdb map[string]string
				jsonData, err := json.Marshal(b)
				if err != nil {
					return append(diags, diag.Errorf("Error marshallng JSON request: %s", err)...)
				}
				err = json.Unmarshal(jsonData, &hdb)
				if err != nil {
					return append(diags, diag.Errorf("Error marshallng JSON request: %s", err)...)
				}
				body = bytes.NewReader([]byte(hdb["body"]))
			} else if strings.Contains(strings.ToLower(contentType.(string)), strings.ToLower("application/json")) {
				jsonData, err := json.Marshal(b)
				if err != nil {
					return append(diags, diag.Errorf("Error marshallng JSON request: %s", err)...)
				}
				body = bytes.NewReader(jsonData)
			}
		} else {
			jsonData, err := json.Marshal(b)
			if err != nil {
				return append(diags, diag.Errorf("Error marshallng JSON request: %s", err)...)
			}
			body = bytes.NewReader(jsonData)
		}
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

	if resp.StatusCode != 200 {
		return append(diags, diag.Errorf("HTTP request error. Response code: %d", resp.StatusCode)...)
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

	d.Set("body", string(bytes))
	if err = d.Set("response_headers", responseHeaders); err != nil {
		return append(diags, diag.Errorf("Error setting HTTP response headers: %s", err)...)
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
