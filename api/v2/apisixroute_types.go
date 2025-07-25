// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package v2

import (
	"strings"

	"github.com/pkg/errors"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/apache/apisix-ingress-controller/api/adc"
)

// ApisixRouteSpec is the spec definition for ApisixRouteSpec.
type ApisixRouteSpec struct {
	IngressClassName string              `json:"ingressClassName,omitempty" yaml:"ingressClassName,omitempty"`
	HTTP             []ApisixRouteHTTP   `json:"http,omitempty" yaml:"http,omitempty"`
	Stream           []ApisixRouteStream `json:"stream,omitempty" yaml:"stream,omitempty"`
}

// ApisixRouteStatus defines the observed state of ApisixRoute.
type ApisixRouteStatus = ApisixStatus

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ar
// +kubebuilder:printcolumn:name="Hosts",type="string",JSONPath=".spec.http[].match.hosts",description="HTTP Hosts",priority=0
// +kubebuilder:printcolumn:name="URIs",type="string",JSONPath=".spec.http[].match.paths",description="HTTP Paths",priority=0
// +kubebuilder:printcolumn:name="Target Service (HTTP)",type="string",JSONPath=".spec.http[].backends[].serviceName",description="Backend Service for HTTP",priority=1
// +kubebuilder:printcolumn:name="Ingress Port (TCP)",type="integer",JSONPath=".spec.tcp[].match.ingressPort",description="TCP Ingress Port",priority=1
// +kubebuilder:printcolumn:name="Target Service (TCP)",type="string",JSONPath=".spec.tcp[].match.backend.serviceName",description="Backend Service for TCP",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Creation time",priority=0

// ApisixRoute is the Schema for the apisixroutes API.
type ApisixRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApisixRouteSpec   `json:"spec,omitempty"`
	Status ApisixRouteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApisixRouteList contains a list of ApisixRoute.
type ApisixRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApisixRoute `json:"items"`
}

// ApisixRouteHTTP represents a single route in for HTTP traffic.
type ApisixRouteHTTP struct {
	// The rule name, cannot be empty.
	Name string `json:"name" yaml:"name"`
	// Route priority, when multiple routes contains
	// same URI path (for path matching), route with
	// higher priority will take effect.
	Priority int                  `json:"priority,omitempty" yaml:"priority,omitempty"`
	Timeout  *UpstreamTimeout     `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Match    ApisixRouteHTTPMatch `json:"match,omitempty" yaml:"match,omitempty"`
	// Backends represents potential backends to proxy after the route
	// rule matched. When number of backends are more than one, traffic-split
	// plugin in APISIX will be used to split traffic based on the backend weight.
	Backends []ApisixRouteHTTPBackend `json:"backends,omitempty" yaml:"backends,omitempty"`
	// Upstreams refer to ApisixUpstream CRD
	Upstreams []ApisixRouteUpstreamReference `json:"upstreams,omitempty" yaml:"upstreams,omitempty"`

	// +kubebuilder:validation:Optional
	Websocket        bool   `json:"websocket" yaml:"websocket"`
	PluginConfigName string `json:"plugin_config_name,omitempty" yaml:"plugin_config_name,omitempty"`
	// By default, PluginConfigNamespace will be the same as the namespace of ApisixRoute
	PluginConfigNamespace string                    `json:"plugin_config_namespace,omitempty" yaml:"plugin_config_namespace,omitempty"`
	Plugins               []ApisixRoutePlugin       `json:"plugins,omitempty" yaml:"plugins,omitempty"`
	Authentication        ApisixRouteAuthentication `json:"authentication,omitempty" yaml:"authentication,omitempty"`
}

// ApisixRouteStream is the configuration for level 4 route
type ApisixRouteStream struct {
	// The rule name cannot be empty.
	Name     string                   `json:"name" yaml:"name"`
	Protocol string                   `json:"protocol" yaml:"protocol"`
	Match    ApisixRouteStreamMatch   `json:"match" yaml:"match"`
	Backend  ApisixRouteStreamBackend `json:"backend" yaml:"backend"`
	Plugins  []ApisixRoutePlugin      `json:"plugins,omitempty" yaml:"plugins,omitempty"`
}

// UpstreamTimeout is settings for the read, send and connect to the upstream.
type UpstreamTimeout struct {
	Connect metav1.Duration `json:"connect,omitempty" yaml:"connect,omitempty"`
	Send    metav1.Duration `json:"send,omitempty" yaml:"send,omitempty"`
	Read    metav1.Duration `json:"read,omitempty" yaml:"read,omitempty"`
}

// ApisixRouteHTTPMatch represents the match condition for hitting this route.
type ApisixRouteHTTPMatch struct {
	// URI path predicates, at least one path should be
	// configured, path could be exact or prefix, for prefix path,
	// append "*" after it, for instance, "/foo*".
	Paths []string `json:"paths" yaml:"paths"`
	// HTTP request method predicates.
	Methods []string `json:"methods,omitempty" yaml:"methods,omitempty"`
	// HTTP Host predicates, host can be a wildcard domain or
	// an exact domain. For wildcard domain, only one generic
	// level is allowed, for instance, "*.foo.com" is valid but
	// "*.*.foo.com" is not.
	Hosts []string `json:"hosts,omitempty" yaml:"hosts,omitempty"`
	// Remote address predicates, items can be valid IPv4 address
	// or IPv6 address or CIDR.
	RemoteAddrs []string `json:"remoteAddrs,omitempty" yaml:"remoteAddrs,omitempty"`
	// NginxVars represents generic match predicates,
	// it uses Nginx variable systems, so any predicate
	// like headers, querystring and etc can be leveraged
	// here to match the route.
	// For instance, it can be:
	// nginxVars:
	//   - subject: "$remote_addr"
	//     op: in
	//     value:
	//       - "127.0.0.1"
	//       - "10.0.5.11"
	NginxVars ApisixRouteHTTPMatchExprs `json:"exprs,omitempty" yaml:"exprs,omitempty"`
	// Matches based on a user-defined filtering function.
	// These functions can accept an input parameter `vars`
	// which can be used to access the Nginx variables.
	FilterFunc string `json:"filter_func,omitempty" yaml:"filter_func,omitempty"`
}

// ApisixRoutePlugin represents an APISIX plugin.
type ApisixRoutePlugin struct {
	// The plugin name.
	Name string `json:"name" yaml:"name"`
	// Whether this plugin is in use, default is true.
	// +kubebuilder:default=true
	Enable bool `json:"enable" yaml:"enable"`
	// Plugin configuration.
	// +kubebuilder:validation:Optional
	Config apiextensionsv1.JSON `json:"config" yaml:"config"`
	// Plugin configuration secretRef.
	// +kubebuilder:validation:Optional
	SecretRef string `json:"secretRef" yaml:"secretRef"`
}

// ApisixRouteHTTPBackend represents an HTTP backend (a Kubernetes Service).
type ApisixRouteHTTPBackend struct {
	// The name (short) of the service, note cross namespace is forbidden,
	// so be sure the ApisixRoute and Service are in the same namespace.
	ServiceName string `json:"serviceName" yaml:"serviceName"`
	// The service port, could be the name or the port number.
	ServicePort intstr.IntOrString `json:"servicePort" yaml:"servicePort"`
	// The resolve granularity, can be "endpoints" or "service",
	// when set to "endpoints", the pod ips will be used; other
	// wise, the service ClusterIP or ExternalIP will be used,
	// default is endpoints.
	ResolveGranularity string `json:"resolveGranularity,omitempty" yaml:"resolveGranularity,omitempty"`
	// Weight of this backend.
	// +kubebuilder:validation:Optional
	Weight *int `json:"weight" yaml:"weight"`
	// Subset specifies a subset for the target Service. The subset should be pre-defined
	// in ApisixUpstream about this service.
	Subset string `json:"subset,omitempty" yaml:"subset,omitempty"`
}

// ApisixRouteUpstreamReference contains a ApisixUpstream CRD reference
type ApisixRouteUpstreamReference struct {
	Name string `json:"name,omitempty" yaml:"name"`
	// +kubebuilder:validation:Optional
	Weight *int `json:"weight,omitempty" yaml:"weight"`
}

// ApisixRouteAuthentication is the authentication-related
// configuration in ApisixRoute.
type ApisixRouteAuthentication struct {
	Enable   bool                              `json:"enable" yaml:"enable"`
	Type     string                            `json:"type" yaml:"type"`
	KeyAuth  ApisixRouteAuthenticationKeyAuth  `json:"keyAuth,omitempty" yaml:"keyAuth,omitempty"`
	JwtAuth  ApisixRouteAuthenticationJwtAuth  `json:"jwtAuth,omitempty" yaml:"jwtAuth,omitempty"`
	LDAPAuth ApisixRouteAuthenticationLDAPAuth `json:"ldapAuth,omitempty" yaml:"ldapAuth,omitempty"`
}

// ApisixRouteStreamMatch represents the match conditions of stream route.
type ApisixRouteStreamMatch struct {
	// IngressPort represents the port listening on the Ingress proxy server.
	// It should be pre-defined as APISIX doesn't support dynamic listening.
	IngressPort int32  `json:"ingressPort" yaml:"ingressPort"`
	Host        string `json:"host,omitempty" yaml:"host,omitempty"`
}

// ApisixRouteStreamBackend represents a TCP backend (a Kubernetes Service).
type ApisixRouteStreamBackend struct {
	// The name (short) of the service, note cross namespace is forbidden,
	// so be sure the ApisixRoute and Service are in the same namespace.
	ServiceName string `json:"serviceName" yaml:"serviceName"`
	// The service port, could be the name or the port number.
	ServicePort intstr.IntOrString `json:"servicePort" yaml:"servicePort"`
	// The resolve granularity, can be "endpoints" or "service",
	// when set to "endpoints", the pod ips will be used; other
	// wise, the service ClusterIP or ExternalIP will be used,
	// default is endpoints.
	ResolveGranularity string `json:"resolveGranularity,omitempty" yaml:"resolveGranularity,omitempty"`
	// Subset specifies a subset for the target Service. The subset should be pre-defined
	// in ApisixUpstream about this service.
	Subset string `json:"subset,omitempty" yaml:"subset,omitempty"`
}

// ApisixRouteHTTPMatchExpr represents a binary route match expression .
type ApisixRouteHTTPMatchExpr struct {
	// Subject is the expression subject, it can
	// be any string composed by literals and nginx
	// vars.
	Subject ApisixRouteHTTPMatchExprSubject `json:"subject" yaml:"subject"`
	// Op is the operator.
	Op string `json:"op" yaml:"op"`
	// Set is an array type object of the expression.
	// It should be used when the Op is "in" or "not_in";
	// +kubebuilder:validation:Optional
	Set []string `json:"set" yaml:"set"`
	// Value is the normal type object for the expression,
	// it should be used when the Op is not "in" and "not_in".
	// Set and Value are exclusive so only of them can be set
	// in the same time.
	// +kubebuilder:validation:Optional
	Value *string `json:"value" yaml:"value"`
}

type ApisixRouteHTTPMatchExprs []ApisixRouteHTTPMatchExpr

func (exprs ApisixRouteHTTPMatchExprs) ToVars() (result adc.Vars, err error) {
	for _, expr := range exprs {
		if expr.Subject.Name == "" && expr.Subject.Scope != ScopePath {
			return result, errors.New("empty subject.name")
		}

		// process key
		var (
			subj string
			this adc.StringOrSlice
		)
		switch expr.Subject.Scope {
		case ScopeQuery:
			subj = "arg_" + expr.Subject.Name
		case ScopeHeader:
			subj = "http_" + strings.ReplaceAll(strings.ToLower(expr.Subject.Name), "-", "_")
		case ScopeCookie:
			subj = "cookie_" + expr.Subject.Name
		case ScopePath:
			subj = "uri"
		case ScopeVariable:
			subj = expr.Subject.Name
		default:
			return result, errors.New("invalid http match expr: subject.scope should be one of [query, header, cookie, path, variable]")
		}
		this.SliceVal = append(this.SliceVal, adc.StringOrSlice{StrVal: subj})

		// process operator
		var (
			op string
		)
		switch expr.Op {
		case OpEqual:
			op = "=="
		case OpGreaterThan:
			op = ">"
		case OpGreaterThanEqual:
			op = ">="
		case OpIn:
			op = "in"
		case OpLessThan:
			op = "<"
		case OpLessThanEqual:
			op = "<="
		case OpNotEqual:
			op = "~="
		case OpNotIn:
			op = "in"
		case OpRegexMatch:
			op = "~~"
		case OpRegexMatchCaseInsensitive:
			op = "~*"
		case OpRegexNotMatch:
			op = "~~"
		case OpRegexNotMatchCaseInsensitive:
			op = "~*"
		default:
			return result, errors.New("unknown operator")
		}
		if expr.Op == OpNotIn || expr.Op == OpRegexNotMatch || expr.Op == OpRegexNotMatchCaseInsensitive {
			this.SliceVal = append(this.SliceVal, adc.StringOrSlice{StrVal: "!"})
		}
		this.SliceVal = append(this.SliceVal, adc.StringOrSlice{StrVal: op})

		// process value
		switch expr.Op {
		case OpIn, OpNotIn:
			if expr.Set == nil {
				return result, errors.New("empty set value")
			}
			var value adc.StringOrSlice
			for _, item := range expr.Set {
				value.SliceVal = append(value.SliceVal, adc.StringOrSlice{StrVal: item})
			}
			this.SliceVal = append(this.SliceVal, value)
		default:
			if expr.Value == nil {
				return result, errors.New("empty value")
			}
			this.SliceVal = append(this.SliceVal, adc.StringOrSlice{StrVal: *expr.Value})
		}

		// append to result
		result = append(result, this.SliceVal)
	}

	return result, nil
}

// ApisixRoutePluginConfig is the configuration for
// any plugins.
type ApisixRoutePluginConfig map[string]apiextensionsv1.JSON

// ApisixRouteAuthenticationKeyAuth is the keyAuth-related
// configuration in ApisixRouteAuthentication.
type ApisixRouteAuthenticationKeyAuth struct {
	Header string `json:"header,omitempty" yaml:"header,omitempty"`
}

// ApisixRouteAuthenticationJwtAuth is the jwt auth related
// configuration in ApisixRouteAuthentication.
type ApisixRouteAuthenticationJwtAuth struct {
	Header string `json:"header,omitempty" yaml:"header,omitempty"`
	Query  string `json:"query,omitempty" yaml:"query,omitempty"`
	Cookie string `json:"cookie,omitempty" yaml:"cookie,omitempty"`
}

// ApisixRouteAuthenticationLDAPAuth is the LDAP auth related
// configuration in ApisixRouteAuthentication.
type ApisixRouteAuthenticationLDAPAuth struct {
	BaseDN  string `json:"base_dn,omitempty" yaml:"base_dn,omitempty"`
	LDAPURI string `json:"ldap_uri,omitempty" yaml:"ldap_uri,omitempty"`
	UseTLS  bool   `json:"use_tls,omitempty" yaml:"use_tls,omitempty"`
	UID     string `json:"uid,omitempty" yaml:"uid,omitempty"`
}

// ApisixRouteHTTPMatchExprSubject describes the route match expression subject.
type ApisixRouteHTTPMatchExprSubject struct {
	// The subject scope, can be:
	// ScopeQuery, ScopeHeader, ScopePath
	// when subject is ScopePath, Name field
	// will be ignored.
	Scope string `json:"scope" yaml:"scope"`
	// The name of subject.
	Name string `json:"name" yaml:"name"`
}

func init() {
	SchemeBuilder.Register(&ApisixRoute{}, &ApisixRouteList{})
}
