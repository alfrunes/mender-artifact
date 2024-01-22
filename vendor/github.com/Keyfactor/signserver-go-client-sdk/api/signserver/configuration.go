/*
Copyright 2022 Keyfactor
Licensed under the Apache License, Version 2.0 (the "License"); you may
not use this file except in compliance with the License.  You may obtain a
copy of the License at http://www.apache.org/licenses/LICENSE-2.0.  Unless
required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES
OR CONDITIONS OF ANY KIND, either express or implied. See the License for
thespecific language governing permissions and limitations under the
License.

SignServer REST Interface

No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)

API version: 1.0
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package signserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
)

// contextKeys are used to identify the type of value in the context.
// Since these are string, it is possible to get a short description of the
// context key for logging and debugging using key.String().

type contextKey string

func (c contextKey) String() string {
	return "auth " + string(c)
}

var ()

// BasicAuth provides basic http authentication to a request passed via context using ContextBasicAuth
type BasicAuth struct {
	UserName string `json:"userName,omitempty"`
	Password string `json:"password,omitempty"`
}

// APIKey provides API key based authentication to a request passed via context using ContextAPIKey
type APIKey struct {
	Key    string
	Prefix string
}

// Configuration stores the configuration of the API client
type Configuration struct {
	Host                     string            `json:"host,omitempty"`
	DefaultHeader            map[string]string `json:"defaultHeader,omitempty"`
	UserAgent                string            `json:"userAgent,omitempty"`
	Debug                    bool              `json:"debug,omitempty"`
	ClientCertificatePath    string            `json:"clientCertificatePath,omitempty"`
	ClientCertificateKeyPath string            `json:"clientCertificateKeyPath,omitempty"`
	CaCertificatePath        string            `json:"caCertificatePath,omitempty"`
	HTTPClient               *http.Client
	clientTlsCertificate     *tls.Certificate
	caCertificates           []*x509.Certificate
}

// NewConfiguration returns a new Configuration object
func NewConfiguration() *Configuration {
	cfg := &Configuration{
		DefaultHeader: make(map[string]string),
		UserAgent:     "OpenAPI-Generator/1.0.0/go",
		Debug:         false,
	}

	// Get hostname from environment variable
	hostname := os.Getenv("SIGNSERVER_HOSTNAME")
	if hostname != "" {
		if hostname, err := cleanHostname(hostname); err == nil {
			cfg.Host = hostname
		} else {
			fmt.Errorf("SIGNSERVER_HOSTNAME is not a valid URL: %s", err)
		}
	}

	// Get client certificate path from environment variable
	clientCertPath := os.Getenv("SIGNSERVER_CLIENT_CERT_PATH")
	if clientCertPath != "" {
		cfg.ClientCertificatePath = clientCertPath
	}

	// Get client certificate key path from environment variable
	clientCertKeyPath := os.Getenv("SIGNSERVER_CLIENT_CERT_KEY_PATH")
	if clientCertKeyPath != "" {
		cfg.ClientCertificateKeyPath = clientCertKeyPath
	}

	// Get CA certificate path from environment variable
	caCertPath := os.Getenv("SIGNSERVER_CA_CERT_PATH")
	if caCertPath != "" {
		cfg.CaCertificatePath = caCertPath
	}

	return cfg
}

// AddDefaultHeader adds a new HTTP header to the default header in the request
func (c *Configuration) AddDefaultHeader(key string, value string) {
	c.DefaultHeader[key] = value
}

func (c *Configuration) SetClientCertificate(clientCertificate *tls.Certificate) {
	if clientCertificate != nil {
		c.clientTlsCertificate = clientCertificate
	}
}

func (c *Configuration) SetCaCertificates(caCertificates []*x509.Certificate) {
	if caCertificates != nil {
		c.caCertificates = caCertificates
	}
}
