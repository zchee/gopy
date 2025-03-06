// Copyright 2025 The go-python Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"log/slog"
)

type logKey struct{}

// FromContext returns a logger with predefined values from a [context.Context].
func FromContext(ctx context.Context, args ...any) *slog.Logger {
	return ctx.Value(logKey{}).(*slog.Logger).With(args...)
}

// IntoContext takes a context and sets the logger as one of its values.
// Use [FromContext] function to retrieve the logger.
func IntoContext(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, logKey{}, log)
}
