package log

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	"github.com/hermeznetwork/tracerr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogEnvironment represents the possible log environments.
type LogEnvironment string

const (
	// EnvironmentProduction production log environment.
	EnvironmentProduction = LogEnvironment("production")
	// EnvironmentDevelopment development log environment.
	EnvironmentDevelopment = LogEnvironment("development")
)

// Config for log
type Config struct {
	// Environment defining the log format ("production" or "development").
	Environment LogEnvironment `mapstructure:"Environment" jsonschema:"enum=production,enum=development"`
	// Level of log. As lower value more logs are going to be generated
	Level string `mapstructure:"Level" jsonschema:"enum=debug,enum=info,enum=warn,enum=error,enum=dpanic,enum=panic,enum=fatal"`
	// Outputs
	Outputs []string `mapstructure:"Outputs"`
}

// root logger
var log atomic.Pointer[zap.SugaredLogger]

// InitLogger creates the logger with defined level.
// Outputs parameter defines the outputs where the logs will be sent.
// By default, outputs contains "stdout", which prints the
// logs at the output of the process. To add a log file as output, the path
// should be added at the outputs array. To avoid printing the logs but storing
// them on a file, can use []string{"pathtofile.log"}
func InitLogger(cfg Config) error {
	logger, err := newLogger(cfg)
	if err != nil {
		return err
	}

	log.Store(logger)

	return nil
}

// newLogger creates a new logger based on the provided configuration.
// It initializes the log level, output paths, and additional fields.
// The logger is built using the zap library and returns a pointer to the Logger struct.
// If an error occurs during the initialization or building of the logger, an error is returned.
func newLogger(cfg Config) (*zap.SugaredLogger, error) {
	var level zap.AtomicLevel
	err := level.UnmarshalText([]byte(cfg.Level))
	if err != nil {
		return nil, fmt.Errorf("error on setting log level: %s", err)
	}

	var zapCfg zap.Config

	switch cfg.Environment {
	case EnvironmentProduction:
		zapCfg = zap.NewProductionConfig()
	default:
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	zapCfg.Level = level
	zapCfg.OutputPaths = cfg.Outputs
	zapCfg.InitialFields = map[string]interface{}{
		"pid": os.Getpid(),
	}

	logger, err := zapCfg.Build()
	if err != nil {
		return nil, err
	}

	defer logger.Sync() //nolint:gosec,errcheck

	// skip 2 callers: one for our wrapper methods and one for the package functions
	withOptions := logger.WithOptions(zap.AddCallerSkip(2)) //nolint:gomnd

	return withOptions.Sugar(), nil
}

// getLogger returns the logger instance.
// If a logger instance is already loaded, it returns that instance.
// Otherwise, it creates a new logger with default settings and stores it for future use.
// The default logger level is set to "debug" and the output is directed to stderr.
// The logger is intended for use in a development environment.
func getLogger() *zap.SugaredLogger {
	l := log.Load()
	if l != nil {
		return l
	}

	// default level: debug
	zapLogger, err := newLogger(Config{
		Environment: EnvironmentDevelopment,
		Level:       "debug",
		Outputs:     []string{"stderr"},
	})
	if err != nil {
		panic(err)
	}

	log.Store(zapLogger)

	return log.Load()
}

// Info calls log.Info on the root Logger.
func Info(args ...interface{}) {
	getLogger().Info(args...)
}

// Warn calls log.Warn on the root Logger.
func Warn(args ...interface{}) {
	getLogger().Warn(args...)
}

// Error calls log.Error on the root Logger.
func Error(args ...interface{}) {
	args = appendStackTraceMaybeArgs(args)
	getLogger().Error(args...)
}

// Fatal calls log.Fatal on the root Logger.
func Fatal(args ...interface{}) {
	args = appendStackTraceMaybeArgs(args)
	getLogger().Fatal(args...)
}

// Debugf calls log.Debugf on the root Logger.
func Debugf(template string, args ...interface{}) {
	getLogger().Debugf(template, args...)
}

// Infof calls log.Infof on the root Logger.
func Infof(template string, args ...interface{}) {
	getLogger().Infof(template, args...)
}

// Warnf calls log.Warnf on the root Logger.
func Warnf(template string, args ...interface{}) {
	getLogger().Warnf(template, args...)
}

// Fatalf calls log.Fatalf on the root Logger.
func Fatalf(template string, args ...interface{}) {
	args = appendStackTraceMaybeArgs(args)
	getLogger().Fatalf(template, args...)
}

// Errorf calls log.Errorf on the root logger and stores the error message into
// the ErrorFile.
func Errorf(template string, args ...interface{}) {
	args = appendStackTraceMaybeArgs(args)
	getLogger().Errorf(template, args...)
}

// WithFields returns a new Logger (derived from the root one) with additional
// fields as per keyValuePairs.  The root Logger instance is not affected.
func WithFields(keyValuePairs ...interface{}) *zap.SugaredLogger {
	l := getLogger().With(keyValuePairs...)

	// since we are returning a new instance, remove one caller from the
	// stack, because we'll be calling the retruned Logger methods
	// directly, not the package functions.
	x := l.WithOptions(zap.AddCallerSkip(-1))
	l = x

	return l
}

// sprintStackTrace formats the given stack trace into a string.
// It skips the deepest frame because it belongs to the go runtime and is not relevant.
// The formatted string includes the file path, line number, and function name for each frame.
func sprintStackTrace(st []tracerr.Frame) string {
	builder := strings.Builder{}
	// Skip deepest frame because it belongs to the go runtime and we don't
	// care about it.
	if len(st) > 0 {
		st = st[:len(st)-1]
	}

	for _, f := range st {
		builder.WriteString(fmt.Sprintf("\n%s:%d %s()", f.Path, f.Line, f.Func))
	}

	builder.WriteString("\n")

	return builder.String()
}

// appendStackTraceMaybeArgs will append the stacktrace to the args
func appendStackTraceMaybeArgs(args []interface{}) []interface{} {
	for i := range args {
		if err, ok := args[i].(error); ok {
			err = tracerr.Wrap(err)
			st := tracerr.StackTrace(err)

			return append(args, sprintStackTrace(st))
		}
	}

	return args
}
