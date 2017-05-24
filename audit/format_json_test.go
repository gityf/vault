package audit

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"errors"

	"github.com/hashicorp/vault/helper/jsonutil"
	"github.com/hashicorp/vault/helper/salt"
	"github.com/hashicorp/vault/logical"
)

func TestFormatJSON_formatRequest(t *testing.T) {
	salter, err := salt.NewSalt(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	saltFunc := func() (*salt.Salt, error) {
		return salter, nil
	}
	cases := map[string]struct {
		Auth   *logical.Auth
		Req    *logical.Request
		Err    error
		Prefix string
		Result string
	}{
		"auth, request": {
			&logical.Auth{ClientToken: "foo", Policies: []string{"root"}},
			&logical.Request{
				Operation: logical.UpdateOperation,
				Path:      "/foo",
				Connection: &logical.Connection{
					RemoteAddr: "127.0.0.1",
				},
				WrapInfo: &logical.RequestWrapInfo{
					TTL: 60 * time.Second,
				},
				Headers: map[string][]string{
					"foo": []string{"bar"},
				},
			},
			errors.New("this is an error"),
			"",
			testFormatJSONReqBasicStr,
		},
		"auth, request with prefix": {
			&logical.Auth{ClientToken: "foo", Policies: []string{"root"}},
			&logical.Request{
				Operation: logical.UpdateOperation,
				Path:      "/foo",
				Connection: &logical.Connection{
					RemoteAddr: "127.0.0.1",
				},
				WrapInfo: &logical.RequestWrapInfo{
					TTL: 60 * time.Second,
				},
				Headers: map[string][]string{
					"foo": []string{"bar"},
				},
			},
			errors.New("this is an error"),
			"@cee: ",
			testFormatJSONReqBasicStr,
		},
	}

	for name, tc := range cases {
		var buf bytes.Buffer
		formatter := AuditFormatter{
			AuditFormatWriter: &JSONFormatWriter{
				Prefix:   tc.Prefix,
				SaltFunc: saltFunc,
			},
		}
		config := FormatterConfig{}
		if err := formatter.FormatRequest(&buf, config, tc.Auth, tc.Req, tc.Err); err != nil {
			t.Fatalf("bad: %s\nerr: %s", name, err)
		}

		if !strings.HasPrefix(buf.String(), tc.Prefix) {
			t.Fatalf("no prefix: %s \n log: %s\nprefix: %s", name, tc.Result, tc.Prefix)
		}

		var expectedjson = new(AuditRequestEntry)
		if err := jsonutil.DecodeJSON([]byte(tc.Result), &expectedjson); err != nil {
			t.Fatalf("bad json: %s", err)
		}

		var actualjson = new(AuditRequestEntry)
		if err := jsonutil.DecodeJSON([]byte(buf.String())[len(tc.Prefix):], &actualjson); err != nil {
			t.Fatalf("bad json: %s", err)
		}

		expectedjson.Time = actualjson.Time

		expectedBytes, err := json.Marshal(expectedjson)
		if err != nil {
			t.Fatalf("unable to marshal json: %s", err)
		}

		if !strings.HasSuffix(strings.TrimSpace(buf.String()), string(expectedBytes)) {
			t.Fatalf(
				"bad: %s\nResult:\n\n'%s'\n\nExpected:\n\n'%s'",
				name, buf.String(), string(expectedBytes))
		}
	}
}

const testFormatJSONReqBasicStr = `{"time":"2015-08-05T13:45:46Z","type":"request","auth":{"display_name":"","policies":["root"],"metadata":null},"request":{"operation":"update","path":"/foo","data":null,"wrap_ttl":60,"remote_address":"127.0.0.1","headers":{"foo":["bar"]}},"error":"this is an error"}
`
