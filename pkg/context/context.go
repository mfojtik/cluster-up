package context

import (
	"context"
	"time"
)

type Context context.Context

var defaultTimeout = 10 * time.Second

func Run(runFn func(Context) error) error {
	ctx, cancelFn := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancelFn()
	return runFn(ctx)
}
