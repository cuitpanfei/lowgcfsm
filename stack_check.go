package fsm

import (
	"runtime"
	"strings"
)

// 记录函数调用信息
type CallInfo struct {
	Function string
	File     string
	Line     int
}

// 获取当前调用栈信息
func getCallStack(skip int) []CallInfo {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(skip, pcs)
	if n == 0 {
		return nil
	}

	pcs = pcs[:n]
	frames := runtime.CallersFrames(pcs)

	var stack []CallInfo
	for {
		frame, more := frames.Next()
		stack = append(stack, CallInfo{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
		})
		if !more {
			break
		}
	}

	return stack
}

// 检查是否递归调用了指定函数
func IsRecursiveCall() bool {
	stack := getCallStack(3) // 跳过isRecursiveCall自身
	// IsRecursiveCall的调用者，假设叫FuncA
	// 在同一个调用栈中出现两次或以上表示存在递归调用FuncA
	first := stack[0]
	count := 0
	for _, call := range stack {
		if strings.Contains(call.Function, first.Function) {
			count++
			if count >= 2 {
				return true
			}
		}
	}

	return false
}
