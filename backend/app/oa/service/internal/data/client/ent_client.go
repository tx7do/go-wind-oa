package client

import (
	"github.com/go-kratos/kratos/v2/log"

	"entgo.io/ent/dialect/sql"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"

	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/kratos-bootstrap/bootstrap"
	entBootstrap "github.com/tx7do/kratos-bootstrap/database/ent"

	"go-wind-oa/app/oa/service/internal/data/ent"
	"go-wind-oa/app/oa/service/internal/data/ent/migrate"
	_ "go-wind-oa/app/oa/service/internal/data/ent/runtime"
)

// NewEntClient 创建 OA 模块的 Ent ORM 客户端。
//
// 构造方式与 go-wind-cms/app/core/service/internal/data/client/ent_client.go 同构：
// 由 entBootstrap.NewEntClient 接管驱动/连接配置，闭包内装配 *ent.Client，
// 并在配置允许时执行 Schema.Create 自动迁移（含外键）。
func NewEntClient(ctx *bootstrap.Context) (*entCrud.EntClient[*ent.Client], func(), error) {
	l := ctx.NewLoggerHelper("ent/data/oa-service")

	cfg := ctx.GetConfig()
	if cfg == nil || cfg.Data == nil {
		l.Fatalf("[ENT] failed getting config")
		return nil, func() {}, nil
	}

	cli, err := entBootstrap.NewEntClient(cfg, func(drv *sql.Driver) *ent.Client {
		client := ent.NewClient(
			ent.Driver(drv),
			ent.Log(func(a ...any) {
				l.Debug(a...)
			}),
		)
		if client == nil {
			l.Fatalf("[ENT] failed creating ent client")
			return nil
		}

		// run the auto migration tool
		if cfg.Data.Database.GetMigrate() {
			if err := client.Schema.Create(ctx.Context(), migrate.WithForeignKeys(true)); err != nil {
				l.Fatalf("[ENT] failed creating schema resources: %v", err)
			}
		}

		return client
	})
	if err != nil {
		log.Fatalf("[ENT] failed creating ent client: %v", err)
		return nil, func() {}, err
	}

	return cli, func() {
		if cleanErr := cli.Close(); cleanErr != nil {
			log.Errorf("[ENT] failed closing ent client: %v", cleanErr)
		}
	}, nil
}
