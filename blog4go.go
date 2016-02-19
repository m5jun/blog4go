// Copyright (c) 2015, huangjunwei <huangjunwei@youmi.net>. All rights reserved.

// Package blog4go provide an efficient and easy-to-use writers library for
// logging into files, console or sockets. Writers suports formatting
// string filtering and calling user defined hook in asynchronous mode.
package blog4go

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

const (
	// EOL end of a line
	EOL = '\n'
	// ESCAPE escape character
	ESCAPE = '\\'
	// PLACEHOLDER placeholder
	PLACEHOLDER = '%'
)

var (
	// blog is the singleton instance use for blog.write/writef
	blog Writer

	// global mutex log used for singlton
	singltonLock *sync.Mutex

	// DefaultBufferSize bufio buffer size
	DefaultBufferSize = 4096 // default memory page size
	// ErrInvalidFormat invalid format error
	ErrInvalidFormat = errors.New("Invalid format type.")
)

// Writer interface is a common definition of any writers in this package.
// Any struct implements Writer interface must implement functions below.
// Close is used for close the writer and free any elements if needed.
// write is an internal function that write pure message with specific
// logging level.
// writef is an internal function that formatting message with specific
// logging level. Placeholders in the format string will be replaced with
// args given.
// Both write and writef may have an asynchronous call of user defined
// function before write and writef function end..
type Writer interface {
	// Close do anything end before program end
	Close()

	// Level return logging level threshold
	Level() Level
	// SetLevel set logging level threshold
	SetLevel(level Level)

	// write/writef functions with different levels
	Debug(format string)
	Debugf(format string, args ...interface{})
	Trace(format string)
	Tracef(format string, args ...interface{})
	Info(format string)
	Infof(format string, args ...interface{})
	Warn(format string)
	Warnf(format string, args ...interface{})
	Error(format string)
	Errorf(format string, args ...interface{})
	Critical(format string)
	Criticalf(format string, args ...interface{})
}

func init() {
	singltonLock = new(sync.Mutex)
	DefaultBufferSize = os.Getpagesize()
}

// BLog struct is a threadsafe log writer inherit bufio.Writer
type BLog struct {
	// logging level
	// every message level exceed this level will be written
	level Level

	// input io
	in io.Writer

	// bufio.Writer object of the input io
	writer *bufio.Writer

	// exclusive lock while calling write function of bufio.Writer
	lock *sync.Mutex
}

// NewBLog create a BLog instance and return the pointer of it.
// fileName must be an absolute path to the destination log file
func NewBLog(in io.Writer) (blog *BLog) {
	blog = new(BLog)
	blog.in = in
	blog.level = DEBUG
	blog.lock = new(sync.Mutex)

	blog.writer = bufio.NewWriterSize(in, DefaultBufferSize)
	return
}

// write writes pure message with specific level
func (blog *BLog) write(level Level, format string) int {
	// 统计日志size
	var size = 0

	blog.lock.Lock()
	defer blog.lock.Unlock()

	blog.writer.Write(timeCache.format)
	blog.writer.WriteString(level.Prefix())
	blog.writer.WriteString(format)
	blog.writer.WriteByte(EOL)

	size = len(timeCache.format) + len(level.Prefix()) + len(format) + 1
	return size
}

// write formats message with specific level and write it
func (blog *BLog) writef(level Level, format string, args ...interface{}) int {
	// 格式化构造message
	// 边解析边输出
	// 使用 % 作占位符
	blog.lock.Lock()
	defer blog.lock.Unlock()

	// 统计日志size
	var size = 0

	// 识别占位符标记
	var tag = false
	var tagPos int
	// 转义字符标记
	var escape = false
	// 在处理的args 下标
	var n int
	// 未输出的，第一个普通字符位置
	var last int
	var s int

	blog.writer.Write(timeCache.format)
	blog.writer.WriteString(level.Prefix())

	size += len(timeCache.format) + len(level.Prefix())

	for i, v := range format {
		if tag {
			switch v {
			case 'd', 'f', 'v', 'b', 'o', 'x', 'X', 'c', 'p', 't', 's', 'T', 'q', 'U', 'e', 'E', 'g', 'G':
				if escape {
					escape = false
				}

				s, _ = blog.writer.WriteString(fmt.Sprintf(format[tagPos:i+1], args[n]))
				size += s
				n++
				last = i + 1
				tag = false
			//转义符
			case ESCAPE:
				if escape {
					blog.writer.WriteByte(ESCAPE)
					size++
				}
				escape = !escape
			//默认
			default:

			}

		} else {
			// 占位符，百分号
			if PLACEHOLDER == format[i] && !escape {
				tag = true
				tagPos = i
				s, _ = blog.writer.WriteString(format[last:i])
				size += s
				escape = false
			}
		}
	}
	blog.writer.WriteString(format[last:])
	blog.writer.WriteByte(EOL)

	size += len(format[last:]) + 1
	return size
}

// Flush flush buffer to disk
func (blog *BLog) flush() {
	blog.lock.Lock()
	defer blog.lock.Unlock()
	blog.writer.Flush()
}

// Close close file writer
func (blog *BLog) Close() {
	blog.lock.Lock()
	defer blog.lock.Unlock()

	blog.writer.Flush()
	blog.writer = nil
}

// In return the input io.Writer
func (blog *BLog) In() io.Writer {
	return blog.in
}

// Level return logging level threshold
func (blog *BLog) Level() Level {
	return blog.level
}

// SetLevel set logging level threshold
func (blog *BLog) SetLevel(level Level) *BLog {
	blog.level = level
	return blog
}

// resetFile resets file descriptor of the writer with specific file name
func (blog *BLog) resetFile(in io.Writer) (err error) {
	blog.lock.Lock()
	defer blog.lock.Unlock()
	blog.writer.Flush()

	blog.in = in
	blog.writer.Reset(in)

	return
}
