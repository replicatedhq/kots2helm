package builder

import (
	"testing"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/kotskinds/multitype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_numKotsTemplateFunctions(t *testing.T) {
	tests := []struct {
		name    string
		content string
		expect  int
	}{
		{
			name:    "no repl or repl{{",
			content: "nothing here",
			expect:  0,
		},
		{
			name: "a standard deployment",
			content: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  selector:
    matchLabels:
      app: api
  template:
    metadata:
      labels:
        app: api
    spec:
      restartPolicy: Always
      containers:
        - name: api
          image: api
          imagePullPolicy: IfNotPresent
          resources:
            limits:
              cpu: 200m
              memory: 1000Mi
            requests:
              cpu: 100m
              memory: 500Mi
          command:
            - "/api"
          ports:
            - name: http
              containerPort: 3000
          env:
            - name: POD_NAMESPACE
              value: repl{{ Namespace }}
			- name: POD_NAMESPACE2
              value: {{repl Namespace}}
            - name: TIMESCALE_URI
              valueFrom:
                secretKeyRef:
                  name: timescale # This secret is created in the migrations directory
                  key: uri
            - name: GOOGLE_AUTH_CLIENT_ID
              valueFrom:
                secretKeyRef:
                  name: google-auth
                  key: clientId
            - name: GOOGLE_AUTH_CLIENT_SECRET
              valueFrom:
                secretKeyRef:
                  name: google-auth
                  key: clientSecret
            - name: GOOGLE_REDIRECT_URI
              valueFrom:
                secretKeyRef:
                  name: google-auth
                  key: redirectUri
`,
			expect: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			actual, err := numKotsTemplateFunctions([]byte(tt.content))

			req.NoError(err)
			assert.Equal(t, tt.expect, actual)
		})
	}
}

func Test_replaceConfigOption(t *testing.T) {
	type args struct {
		content    string
		kotsConfig *kotsv1beta1.Config
	}
	tests := []struct {
		name   string
		args   args
		expect string
	}{
		{
			name: "no configoption templates",
			args: args{
				content: `apiVersion: apps/v1`,
				kotsConfig: &kotsv1beta1.Config{
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{},
					},
				},
			},
			expect: `apiVersion: apps/v1`,
		},
		{
			name: "direct replace",
			args: args{
				content: `name: "{{repl ConfigOption "foo1"}}"`,
				kotsConfig: &kotsv1beta1.Config{
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Name: "group1",
								Items: []kotsv1beta1.ConfigItem{
									{
										Name:  "foo1",
										Value: multitype.FromString("bar"),
									},
								},
							},
						},
					},
				},
			},
			expect: `name: "{{ .Values.group1.foo1 }}"`,
		},
		{
			name: "direct replace with ` as quotes",
			args: args{
				content: "name: repl{{ ConfigOption `foo`}}",
				kotsConfig: &kotsv1beta1.Config{
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Name: "group1",
								Items: []kotsv1beta1.ConfigItem{
									{
										Name:  "foo",
										Value: multitype.FromString("bar"),
									},
								},
							},
						},
					},
				},
			},
			expect: `name: {{ .Values.group1.foo }}`,
		},
		{
			name: "direct replace with reverse template",
			args: args{
				content: `name: "repl{{ ConfigOption "foo"}}"`,
				kotsConfig: &kotsv1beta1.Config{
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Name: "group1",
								Items: []kotsv1beta1.ConfigItem{
									{
										Name:  "foo",
										Value: multitype.FromString("bar"),
									},
								},
							},
						},
					},
				},
			},
			expect: `name: "{{ .Values.group1.foo }}"`,
		},
		{
			name: "ignoring ConfigOptionEquals",
			args: args{
				content: `name: "repl{{ ConfigOptionEquals "foo" "bar"}}"`,
				kotsConfig: &kotsv1beta1.Config{
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Name: "group1",
								Items: []kotsv1beta1.ConfigItem{
									{
										Name:  "foo",
										Value: multitype.FromString("bar"),
									},
								},
							},
						},
					},
				},
			},
			expect: `name: "repl{{ ConfigOptionEquals "foo" "bar"}}"`,
		},
		{
			name: "config item not found",
			args: args{
				content: `name: "{{repl ConfigOption "foo"}}"`,
				kotsConfig: &kotsv1beta1.Config{
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Name: "group1",
								Items: []kotsv1beta1.ConfigItem{
									{
										Name:  "bar",
										Value: multitype.FromString("baz"),
									},
								},
							},
						},
					},
				},
			},
			expect: `name: "{{repl ConfigOption "foo"}}"`,
		},
		{
			name: "config item piped to base64",
			args: args{
				content: `password: '{{repl ConfigOption "postgres_password" | Base64Encode }}'`,
				kotsConfig: &kotsv1beta1.Config{
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Name: "group1",
								Items: []kotsv1beta1.ConfigItem{
									{
										Name: "postgres_password",
										Type: "password",
									},
								},
							},
						},
					},
				},
			},
			expect: `password: '{{ .Values.group1.postgres_password | Base64Encode }}'`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			actual, err := replaceConfigOption([]byte(tt.args.content), tt.args.kotsConfig)
			req.NoError(err)
			assert.Equal(t, tt.expect, string(actual))
		})
	}
}

func Test_replaceConfigOptionEquals(t *testing.T) {
	type args struct {
		content    string
		kotsConfig *kotsv1beta1.Config
	}
	tests := []struct {
		name   string
		args   args
		expect string
	}{
		{
			name: "no configoptionequals templates",
			args: args{
				content: `apiVersion: apps/v1`,
				kotsConfig: &kotsv1beta1.Config{
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{},
					},
				},
			},
			expect: `apiVersion: apps/v1`,
		},
		{
			name: "direct replace, string type",
			args: args{
				content: `name: "{{repl ConfigOptionEquals "foo" "bar"}}"`,
				kotsConfig: &kotsv1beta1.Config{
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Name: "group1",
								Items: []kotsv1beta1.ConfigItem{
									{
										Name:  "foo",
										Value: multitype.FromString("bar"),
										Type:  "string",
									},
								},
							},
						},
					},
				},
			},
			expect: `name: "{{ if eq .Values.group1.foo "bar" }}true{{ else }}false{{ end }}"`,
		},
		{
			name: "direct replace, bool type 1",
			args: args{
				content: `name: "{{repl ConfigOptionEquals "foo" "1"}}"`,
				kotsConfig: &kotsv1beta1.Config{
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Name: "group1",
								Items: []kotsv1beta1.ConfigItem{
									{
										Name:    "foo",
										Default: multitype.FromBool(false),
										Type:    "bool",
									},
								},
							},
						},
					},
				},
			},
			expect: `name: "{{ if eq .Values.group1.foo true }}true{{ else }}false{{ end }}"`,
		},
		{
			name: "direct replace, bool type 0",
			args: args{
				content: `name: "{{repl ConfigOptionEquals "foo" "0"}}"`,
				kotsConfig: &kotsv1beta1.Config{
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Name: "group1",
								Items: []kotsv1beta1.ConfigItem{
									{
										Name:    "foo",
										Default: multitype.FromBool(false),
										Type:    "bool",
									},
								},
							},
						},
					},
				},
			},
			expect: `name: "{{ if eq .Values.group1.foo false }}true{{ else }}false{{ end }}"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			actual, err := replaceConfigOptionEquals([]byte(tt.args.content), tt.args.kotsConfig)
			req.NoError(err)
			assert.Equal(t, tt.expect, string(actual))
		})
	}
}

func Test_replaceNamespace(t *testing.T) {
	type args struct {
		content string
	}
	tests := []struct {
		name   string
		args   args
		expect string
	}{
		{
			name: "no namespace",
			args: args{
				content: `namespace: "test"`,
			},
			expect: `namespace: "test"`,
		},
		{
			name: "namespace",
			args: args{
				content: `namespace: "{{repl Namespace}}"`,
			},
			expect: `namespace: "{{ .Release.Namespace }}"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			actual, err := replaceNamespace([]byte(tt.args.content))
			req.NoError(err)
			assert.Equal(t, tt.expect, string(actual))
		})
	}
}

func Test_replaceIsKurl(t *testing.T) {
	type args struct {
		content string
	}
	tests := []struct {
		name   string
		args   args
		expect string
	}{
		{
			name: "no isKurl",
			args: args{
				content: `namespace: "test"`,
			},
			expect: `namespace: "test"`,
		},
		{
			name: "simple isKurl",
			args: args{
				content: `isKurl: "{{repl IsKurl}}"`,
			},
			expect: `isKurl: "{{ .Values.isKurl }}"`,
		},
		{
			name: "simple not isKurl",
			args: args{
				content: `"{{repl not IsKurl}}"`,
			},
			expect: `"{{ not .Values.isKurl }}"`,
		},
		// {
		// 	name: "isKurl",
		// 	args: args{
		// 		content: `storageClassName: repl{{ if eq IsKurl false}} repl{{ ConfigOption "storage.shared_storage_class" }} repl{{ else}} longhorn repl{{ end}}`,
		// 	},
		// 	expect: `storageClassName: {{ if eq .Values.isKurl false}} {{ .Values.storage.storage.shared_storage_class }}}} {{ else}} longhorn {{ end}}`,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			actual, err := replaceIsKurl([]byte(tt.args.content))
			req.NoError(err)
			assert.Equal(t, tt.expect, string(actual))
		})
	}
}

func Test_replaceIfAndConditional(t *testing.T) {
	type args struct {
		content string
	}
	tests := []struct {
		name   string
		args   args
		expect string
	}{
		{
			name: "{{repl if",
			args: args{
				content: `{{repl if eq`,
			},
			expect: `{{ if eq`,
		},
		{
			name: "repl{{ if",
			args: args{
				content: `repl{{ if eq`,
			},
			expect: `{{ if eq`,
		},
		{
			name: "repl{{ else}}",
			args: args{
				content: `repl{{ else}}`,
			},
			expect: `{{ else }}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			actual, err := replaceIfAndConditional([]byte(tt.args.content))
			req.NoError(err)
			assert.Equal(t, tt.expect, string(actual))
		})
	}
}

func Test_replaceWhenAndExcludeAnnotations(t *testing.T) {
	type args struct {
		content    string
		kotsConfig *kotsv1beta1.Config
	}
	tests := []struct {
		name   string
		args   args
		expect string
	}{
		{
			name: "when",
			args: args{
				content: `apiVersion: v1
kind: Service
metadata:
  annotations:
    kots.io/when: "{{repl IsKurl}}"
spec:
  type: ClusterIP`,
				kotsConfig: &kotsv1beta1.Config{
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{},
					},
				},
			},
			expect: `{{ if {{ .Values.isKurl }} }}
apiVersion: v1
kind: Service
metadata:
  annotations: {}
spec:
  type: ClusterIP
{{ end }}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			actual, err := replaceWhenAndExcludeAnnotations([]byte(tt.args.content), tt.args.kotsConfig)
			req.NoError(err)
			assert.Equal(t, tt.expect, string(actual))
		})
	}
}
