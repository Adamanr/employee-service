// internal/logging/logger.go
package utils

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

type CustomHandler struct {
	jsonHandler *slog.JSONHandler
}

func NewCustomHandler(consoleWriter io.Writer, fileWriter io.Writer, level slog.Level) *CustomHandler {
	jsonHandler := slog.NewJSONHandler(fileWriter, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{Key: "timestamp", Value: slog.StringValue(a.Value.Time().Format(time.RFC3339))}
			}
			if a.Key == "" {
				return slog.Attr{Key: "job", Value: slog.StringValue("employes_service")}
			}
			return a
		},
	})
	return &CustomHandler{jsonHandler: jsonHandler}
}

func (h *CustomHandler) Enabled(_ context.Context, level slog.Level) bool {
	return h.jsonHandler.Enabled(context.Background(), level)
}

func (h *CustomHandler) Handle(ctx context.Context, r slog.Record) error {
	if err := h.jsonHandler.Handle(ctx, r); err != nil {
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

	fmt.Fprintf(os.Stdout, "%s %s %s\n",
		color.New(color.FgBlue).Sprintf("%s", timeStr),
		colorFn("%-6s", levelStr),
		message,
	)

	return nil
}

func (h *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CustomHandler{jsonHandler: h.jsonHandler.WithAttrs(attrs).(*slog.JSONHandler)}
}

func (h *CustomHandler) WithGroup(name string) slog.Handler {
	return &CustomHandler{jsonHandler: h.jsonHandler.WithGroup(name).(*slog.JSONHandler)}
}

func SetupLogger(logFilePath string, level slog.Level) (*slog.Logger, error) {
	logFile := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}
	handler := NewCustomHandler(os.Stdout, logFile, level)
	return slog.New(handler), nil
}

func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
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
