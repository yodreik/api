package sl

import "log/slog"

// Err returns an error attribute for slog.Record
func Err(err error) slog.Attr {
	return slog.Attr{Key: "error", Value: slog.StringValue(err.Error())}
}
