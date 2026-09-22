#!/usr/bin/env sh
set -e
export NOFRILLS_REGION=0
export NOFRILLS_NODE=0
export NOFRILLS_ENV=dev
export NOFRILLS_HTTP_ADDR=:3000
export NOFRILLS_DB_DSN='nofrillz:eJ3JNhbx2KTPPn2Dp2vXb4zN$@tcp(10.10.0.230:3306)/nofrillz?parseTime=true'
export NOFRILLS_REDIS_ADDR='127.0.0.1:6379'
export NOFRILLS_REFRESH_HMAC_SECRET='devsecret'
export NOFRILLS_ACCESS_TTL_MINUTES=30
export NOFRILLS_REFRESH_TTL_DAYS=365
go run ./cmd/api
