package utils

import (
	"fmt"
	"runtime/debug"
	"slices"
	"strings"
	"time"
)

type Logger struct {
	path []string
	name string
}

type LoggerLevel int

type loggerLevelTreeNode struct {
	level    LoggerLevel
	nodeName string
	children map[string]*loggerLevelTreeNode
	parent   *loggerLevelTreeNode
}

const (
	Default LoggerLevel = iota // equal loggerLevelTree.level
	Trace
	Debug
	Info
	Warn
	Error
)

var loggerLevelTree = newLoggerLevelTreeNode("default", Warn, nil)

func newLoggerLevelTreeNode(name string, level LoggerLevel, parent *loggerLevelTreeNode) *loggerLevelTreeNode {
	return &loggerLevelTreeNode{
		level:    level,
		nodeName: name,
		children: make(map[string]*loggerLevelTreeNode),
		parent:   parent,
	}
}

func SetDefualtLoggerLevel(level LoggerLevel) {
	if level == Default {
		return
	}
	loggerLevelTree.level = level
}

func GetDefaultLoggerLevel() LoggerLevel {
	return loggerLevelTree.level
}

func getTreeNode(name []string) *loggerLevelTreeNode {
	children := loggerLevelTree.children
	var child *loggerLevelTreeNode
	for _, n := range name {
		if child, ok := children[n]; ok {
			children = child.children
			continue
		}
		return nil
	}
	return child
}

func getOrCreateTreeNode(name []string) *loggerLevelTreeNode {
	children := loggerLevelTree.children
	parent := loggerLevelTree
	var child *loggerLevelTreeNode
	for _, n := range name {
		if child, ok := children[n]; ok {
			children = child.children
			parent = child
			continue
		}
		newNode := newLoggerLevelTreeNode(n, Default, parent)
		children[n] = newNode
		children = newNode.children
		parent = newNode
	}
	return child
}

func SetLoggerLevel(name string, level LoggerLevel) {
	path := strings.Split(name, "/")
	node := getOrCreateTreeNode(path)
	node.level = level
}

func GetLoggerLevel(name string) LoggerLevel {
	path := strings.Split(name, "/")
	return GetLoggerLevelByPath(path)
}

func GetLoggerLevelByPath(path []string) LoggerLevel {
	node := getTreeNode(path)
	if node == nil || node.level == Default {
		return GetDefaultLoggerLevel()
	}
	return node.level
}

// seperator: /
func NewLogger(name string) *Logger {
	return &Logger{
		path: strings.Split(name, "/"),
		name: name,
	}
}

func (l *Logger) GetLogger(name string) *Logger {
	newName := append(slices.Clone(l.path), strings.Split(name, "/")...)
	return &Logger{
		path: newName,
		name: l.name + "/" + name,
	}
}

func canDoLog(currentLevel LoggerLevel, targetLevel LoggerLevel) bool {
	if targetLevel == Default {
		panic("error target level")
	}
	if currentLevel == Default {
		return canDoLog(GetDefaultLoggerLevel(), targetLevel)
	}
	return int(currentLevel) <= int(targetLevel)
}

func getLevelName(level LoggerLevel) string {
	switch level {
	case Trace:
		return "Trace"
	case Debug:
		return "Debug"
	case Info:
		return "Info "
	case Warn:
		return "Warn "
	case Error:
		return "Error"
	}
	return "     "
}

func printLogf(level LoggerLevel, name string, format string, args []any) {
	if level == Default {
		return
	}
	var builder strings.Builder
	builder.WriteString(time.Now().Format("2006-01-02 15:04:05.000000 MST"))
	builder.WriteString(" [")
	builder.WriteString(getLevelName(level))
	builder.WriteString("] ")
	builder.WriteString(name)
	builder.WriteString(" : ")
	builder.WriteString(format)
	builder.WriteString("\n")
	fmt.Printf(builder.String(), args...)
}

func printLogln(level LoggerLevel, name string, args []any) {
	if level == Default {
		return
	}
	var builder strings.Builder
	builder.WriteString(time.Now().Format("2006-01-02 15:04:05.000000 MST"))
	builder.WriteString(" [")
	builder.WriteString(getLevelName(level))
	builder.WriteString("] ")
	builder.WriteString(name)
	builder.WriteString(" : ")
	fmt.Println(append([]any{builder.String()}, args...)...)
}

func (l *Logger) Trace(format string, args ...any) {
	if canDoLog(GetLoggerLevelByPath(l.path), Trace) {
		printLogf(Trace, l.name, format, args)
		debug.PrintStack()
	}
}

func (l *Logger) Debug(format string, args ...any) {
	if canDoLog(GetLoggerLevelByPath(l.path), Debug) {
		printLogf(Debug, l.name, format, args)
	}
}

func (l *Logger) Info(format string, args ...any) {
	if canDoLog(GetLoggerLevelByPath(l.path), Info) {
		printLogf(Info, l.name, format, args)
	}
}

func (l *Logger) Warn(format string, args ...any) {
	if canDoLog(GetLoggerLevelByPath(l.path), Warn) {
		printLogf(Warn, l.name, format, args)
	}
}

func (l *Logger) Error(format string, args ...any) {
	if canDoLog(GetLoggerLevelByPath(l.path), Error) {
		printLogf(Error, l.name, format, args)
	}
}

func (l *Logger) Traceln(args ...any) {
	if canDoLog(GetLoggerLevelByPath(l.path), Trace) {
		printLogln(Trace, l.name, args)
		debug.PrintStack()
	}
}

func (l *Logger) Debugln(args ...any) {
	if canDoLog(GetLoggerLevelByPath(l.path), Debug) {
		printLogln(Debug, l.name, args)
	}
}

func (l *Logger) Infoln(args ...any) {
	if canDoLog(GetLoggerLevelByPath(l.path), Info) {
		printLogln(Info, l.name, args)
	}
}

func (l *Logger) Warnln(args ...any) {
	if canDoLog(GetLoggerLevelByPath(l.path), Warn) {
		printLogln(Warn, l.name, args)
	}
}

func (l *Logger) Errorln(args ...any) {
	if canDoLog(GetLoggerLevelByPath(l.path), Error) {
		printLogln(Error, l.name, args)
	}
}
