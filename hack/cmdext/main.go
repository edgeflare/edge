// See https://github.com/PostgREST/postgrest/issues/1130
// For cases where it needs to run outside k8s cluster eg on docker
package main

import (
	"context"
	"log"
	"os"

	"github.com/edgeflare/edge/internal/util/postgrest"
	"github.com/edgeflare/edge/internal/util/zitadel"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	zitadel.Configure()

	ctx := context.Background()
	pool, _ := pgxpool.New(context.Background(), os.Getenv("CONN_STRING"))
	err := postgrest.RotateJwtKey(ctx, pool, os.Getenv("ZITADEL_JWK_URL"))
	if err != nil {
		log.Fatalln(err)
	}
}
