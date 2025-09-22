package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/go-chi/chi/v5/middleware"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	MaxSize    = 100
	MaxBackups = 3
	MaxAge     = 28
)

type CustomHandler struct {
	handler slog.Handler
}

func NewCustomHandler(_ io.Writer, fileWriter io.Writer, level slog.Level) *CustomHandler {
	handler := slog.NewJSONHandler(fileWriter, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{Key: "timestamp", Value: slog.StringValue(a.Value.Time().Format(time.RFC3339))}
			}
			if a.Key == "" {
				return slog.Attr{Key: "job", Value: slog.StringValue("employes_service")}
			}
			return a
		},
	})
	return &CustomHandler{handler: handler}
}

func (h *CustomHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *CustomHandler) Handle(ctx context.Context, r slog.Record) error {
	if err := h.handler.Handle(ctx, r); err != nil {
		return err
	}

	var colorFn func(format string, args ...interface{}) string
	switch r.Level {
	case slog.LevelDebug:
		colorFn = color.New(color.FgCyan).Sprintf
	case slog.LevelInfo:
		colorFn = color.New(color.FgGreen).Sprintf
	case slog.LevelWarn:
		colorFn = color.New(color.FgYellow).Sprintf
	case slog.LevelError:
		colorFn = color.New(color.FgRed).Sprintf
	default:
		colorFn = color.New(color.FgWhite).Sprintf
	}

	var attrs []string
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, fmt.Sprintf("%s=%v", a.Key, a.Value))
		return true
	})
	attrStr := strings.Join(attrs, " ")

	timeStr := r.Time.Format("2006-01-02 15:04:05.000")
	levelStr := r.Level.String()
	message := r.Message
	if attrStr != "" {
		message = fmt.Sprintf("%s %s", message, attrStr)
	}

	if _, err := fmt.Fprintf(os.Stdout, "%s %s %s\n",
		color.New(color.FgBlue).Sprintf("%s", timeStr),
		colorFn("%-6s", levelStr),
		message,
	); err != nil {
		return err
	}

	return nil
}

func (h *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CustomHandler{handler: h.handler.WithAttrs(attrs)}
}

func (h *CustomHandler) WithGroup(name string) slog.Handler {
	return &CustomHandler{handler: h.handler.WithGroup(name)}
}

func SetupLogger(logFilePath string, level slog.Level) (*slog.Logger, error) {
	logFile := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    MaxSize,
		MaxBackups: MaxBackups,
		MaxAge:     MaxAge,
		Compress:   true,
	}
	handler := NewCustomHandler(os.Stdout, logFile, level)
	return slog.New(handler), nil
}

func Middleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			requestID := middleware.GetReqID(r.Context())
			if requestID == "" {
				requestID = "unknown"
			}

			logger = logger.With(slog.String("request_id", requestID))

			next.ServeHTTP(ww, r)

			logger.Info("HTTP request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Duration("duration", time.Since(start)),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			)
		})
	}
}
