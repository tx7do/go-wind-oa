package ent

import (
	"context"
	"reflect"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"

	"github.com/tx7do/go-crud/viewer"
	"go.opentelemetry.io/otel/trace"

	authenticationV1 "go-wind-oa/api/gen/go/authentication/service/v1"

	appViewer "go-wind-oa/pkg/entgo/viewer"
	"go-wind-oa/pkg/metadata"
)

// Server 设置 Ent Viewer 到 Context 中的中间件。
//
// 已认证请求（携带 OperatorMetadata）构建 UserViewer。
// 匿名请求（无 OperatorMetadata）：注入 SystemViewer 兜底。
func Server() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			md, err := metadata.FromServerContext(ctx)
			if err != nil {
				reqType := "<nil>"
				if req != nil {
					reqType = reflect.TypeOf(req).String()
				}
				t, _ := transport.FromServerContext(ctx)
				kind, operation, endpoint := "", "", ""
				if t != nil {
					kind = string(t.Kind())
					operation = t.Operation()
					endpoint = t.Endpoint()
				}
				log.Errorf("ent middleware: failed to get metadata from context: %v; req_type=%s; transport=%s; operation=%s; endpoint=%s",
					err, reqType, kind, operation, endpoint)
			}

			var traceID string
			spanContext := trace.SpanContextFromContext(ctx)
			if spanContext.HasTraceID() {
				traceID = spanContext.TraceID().String()
			}

			if md == nil {
				ctx = viewer.WithContext(ctx, appViewer.NewSystemViewer())
			} else {
				ctx = viewer.WithContext(ctx, metaDataToUserViewerContext(md, traceID))
			}

			// TEMP DIAGNOSTIC (oa-viewer-propagation): 记录 /oa/ 请求最终注入的
			// viewer 身份，用于区分两种导致 "missing viewer context" 的情形：
			//   md_nil=true  → (a) OperatorMetadata 未从 admin BFF 传播到 core
			//   md_nil=false 且 tid/uid==0 → (b) 调用方为平台管理员（tenant_id=0）
			//   md_nil=false 且 tid/uid!=0 → 链路正常（不应出现 403）
			// 诊断完成后删除此块。
			if tdbg, _ := transport.FromServerContext(ctx); tdbg != nil {
				if op := tdbg.Operation(); strings.Contains(op, "/oa/") {
					vtid, vuid := uint64(0), uint64(0)
					if vc, exist := viewer.FromContext(ctx); exist && vc != nil {
						vtid = vc.TenantID()
						vuid = vc.UserID()
					}
					log.Warnf("oa-viewer-diag: op=%s md_err=%v md_nil=%v tid=%d uid=%d",
						tdbg.Operation(), err, md == nil, vtid, vuid)
				}
			}

			return handler(ctx, req)
		}
	}
}

func metaDataToUserViewerContext(md *authenticationV1.OperatorMetadata, traceID string) viewer.Context {
	if md == nil {
		return nil
	}

	userViewer := appViewer.NewUserViewer(
		md.GetUserId(),
		md.GetTenantId(),
		md.GetOrgUnitId(),
		traceID,
		md.GetDataScope(),
	)
	return userViewer
}
