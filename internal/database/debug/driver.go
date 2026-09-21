package debug

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/samber/lo"
)

func init() {
	Register()
}

func Register() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil)).With("component", "database-debug-driver")

	sql.Register("debug", &Driver{
		logger,
	})
}

type Driver struct {
	logger *slog.Logger
}

func (d *Driver) Open(name string) (driver.Conn, error) {
	return &Conn{logger: d.logger}, nil
}

var (
	_ driver.Driver = (*Driver)(nil)
)

type Conn struct {
	logger *slog.Logger
}

// PrepareContext implements [driver.ConnPrepareContext].
func (c *Conn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	c.logger.Debug("PrepareContext", "ctx", ctx, "query", query)
	return &Stmt{logger: c.logger}, nil
}

// BeginTx implements [driver.ConnBeginTx].
func (c *Conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	c.logger.Debug("BeginTx", "ctx", ctx, "opts", opts)
	return &Tx{logger: c.logger}, nil
}

// QueryContext implements [driver.QueryerContext].
func (c *Conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	reduced := strings.TrimSuffix(lo.Reduce(args, func(agg string, item driver.NamedValue, index int) string {
		return agg + fmt.Sprintf("%+v,", item)
	}, ""), ",")
	c.logger.Debug("QueryContext", "ctx", ctx, "query", query, "args", reduced)
	return &Rows{logger: c.logger}, nil
}

// ExecContext implements [driver.ExecerContext].
func (c *Conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	reduced := strings.TrimSuffix(lo.Reduce(args, func(agg string, item driver.NamedValue, index int) string {
		return agg + fmt.Sprintf("%+v,", item)
	}, ""), ",")
	c.logger.Debug("ExecContext", "ctx", ctx, "query", query, "args", reduced)
	return &Result{logger: c.logger}, nil
}

// Close implements [driver.Conn].
func (c *Conn) Close() error {
	c.logger.Debug("closing connection")
	return nil
}

// Prepare implements [driver.Conn].
func (c *Conn) Prepare(query string) (driver.Stmt, error) {
	c.logger.Debug("preparing statement", "query", query)
	return &Stmt{logger: c.logger}, nil
}

func (c *Conn) Begin() (driver.Tx, error) {
	c.logger.Debug("beginning connection")
	return &Tx{logger: c.logger}, nil
}

// Ping implements [driver.Pinger].
func (c *Conn) Ping(ctx context.Context) error {
	c.logger.Debug("pinging connection")
	return nil
}

var (
	_ driver.Conn               = (*Conn)(nil)
	_ driver.Pinger             = (*Conn)(nil)
	_ driver.ExecerContext      = (*Conn)(nil)
	_ driver.QueryerContext     = (*Conn)(nil)
	_ driver.ConnBeginTx        = (*Conn)(nil)
	_ driver.ConnPrepareContext = (*Conn)(nil)
)

type Tx struct {
	logger *slog.Logger
}

// Commit implements [driver.Tx].
func (d *Tx) Commit() error {
	d.logger.Debug("commiting transaction")
	return nil
}

// Rollback implements [driver.Tx].
func (d *Tx) Rollback() error {
	d.logger.Debug("rolling back transaction")
	return nil
}

var (
	_ driver.Tx = (*Tx)(nil)
)

type Rows struct {
	logger *slog.Logger
}

// Close implements [driver.Rows].
func (d *Rows) Close() error {
	d.logger.Debug("closing rows")
	return nil
}

// Columns implements [driver.Rows].
func (d *Rows) Columns() []string {
	d.logger.Debug("getting columns of rows")
	return []string{}
}

// Next implements [driver.Rows].
func (d *Rows) Next(dest []driver.Value) error {
	d.logger.Debug("getting next row")
	return io.EOF
}

var (
	_ driver.Rows = (*Rows)(nil)
)

type Result struct {
	logger *slog.Logger
}

// LastInsertId implements [driver.Result].
func (d *Result) LastInsertId() (int64, error) {
	d.logger.Debug("getting last insert id")
	return 0, nil
}

// RowsAffected implements [driver.Result].
func (d *Result) RowsAffected() (int64, error) {
	d.logger.Debug("getting rows affected")
	return 0, nil
}

var (
	_ driver.Result = (*Result)(nil)
)

type Stmt struct {
	logger *slog.Logger
}

// Close implements [driver.Stmt].
func (d *Stmt) Close() error {
	d.logger.Debug("closing statement")
	return nil
}

// Exec implements [driver.Stmt].
func (d *Stmt) Exec(args []driver.Value) (driver.Result, error) {
	reduced := strings.TrimSuffix(lo.Reduce(args, func(agg string, item driver.Value, index int) string {
		return agg + fmt.Sprintf("%+v,", item)
	}, ""), ",")
	d.logger.Debug("statement Exec", "args", reduced)
	return &Result{logger: d.logger}, nil
}

// NumInput implements [driver.Stmt].
func (d *Stmt) NumInput() int {
	d.logger.Debug("getting last insert id")
	return -1
}

// Query implements [driver.Stmt].
func (d *Stmt) Query(args []driver.Value) (driver.Rows, error) {
	reduced := strings.TrimSuffix(lo.Reduce(args, func(agg string, item driver.Value, index int) string {
		return agg + fmt.Sprintf("%+v,", item)
	}, ""), ",")
	d.logger.Debug("statement Query", "args", reduced)
	return &Rows{logger: d.logger}, nil
}

var (
	_ driver.Stmt = (*Stmt)(nil)
)
