package log

var (
	defaultOptions = Options{
		logDir:     "/home/www/logs/applogs",
		filename:   "default.log",
		maxSize:    500,
		maxAge:     1,
		maxBackups: 10,
		callerSkip: 1,
	}
)

type Options struct {
	logDir     string
	filename   string
	maxSize    int
	maxBackups int
	maxAge     int
	compress   bool
	callerSkip int
	debug      bool
}

type Option interface {
	apply(*Options)
}

type OptionFunc func(*Options)

func (o OptionFunc) apply(opts *Options) {
	o(opts)
}

// WithLogDir 设置日志文件存放目录
func WithLogDir(dir string) Option {
	return OptionFunc(func(options *Options) {
		options.logDir = dir
	})
}

// WithHistoryLogFileName 设置历史日志文件名称
func WithHistoryLogFileName(fileName string) Option {
	return OptionFunc(func(options *Options) {
		options.filename = fileName
	})
}

// WithMaxSize 设置最大日志文件最大size
func WithMaxSize(size int) Option {
	return OptionFunc(func(options *Options) {
		options.maxSize = size
	})
}

// WithMaxBackups 设置日志文件最大保存份数
func WithMaxBackups(backup int) Option {
	return OptionFunc(func(options *Options) {
		options.maxBackups = backup
	})
}

// WithMaxAge 设置日志文件最大保存天数
func WithMaxAge(maxAge int) Option {
	return OptionFunc(func(options *Options) {
		options.maxAge = maxAge
	})
}

// WithCompress 设置是否压缩
func WithCompress(b bool) Option {
	return OptionFunc(func(options *Options) {
		options.compress = b
	})
}

// WithCallerSkip 设置调用者跳过层级
func WithCallerSkip(skip int) Option {
	return OptionFunc(func(options *Options) {
		options.callerSkip = skip
	})
}

// WithDebug 设置是否为debug模式
func WithDebug(debug bool) Option {
	return OptionFunc(func(options *Options) {
		options.debug = debug
	})
}
