/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2026 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package calling

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/WebexCommunity/webex-go-sdk/v2/webexsdk"
)

func TestFetchMobiusHostsFromU2C(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/catalog" {
			t.Errorf("path = %q, want /catalog", r.URL.Path)
		}
		if r.URL.Query().Get("format") != "U2CV2" {
			t.Errorf("format = %q, want U2CV2", r.URL.Query().Get("format"))
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("authorization header = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"services":[
				{"serviceName":"mobius","serviceUrls":[
					{"baseUrl":"https://mobius-us-east-2.prod.infra.webex.com/api/v1"},
					{"baseUrl":"https://mobius-eu-central-1.prod.infra.webex.com/api/v1"}
				]}
			]
		}`))
	}))
	defer server.Close()

	core, err := webexsdk.NewClient("test-token", nil)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	client := NewCallingClient(core, DefaultConfig(), nil)
	got, err := client.fetchMobiusHostsFromU2C(server.URL, map[string]bool{})
	if err != nil {
		t.Fatalf("fetchMobiusHostsFromU2C() error = %v", err)
	}
	want := []string{
		"mobius-us-east-2.prod.infra.webex.com",
		"mobius-eu-central-1.prod.infra.webex.com",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fetchMobiusHostsFromU2C() = %v, want %v", got, want)
	}
}

func TestExtractMobiusHosts(t *testing.T) {
	tests := []struct {
		name     string
		services string
		seen     map[string]bool
		want     []string
	}{
		{
			name: "services array",
			services: `[
				{"serviceName":"mobius","serviceUrls":[
					{"baseUrl":"https://mobius-us-east-2.prod.infra.webex.com/api/v1"},
					{"baseUrl":"https://mobius-eu-central-1.prod.infra.webex.com/api/v1"}
				]},
				{"serviceName":"wdm","serviceUrls":[{"baseUrl":"https://wdm-a.wbx2.com/wdm/api/v1"}]}
			]`,
			seen: map[string]bool{},
			want: []string{
				"mobius-us-east-2.prod.infra.webex.com",
				"mobius-eu-central-1.prod.infra.webex.com",
			},
		},
		{
			name: "keyed services object",
			services: `{
				"mobius":{"serviceUrls":[
					{"baseUrl":"https://mobius-us-east-1.prod.infra.webex.com/api/v1"},
					{"baseUrl":"not-a-url"}
				]},
				"wdm":{"serviceUrls":[{"baseUrl":"https://wdm-a.wbx2.com/wdm/api/v1"}]}
			}`,
			seen: map[string]bool{},
			want: []string{"mobius-us-east-1.prod.infra.webex.com"},
		},
		{
			name: "deduplicates existing host",
			services: `[
				{"serviceName":"MOBIUS","serviceUrls":[
					{"baseUrl":"https://mobius-us-east-2.prod.infra.webex.com/api/v1"},
					{"baseUrl":"https://mobius-ca-central-1.prod.infra.webex.com/api/v1"}
				]}
			]`,
			seen: map[string]bool{"mobius-us-east-2.prod.infra.webex.com": true},
			want: []string{"mobius-ca-central-1.prod.infra.webex.com"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := extractMobiusHosts(json.RawMessage(test.services), test.seen)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("extractMobiusHosts() = %v, want %v", got, test.want)
			}
		})
	}
}
