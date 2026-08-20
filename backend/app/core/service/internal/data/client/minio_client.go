package client

import (
	"go-wind-oa/pkg/oss"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
)

func NewMinIoClient(ctx *bootstrap.Context) *oss.MinIOClient {
	return oss.NewMinIoClient(ctx.GetConfig(), ctx.GetLogger())
}
