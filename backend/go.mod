module go-wind-oa

go 1.26.4

// 本地砖块复用：OA 模块通过 replace 指向同仓 go-wind-cms 底座，复用其
// 鉴权 / viewer / TenantPrivacy / 服务发现 / internal_message 通知等组件，
// 不在 OA 内重写。发布前需改为正式版本依赖或私有代理路径。
// 路径相对本 go.mod（go-wind-oa/backend）：../../go-wind-cms/backend。
replace go-wind-cms => ../../go-wind-cms/backend

require (
	entgo.io/ent v0.14.6
	github.com/go-kratos/kratos/v2 v2.9.2
	github.com/go-sql-driver/mysql v1.10.0
	github.com/google/gnostic v0.7.1
	github.com/google/wire v0.7.0
	github.com/jackc/pgx/v5 v5.10.0
	github.com/lib/pq v1.12.3
	github.com/tx7do/go-crud/api v0.0.7
	github.com/tx7do/go-crud/entgo v0.0.52
	github.com/tx7do/go-crud/viewer v0.0.6
	github.com/tx7do/go-utils v1.1.40
	github.com/tx7do/go-utils/copierutil v0.0.8
	github.com/tx7do/go-utils/mapper v0.0.3
	github.com/tx7do/kratos-bootstrap/api v0.0.43
	github.com/tx7do/kratos-bootstrap/bootstrap v0.1.16
	github.com/tx7do/kratos-bootstrap/database/ent v0.1.5
	github.com/tx7do/kratos-bootstrap/registry v0.2.2
	github.com/tx7do/kratos-bootstrap/registry/etcd v0.2.2
	github.com/tx7do/kratos-bootstrap/rpc v0.1.1
	github.com/tx7do/kratos-bootstrap/tracer v0.1.4
	go-wind-cms v0.0.0-00010101000000-000000000000
	google.golang.org/genproto/googleapis/api v0.0.0-20260810153831-ec0a7760b754
	google.golang.org/grpc v1.83.0
	google.golang.org/protobuf v1.36.12
)

require (
	ariga.io/atlas v1.3.0 // indirect
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260209202127-80ab13bee0bf.1 // indirect
	buf.build/go/protovalidate v1.1.2 // indirect
	cel.dev/expr v0.25.2 // indirect
	dario.cat/mergo v1.0.2 // indirect
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/XSAM/otelsql v0.43.0 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/apparentlymart/go-textseg/v17 v17.0.1 // indirect
	github.com/bmatcuk/doublestar v1.3.4 // indirect
	github.com/bwmarrin/snowflake v0.3.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.7.0 // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/felixge/httpsnoop v1.1.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-kratos/aegis v0.2.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-openapi/inflect v1.0.0 // indirect
	github.com/go-playground/form/v4 v4.3.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/cel-go v0.27.0 // indirect
	github.com/google/gnostic-models v0.7.1 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/subcommands v1.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/handlers v1.5.2 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.30.0 // indirect
	github.com/hashicorp/hcl/v2 v2.24.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/copier v0.4.0 // indirect
	github.com/lithammer/shortuuid/v4 v4.2.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20260216142805-b3301c5f2a88 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/mattn/go-runewidth v0.0.23 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/olekukonko/cat v0.0.0-20250911104152-50322a0618f6 // indirect
	github.com/olekukonko/errors v1.1.0 // indirect
	github.com/olekukonko/ll v0.1.4-0.20260115111900-9e59c2286df0 // indirect
	github.com/olekukonko/tablewriter v1.1.3 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/redis/go-redis/v9 v9.22.0 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/segmentio/ksuid v1.0.4 // indirect
	github.com/shirou/gopsutil/v3 v3.24.5 // indirect
	github.com/shoenig/go-m1cpu v0.1.7 // indirect
	github.com/sony/sonyflake v1.3.0 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/tklauser/go-sysconf v0.3.16 // indirect
	github.com/tklauser/numcpus v0.11.0 // indirect
	github.com/tx7do/go-crud/audit v0.0.2 // indirect
	github.com/tx7do/go-crud/cache v0.0.1 // indirect
	github.com/tx7do/go-crud/pagination v0.0.15 // indirect
	github.com/tx7do/go-utils/id v0.0.6 // indirect
	github.com/tx7do/go-wind v0.0.2 // indirect
	github.com/tx7do/go-wind-plugins/encoding v0.0.1 // indirect
	github.com/tx7do/go-wind-plugins/encoding/json v0.0.1 // indirect
	github.com/tx7do/go-wind-toolkit/protoc-gen-go-redact v0.0.0-20260621140333-c4bf78d8d062 // indirect
	github.com/tx7do/kratos-authn v1.1.11 // indirect
	github.com/tx7do/kratos-authz v1.1.8 // indirect
	github.com/tx7do/kratos-authz/middleware v1.1.13 // indirect
	github.com/tx7do/kratos-bootstrap/config v0.2.2 // indirect
	github.com/tx7do/kratos-bootstrap/logger v0.1.2 // indirect
	github.com/tx7do/kratos-transport/broker v1.3.3 // indirect
	github.com/tx7do/kratos-transport/tracing v1.1.2 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	github.com/zclconf/go-cty v1.19.0 // indirect
	github.com/zclconf/go-cty-yaml v1.2.0 // indirect
	go.einride.tech/aip v0.86.3 // indirect
	go.etcd.io/etcd/api/v3 v3.7.1 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.7.1 // indirect
	go.etcd.io/etcd/client/v3 v3.7.1 // indirect
	go.mongodb.org/mongo-driver/v2 v2.8.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.45.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.45.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.45.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.45.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.45.0 // indirect
	go.opentelemetry.io/otel/exporters/zipkin v1.45.0 // indirect
	go.opentelemetry.io/otel/metric v1.45.0 // indirect
	go.opentelemetry.io/otel/sdk v1.45.0 // indirect
	go.opentelemetry.io/otel/trace v1.45.0 // indirect
	go.opentelemetry.io/proto/otlp v1.11.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/exp v0.0.0-20260410095643-746e56fc9e2f // indirect
	golang.org/x/mod v0.40.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/tools v0.49.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260810153831-ec0a7760b754 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
