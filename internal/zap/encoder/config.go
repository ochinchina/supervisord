package encoder

import (
	"strings"
	"time"

	"go.uber.org/zap/zapcore"
)

// NewEncoderConfig returns an EncoderConfig with default settings.
func NewEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		MessageKey:     "msg",
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "name",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeLevel:    AbbrLevelEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

// NewDevelopmentEncoderConfig returns an EncoderConfig which is intended
// for local development.
func NewDevelopmentEncoderConfig() zapcore.EncoderConfig {
	cfg := NewEncoderConfig()
	cfg.EncodeTime = JustTimeEncoder
	cfg.EncodeDuration = zapcore.StringDurationEncoder
	return cfg
}

// JustTimeEncoder is a timestamp encoder function which encodes time
// as a simple time of day, without a date.  Intended for development and testing.
// Not good in a production system, where you probably need to know the date.
//
//     encConfig := flume.EncoderConfig{}
//     encConfig.EncodeTime = flume.JustTimeEncoder
//
func JustTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("15:04:05.000"))
}

// AbbrLevelEncoder encodes logging levels to the strings in the log entries.
// Encodes levels as 3-char abbreviations in upper case.
//
//     encConfig := flume.EncoderConfig{}
//     encConfig.EncodeTime = flume.AbbrLevelEncoder
//
func AbbrLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch l {
	case zapcore.DebugLevel:
		enc.AppendString("DBG")
	case zapcore.InfoLevel:
		enc.AppendString("INF")
	case zapcore.WarnLevel:
		enc.AppendString("WRN")
	case zapcore.ErrorLevel:
		enc.AppendString("ERR")
	case zapcore.PanicLevel, zapcore.FatalLevel, zapcore.DPanicLevel:
		enc.AppendString("FTL")
	default:
		s := l.String()
		if len(s) > 3 {
			s = s[:3]
		}
		enc.AppendString(strings.ToUpper(s))

	}
}
