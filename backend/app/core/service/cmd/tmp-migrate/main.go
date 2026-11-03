// tmp-migrate 临时建表工具：对指定 DSN 跑一次 ent Schema.Create，
// 供演示数据 SQL 导入前准备表结构（与 EntClient 启动迁移同一 schema）。
//
// 用法：go run ./app/core/service/cmd/tmp-migrate -dsn "host=localhost port=5432 user=postgres password=*Abcd123456 dbname=xxx sslmode=disable"
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	entSchema "entgo.io/ent/dialect/sql/schema"
	_ "github.com/lib/pq"

	"go-wind-oa/app/core/service/internal/data/ent"
	_ "go-wind-oa/app/core/service/internal/data/ent/runtime"
)

func main() {
	dsn := flag.String("dsn", "", "postgres DSN")
	drop := flag.Bool("drop", false, "drop removed columns/indexes (atlas destructive)")
	flag.Parse()
	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "-dsn required")
		os.Exit(2)
	}

	client, err := ent.Open("postgres", *dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	opts := []entSchema.MigrateOption{}
	if *drop {
		opts = append(opts, entSchema.WithDropColumn(true), entSchema.WithDropIndex(true))
	}
	if err := client.Schema.Create(context.Background(), opts...); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("migrate ok")
}
