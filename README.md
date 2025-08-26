# LowGC FSM - 高性能低GC状态机库

[![Go Reference](https://pkg.go.dev/badge/github.com/cuitpanfei/lowgcfsm.svg)](https://pkg.go.dev/github.com/cuitpanfei/lowgcfsm)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/cuitpanfei/lowgcfsm)](https://goreportcard.com/report/github.com/cuitpanfei/lowgcfsm)

LowGC FSM 是一个专为高性能、低垃圾回收(GC)压力设计的 Go 语言有限状态机库。它采用数组存储状态转移表，避免了频繁的内存分配，特别适合高并发场景。

## 特性

- 🚀 **高性能**: 单次状态转换仅需约 8 纳秒，并发状态下仅需约 1.2 纳秒
- 🧹 **低 GC 压力**: 零内存分配操作，减少垃圾回收压力
- 🔒 **并发安全**: 内置互斥锁和原子操作，确保并发环境下的安全性
- 📊 **对象池支持**: 提供 FSM 对象池，支持大量状态机实例的高效管理
- 🔧 **灵活回调**: 支持多种事件回调（BeforeEvent、AfterEvent、LeaveState、EnterState）
- 📝 **类型安全**: 使用强类型状态和事件定义

## 性能基准测试

```bash

user@debian:~/workspace/lowGcFsm$ go test -benchmem -run=^$ -bench ^Benchmark github.com/cuitpanfei/lowgcfsm -count 10
goos: linux
goarch: amd64
pkg: github.com/cuitpanfei/lowgcfsm
cpu: 12th Gen Intel(R) Core(TM) i5-12400

# 单线程状态转换
BenchmarkStateTransition-12                     159289903                7.476 ns/op           0 B/op          0 allocs/op
BenchmarkStateTransition-12                     156254066                7.648 ns/op           0 B/op          0 allocs/op
BenchmarkStateTransition-12                     157630310                7.563 ns/op           0 B/op          0 allocs/op
BenchmarkStateTransition-12                     159605096                7.534 ns/op           0 B/op          0 allocs/op
BenchmarkStateTransition-12                     157512194                7.606 ns/op           0 B/op          0 allocs/op
BenchmarkStateTransition-12                     157444098                7.616 ns/op           0 B/op          0 allocs/op
BenchmarkStateTransition-12                     156883615                7.645 ns/op           0 B/op          0 allocs/op
BenchmarkStateTransition-12                     157281100                7.623 ns/op           0 B/op          0 allocs/op
BenchmarkStateTransition-12                     157175672                7.628 ns/op           0 B/op          0 allocs/op
BenchmarkStateTransition-12                     156349323                7.648 ns/op           0 B/op          0 allocs/op

# 并发状态转换（12线程）
BenchmarkConcurrentStateTransition-12           1000000000               1.235 ns/op           0 B/op          0 allocs/op
BenchmarkConcurrentStateTransition-12           966631150                1.228 ns/op           0 B/op          0 allocs/op
BenchmarkConcurrentStateTransition-12           952432707                1.228 ns/op           0 B/op          0 allocs/op
BenchmarkConcurrentStateTransition-12           957008276                1.230 ns/op           0 B/op          0 allocs/op
BenchmarkConcurrentStateTransition-12           961320153                1.231 ns/op           0 B/op          0 allocs/op
BenchmarkConcurrentStateTransition-12           962295644                1.262 ns/op           0 B/op          0 allocs/op
BenchmarkConcurrentStateTransition-12           926938942                1.231 ns/op           0 B/op          0 allocs/op
BenchmarkConcurrentStateTransition-12           957016732                1.231 ns/op           0 B/op          0 allocs/op
BenchmarkConcurrentStateTransition-12           938427033                1.236 ns/op           0 B/op          0 allocs/op
BenchmarkConcurrentStateTransition-12           957605809                1.242 ns/op           0 B/op          0 allocs/op

# 对象池分配
BenchmarkFsmPoolAllocation-12                     567037              1827 ns/op                 1.000 allocated               0 B/op          0 allocs/op
BenchmarkFsmPoolAllocation-12                     614850              1837 ns/op                 1.000 allocated               0 B/op          0 allocs/op
BenchmarkFsmPoolAllocation-12                     595543              1842 ns/op                 1.000 allocated               0 B/op          0 allocs/op
BenchmarkFsmPoolAllocation-12                     626511              1839 ns/op                 1.000 allocated               0 B/op          0 allocs/op
BenchmarkFsmPoolAllocation-12                     641132              1875 ns/op                 1.000 allocated               0 B/op          0 allocs/op
BenchmarkFsmPoolAllocation-12                     620276              1867 ns/op                 1.000 allocated               0 B/op          0 allocs/op
BenchmarkFsmPoolAllocation-12                     636678              1856 ns/op                 1.000 allocated               0 B/op          0 allocs/op
BenchmarkFsmPoolAllocation-12                     650496              1843 ns/op                 1.000 allocated               0 B/op          0 allocs/op
BenchmarkFsmPoolAllocation-12                     654006              1847 ns/op                 1.000 allocated               0 B/op          0 allocs/op
BenchmarkFsmPoolAllocation-12                     648277              1845 ns/op                 1.000 allocated               0 B/op          0 allocs/op
PASS
ok      github.com/cuitpanfei/lowgcfsm  36.663s
```

## 安装

```bash
go get github.com/cuitpanfei/lowgcfsm
```

## 快速开始

### 定义状态和事件

```go
const (
    StateIdle fsm.State = iota
    StateRunning
    StatePaused
    StateStopped
)

const (
    EventStart fsm.Event = iota
    EventPause
    EventResume
    EventStop
)
```

### 创建状态转移表

```go
transitions := []fsm.Transition{
    {From: StateIdle, Event: EventStart, To: StateRunning},
    {From: StateRunning, Event: EventPause, To: StatePaused},
    {From: StateRunning, Event: EventStop, To: StateStopped},
    {From: StatePaused, Event: EventResume, To: StateRunning},
    {From: StatePaused, Event: EventStop, To: StateStopped},
}

table := fsm.NewArrayTransitionTable(transitions)
```

### 创建状态机实例

```go
fsmInstance := fsm.NewFSM("my-fsm", StateIdle, table, nil)
```

### 注册回调函数

```go
table.RegisterCallback(fsm.EnterState, StateRunning, func(f *fsm.FSM, from, to fsm.State, event fsm.Event, args ...any) {
    fmt.Printf("Entered running state from %d\n", from)
})
```

### 触发状态转换

```go
// 触发事件
success := fsmInstance.Trigger(EventStart)
if success {
    fmt.Println("State transition successful")
} else {
    fmt.Println("Invalid state transition")
}
```

### 使用对象池

```go
// 创建对象池
pool := fsm.NewFsmPool(1000, StateIdle, table)

// 从池中获取状态机
fsmInstance := pool.Allocate()
if fsmInstance != nil {
    defer pool.Release(fsmInstance)
    
    // 使用状态机
    fsmInstance.Trigger(EventStart)
}
```

## API 文档

完整的 API 文档请参考 [GoDoc](https://pkg.go.dev/github.com/cuitpanfei/lowgcfsm)。

## 使用场景

- 网络协议状态管理
- 游戏状态管理
- 工作流引擎
- 物联网设备状态管理
- 高并发服务器状态管理

## 贡献

欢迎提交 Issue 和 Pull Request！对于重大更改，请先开 Issue 讨论您想要更改的内容。

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 作者

- [cuitpanfei](https://github.com/cuitpanfei)
