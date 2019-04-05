package util

import (
	"fmt"
	"io"
	"strings"
)

type Logger interface {
	Debugf(format string, args ...interface{})

	Logf(format string, args ...interface{})

	SubLogger(name string) Logger
}

type StdoutLogger struct {
}

type PrefixLogger struct {
	prefix string
}

type PipeLogger struct {
	prefix string
	writer io.WriteCloser
	reader io.Reader
}

func NewPipeLogger() *PipeLogger {
	reader, writer := io.Pipe()
	return &PipeLogger{
		writer: writer,
		reader: reader,
	}
}

func NewPrefixLogger(prefix string) *PrefixLogger {
	return &PrefixLogger{
		prefix: prefix,
	}
}

func (l *PipeLogger) Writer() io.Writer {
	return l.writer
}

func (l *PipeLogger) Reader() io.Reader {
	return l.reader
}
func (l *PipeLogger) Close() error {
	return l.writer.Close()
}

func (l *PipeLogger) Debugf(format string, msg ...interface{}) {
	l.Logf(format, msg...)
}

func (l *PipeLogger) Logf(format string, msg ...interface{}) {
	message := l.prefix + " " + strings.Trim(fmt.Sprintf(format, msg...), "\n") + "\n"
	_, _ = l.writer.Write([]byte(message))
}

func (l *PipeLogger) SubLogger(name string) Logger {
	return &PipeLogger{
		reader: l.reader,
		writer: l.writer,
		prefix: l.prefix + " [" + name + "]",
	}
}

func (l *StdoutLogger) Logf(format string, msg ...interface{}) {
	fmt.Println(fmt.Sprintf(format, msg...))
}

func (l *StdoutLogger) SubLogger(name string) Logger {
	return l
}

func (l *StdoutLogger) Debugf(format string, msg ...interface{}) {
	l.Logf(format, msg...)
}

func (l *PrefixLogger) Logf(format string, msg ...interface{}) {
	message := strings.Trim(fmt.Sprintf(format, msg...), "\n")
	fmt.Println(l.prefix, message)
}

func (l *PrefixLogger) SubLogger(name string) Logger {
	return &PrefixLogger{
		prefix: l.prefix + " [" + name + "]",
	}
}

func (l *PrefixLogger) Debugf(format string, msg ...interface{}) {
	l.Logf(format, msg...)
}
