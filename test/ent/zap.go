package ent

import (
	"context"
	"database/sql"
	"fmt"

	"entgo.io/ent/dialect"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (c *Client) DebugWithZap(l *zap.Logger) *Client {
	if c.debug {
		return c
	}
	cfg := c.config
	cfg.driver = debugEntDriver(c.driver, l)
	client := &Client{config: cfg}
	client.init()
	return client
}

// DebugDriver is a driver that logs all driver operations.
type DebugDriver struct {
	dialect.Driver             // underlying driver.
	logger         *zap.Logger // logger.
}

// Debug gets a driver and an optional logging function, and returns
// a new debugged-driver that prints all outgoing operations.
func debugEntDriver(d dialect.Driver, logger *zap.Logger) dialect.Driver {
	drv := &DebugDriver{d, logger}
	return drv
}

// Exec logs its params and calls the underlying driver Exec method.
func (d *DebugDriver) Exec(ctx context.Context, query string, args, v any) error {
	d.logger.Debug("driver.Exec", zap.String("query", query), zap.Any("args", args))
	return d.Driver.Exec(ctx, query, args, v)
}

// ExecContext logs its params and calls the underlying driver ExecContext method if it is supported.
func (d *DebugDriver) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	drv, ok := d.Driver.(interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
	})
	if !ok {
		return nil, fmt.Errorf("Driver.ExecContext is not supported")
	}
	d.logger.Debug("driver.ExecContext", zap.String("query", query), zap.Any("args", args))
	return drv.ExecContext(ctx, query, args...)
}

// Query logs its params and calls the underlying driver Query method.
func (d *DebugDriver) Query(ctx context.Context, query string, args, v any) error {
	d.logger.Debug("driver.Query", zap.String("query", query), zap.Any("args", args))
	return d.Driver.Query(ctx, query, args, v)
}

// QueryContext logs its params and calls the underlying driver QueryContext method if it is supported.
func (d *DebugDriver) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	drv, ok := d.Driver.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, fmt.Errorf("Driver.QueryContext is not supported")
	}
	d.logger.Debug("driver.QueryContext", zap.String("query", query), zap.Any("args", args))
	return drv.QueryContext(ctx, query, args...)
}

// Tx adds an log-id for the transaction and calls the underlying driver Tx command.
func (d *DebugDriver) Tx(ctx context.Context) (dialect.Tx, error) {
	tx, err := d.Driver.Tx(ctx)
	if err != nil {
		return nil, err
	}
	id := uuid.New().String()
	d.logger.Debug("driver.Tx", zap.String("id", id))
	return &DebugTx{tx, id, d.logger, ctx}, nil
}

// BeginTx adds an log-id for the transaction and calls the underlying driver BeginTx command if it is supported.
func (d *DebugDriver) BeginTx(ctx context.Context, opts *sql.TxOptions) (dialect.Tx, error) {
	drv, ok := d.Driver.(interface {
		BeginTx(context.Context, *sql.TxOptions) (dialect.Tx, error)
	})
	if !ok {
		return nil, fmt.Errorf("Driver.BeginTx is not supported")
	}
	tx, err := drv.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	id := uuid.New().String()
	d.logger.Debug("driver.BeginTx: started", zap.String("id", id))
	return &DebugTx{tx, id, d.logger, ctx}, nil
}

// DebugTx is a transaction implementation that logs all transaction operations.
type DebugTx struct {
	dialect.Tx                 // underlying transaction.
	id         string          // transaction logging id.
	logger     *zap.Logger     // logger.
	ctx        context.Context // underlying transaction context.
}

// Exec logs its params and calls the underlying transaction Exec method.
func (d *DebugTx) Exec(ctx context.Context, query string, args, v any) error {
	d.logger.Debug("Tx.Exec", zap.String("query", query), zap.Any("args", args), zap.String("id", d.id))
	return d.Tx.Exec(ctx, query, args, v)
}

// ExecContext logs its params and calls the underlying transaction ExecContext method if it is supported.
func (d *DebugTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	drv, ok := d.Tx.(interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
	})
	if !ok {
		return nil, fmt.Errorf("Tx.ExecContext is not supported")
	}
	d.logger.Debug("Tx.ExecContext", zap.String("query", query), zap.Any("args", args), zap.String("id", d.id))
	return drv.ExecContext(ctx, query, args...)
}

// Query logs its params and calls the underlying transaction Query method.
func (d *DebugTx) Query(ctx context.Context, query string, args, v any) error {
	d.logger.Debug("Tx.Query", zap.String("query", query), zap.Any("args", args), zap.String("id", d.id))
	return d.Tx.Query(ctx, query, args, v)
}

// QueryContext logs its params and calls the underlying transaction QueryContext method if it is supported.
func (d *DebugTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	drv, ok := d.Tx.(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return nil, fmt.Errorf("Tx.QueryContext is not supported")
	}
	d.logger.Debug("Tx.QueryContext", zap.String("query", query), zap.Any("args", args), zap.String("id", d.id))
	return drv.QueryContext(ctx, query, args...)
}

// Commit logs this step and calls the underlying transaction Commit method.
func (d *DebugTx) Commit() error {
	d.logger.Debug("Tx.Commit", zap.String("id", d.id))
	return d.Tx.Commit()
}

// Rollback logs this step and calls the underlying transaction Rollback method.
func (d *DebugTx) Rollback() error {
	d.logger.Debug("Tx.Rollback", zap.String("id", d.id))
	return d.Tx.Rollback()
}
