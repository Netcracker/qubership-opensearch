// Copyright 2024-2025 NetCracker Technology Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"context"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func newCreateRoleFunc(t opensearchapi.Transport) CreateRole {
	return func(role string, o ...func(request *CreateRoleRequest)) (*opensearchapi.Response, error) {
		var r = CreateRoleRequest{Role: role}
		for _, f := range o {
			f(&r)
		}
		return r.Do(r.ctx, t)
	}
}

// ----- API Definition -------------------------------------------------------

// CreateRole creates a role
type CreateRole func(role string, o ...func(request *CreateRoleRequest)) (*opensearchapi.Response, error)

// CreateRoleRequest configures the Role API request.
type CreateRoleRequest struct {
	Role string

	Body io.Reader

	WaitForCompletion *bool

	Pretty     bool
	Human      bool
	ErrorTrace bool
	FilterPath []string

	Header http.Header

	ctx context.Context
}

// Do function executes the request and returns response or error.
func (r CreateRoleRequest) Do(ctx context.Context, transport opensearchapi.Transport) (*opensearchapi.Response, error) {
	var (
		method string
		path   strings.Builder
		params map[string]string
	)

	method = http.MethodPut
	path.Grow(1 + len("_plugins/_security/api/roles") + 1 + len(r.Role))
	path.WriteString("/_plugins/_security/api/roles")
	path.WriteString("/")
	path.WriteString(r.Role)

	params = make(map[string]string)

	if r.WaitForCompletion != nil {
		params["wait_for_completion"] = strconv.FormatBool(*r.WaitForCompletion)
	}

	if r.Pretty {
		params["pretty"] = "true"
	}

	if r.Human {
		params["human"] = "true"
	}

	if r.ErrorTrace {
		params["error_trace"] = "true"
	}

	if len(r.FilterPath) > 0 {
		params["filter_path"] = strings.Join(r.FilterPath, ",")
	}

	req, err := http.NewRequest(method, path.String(), r.Body)
	if err != nil {
		return nil, err
	}
	defer req.Body.Close()

	if len(params) > 0 {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	if len(r.Header) > 0 {
		if len(req.Header) == 0 {
			req.Header = r.Header
		} else {
			for k, vv := range r.Header {
				for _, v := range vv {
					req.Header.Add(k, v)
				}
			}
		}
	}

	if ctx != nil {
		req = req.WithContext(ctx)
	}
	//nolint:bodyclose
	res, err := transport.Perform(req)
	if err != nil {
		return nil, err
	}

	response := opensearchapi.Response{
		StatusCode: res.StatusCode,
		Body:       res.Body,
		Header:     res.Header,
	}

	return &response, nil
}

// WithRole sets the request role name.
func (f CreateRole) WithRole(v string) func(*CreateRoleRequest) {
	return func(r *CreateRoleRequest) {
		r.Role = v
	}
}

// WithBody sets the request body.
func (f CreateRole) WithBody(v io.Reader) func(*CreateRoleRequest) {
	return func(r *CreateRoleRequest) {
		r.Body = v
	}
}

// WithContext sets the request context.
func (f CreateRole) WithContext(v context.Context) func(*CreateRoleRequest) {
	return func(r *CreateRoleRequest) {
		r.ctx = v
	}
}

// WithPretty makes the response body pretty-printed.
func (f CreateRole) WithPretty() func(*CreateRoleRequest) {
	return func(r *CreateRoleRequest) {
		r.Pretty = true
	}
}

// WithHuman makes statistical values human-readable.
func (f CreateRole) WithHuman() func(*CreateRoleRequest) {
	return func(r *CreateRoleRequest) {
		r.Human = true
	}
}

// WithErrorTrace includes the stack trace for errors in the response body.
func (f CreateRole) WithErrorTrace() func(*CreateRoleRequest) {
	return func(r *CreateRoleRequest) {
		r.ErrorTrace = true
	}
}

// WithFilterPath filters the properties of the response body.
func (f CreateRole) WithFilterPath(v ...string) func(*CreateRoleRequest) {
	return func(r *CreateRoleRequest) {
		r.FilterPath = v
	}
}

// WithHeader adds the headers to the HTTP request.
func (f CreateRole) WithHeader(h map[string]string) func(*CreateRoleRequest) {
	return func(r *CreateRoleRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		for k, v := range h {
			r.Header.Add(k, v)
		}
	}
}

// WithOpaqueID adds the X-Opaque-Id header to the HTTP request.
func (f CreateRole) WithOpaqueID(s string) func(*CreateRoleRequest) {
	return func(r *CreateRoleRequest) {
		if r.Header == nil {
			r.Header = make(http.Header)
		}
		r.Header.Set("X-Opaque-Id", s)
	}
}
