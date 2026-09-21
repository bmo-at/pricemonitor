package debug

import (
	"context"
	"log/slog"
	"os"

	model "github.com/bmo-at/pricemonitor/internal/model/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func New() *DBTX {
	dbtx := new(DBTX)

	dbtx.logger = slog.New(slog.NewTextHandler(os.Stdout, nil)).With("component", "database-debug-dbtx")

	return dbtx
}

type DBTX struct {
	logger *slog.Logger
}

// CopyFrom implements [model.DBTX].
func (d *DBTX) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	d.logger.Debug("CopyFrom", "ctx", ctx, "tableName", tableName.Sanitize(), "columnNames", columnNames, "rowSrc", rowSrc)

	return 0, nil
}

// Exec implements [model.DBTX].
func (d *DBTX) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	d.logger.Debug("CopyFrom", "ctx", ctx, "sql", sql, "arguments", arguments)
	return pgconn.CommandTag{}, nil
}

// Query implements [model.DBTX].
func (d *DBTX) Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error) {
	d.logger.Debug("CopyFrom", "ctx", ctx, "sql", sql, "arguments", arguments)
	return nil, nil
}

// QueryRow implements [model.DBTX].
func (d *DBTX) QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row {
	d.logger.Debug("CopyFrom", "ctx", ctx, "sql", sql, "arguments", arguments)
	return &Row{logger: d.logger}
}

var _ model.DBTX = (*DBTX)(nil)

type Row struct {
	logger *slog.Logger
}

// Scan implements [pgx.Row].
func (r *Row) Scan(dest ...any) error {
	r.logger.Debug("Scan", "dest", dest)
	return nil
}

var _ pgx.Row = (*Row)(nil)
