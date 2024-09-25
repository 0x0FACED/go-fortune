package logger

import (
	"bytes"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ZapLogger struct {
	log    *zap.Logger
	logBuf *bytes.Buffer
	Logs   []string
}

func New() *ZapLogger {
	logBuf := &bytes.Buffer{}

	config := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    colorLevelEncoder,
		EncodeTime:     customTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	encoder := zapcore.NewConsoleEncoder(config)

	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(logBuf), zap.DebugLevel),
	)

	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1), zap.AddStacktrace(zapcore.ErrorLevel))

	return &ZapLogger{
		log:    logger,
		logBuf: logBuf,
	}
}

func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("[2006-01-02 | 15:04:05]"))
}

func colorLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	var colorCode string
	switch level {
	case zapcore.DebugLevel:
		colorCode = "\033[36m" // Cyan
	case zapcore.InfoLevel:
		colorCode = "\033[32m" // Green
	case zapcore.WarnLevel:
		colorCode = "\033[33m" // Yellow
	case zapcore.ErrorLevel:
		colorCode = "\033[31m" // Red
	default:
		colorCode = "\033[0m" // Default
	}
	enc.AppendString(colorCode + level.String() + "\033[0m")
}

// Converts ANSI color codes to HTML span with inline styles
func ansiToHTML(input string) string {
	// Pattern to match ANSI color codes
	re := regexp.MustCompile(`\033\[(\d+)m`)

	var result strings.Builder
	var lastIndex int

	// Map to keep track of the currently opened color styles
	var openTags []string

	// Replace ANSI color codes with HTML color spans
	result.WriteString("<pre>") // Use <pre> tag for preserving whitespace and formatting

	// Iterate over matches and replace ANSI color codes
	for _, match := range re.FindAllStringIndex(input, -1) {
		start := match[0]
		end := match[1]

		// Write text before the match
		if start > lastIndex {
			result.WriteString(input[lastIndex:start])
		}

		// Process the color code
		colorCode := input[start+2 : end-1]
		color, ok := colorMap[colorCode]
		if ok {
			// Close the previous color tag if any
			if len(openTags) > 0 {
				result.WriteString("</span>")
				openTags = nil
			}
			// Add the new color tag
			result.WriteString(`<span style="color: ` + color + `;">`)
			openTags = append(openTags, color)
		} else if colorCode == "0" {
			// Close all color tags on reset
			if len(openTags) > 0 {
				result.WriteString("</span>")
				openTags = nil
			}
		}

		lastIndex = end
	}

	// Write any remaining text
	if lastIndex < len(input) {
		result.WriteString(input[lastIndex:])
	}

	// Close any remaining open tags
	if len(openTags) > 0 {
		result.WriteString("</span>")
	}

	result.WriteString("</pre>")

	return result.String()
}

// Color mapping for ANSI codes
var colorMap = map[string]string{
	"31": "red",    // Red
	"32": "green",  // Green
	"33": "yellow", // Yellow
	"34": "blue",   // Blue
	"36": "cyan",   // Cyan
	// Add more colors as needed
}

func (z *ZapLogger) UpdateLogs() {
	htmlLogs := ansiToHTML(z.logBuf.String())
	z.Logs = []string{htmlLogs}
}

func (z *ZapLogger) ClearLogs() {
	z.logBuf.Reset()
	z.Logs = nil
}

func (z *ZapLogger) Info(wrappedMsg string, fields ...zap.Field) {
	z.log.Info(wrappedMsg, fields...)
	z.UpdateLogs()
}

func (z *ZapLogger) Debug(wrappedMsg string, fields ...zap.Field) {
	z.log.Debug(wrappedMsg, fields...)
	z.UpdateLogs()
}

func (z *ZapLogger) Error(wrappedMsg string, fields ...zap.Field) {
	z.log.Error(wrappedMsg, fields...)
	z.UpdateLogs()
}

func (z *ZapLogger) Fatal(wrappedMsg string, fields ...zap.Field) {
	z.log.Fatal(wrappedMsg, fields...)
	z.UpdateLogs()
}
