package fsm_test

import (
	"runtime"
	"testing"

	fsm "github.com/cuitpanfei/lowgcfsm"
)

// 状态枚举
const (
	StateIdle fsm.State = iota
	StateRunning
	StatePaused
	StateStopped
)

// 事件枚举
const (
	EventStart fsm.Event = iota
	EventPause
	EventResume
	EventStop
)

// 创建测试用的状态转移表
func createTestTransitionTable() *fsm.ArrayTransitionTable {
	transitions := []fsm.Transition{
		{From: StateIdle, Event: EventStart, To: StateRunning},
		{From: StateRunning, Event: EventPause, To: StatePaused},
		{From: StateRunning, Event: EventStop, To: StateStopped},
		{From: StatePaused, Event: EventResume, To: StateRunning},
		{From: StatePaused, Event: EventStop, To: StateStopped},
	}

	return fsm.NewArrayTransitionTable(transitions)
}

// 测试基本状态转移
func TestBasicTransitions(t *testing.T) {
	table := createTestTransitionTable()
	fsmInstance := fsm.NewFSM(0, StateIdle, table)

	// 初始状态应为Idle
	if fsmInstance.CurrentState() != StateIdle {
		t.Errorf("Expected initial state %d, got %d", StateIdle, fsmInstance.CurrentState())
	}

	// 从Idle到Running
	if !fsmInstance.Trigger(EventStart) {
		t.Error("Failed to trigger EventStart from StateIdle")
	}
	if fsmInstance.CurrentState() != StateRunning {
		t.Errorf("Expected state %d, got %d", StateRunning, fsmInstance.CurrentState())
	}

	// 从Running到Paused
	if !fsmInstance.Trigger(EventPause) {
		t.Error("Failed to trigger EventPause from StateRunning")
	}
	if fsmInstance.CurrentState() != StatePaused {
		t.Errorf("Expected state %d, got %d", StatePaused, fsmInstance.CurrentState())
	}

	// 从Paused到Running
	if !fsmInstance.Trigger(EventResume) {
		t.Error("Failed to trigger EventResume from StatePaused")
	}
	if fsmInstance.CurrentState() != StateRunning {
		t.Errorf("Expected state %d, got %d", StateRunning, fsmInstance.CurrentState())
	}

	// 从Running到Stopped
	if !fsmInstance.Trigger(EventStop) {
		t.Error("Failed to trigger EventStop from StateRunning")
	}
	if fsmInstance.CurrentState() != StateStopped {
		t.Errorf("Expected state %d, got %d", StateStopped, fsmInstance.CurrentState())
	}

	// 无效转移测试
	if fsmInstance.Trigger(EventStart) {
		t.Error("Expected invalid transition from Stopped with EventStart")
	}
}

// 测试回调函数
func TestCallbacks(t *testing.T) {
	table := createTestTransitionTable()
	fsmInstance := fsm.NewFSM(0, StateIdle, table)

	var (
		beforeEventCalled bool
		afterEventCalled  bool
		leaveStateCalled  bool
		enterStateCalled  bool
	)

	// 注册回调函数
	table.RegisterCallback(fsm.BeforeEvent, StateIdle, EventStart, func(f *fsm.FSM, from, to fsm.State, event fsm.Event, args ...any) {
		beforeEventCalled = true
		if from != StateIdle || to != StateRunning || event != EventStart {
			t.Error("BeforeEvent callback received wrong parameters")
		}
	})

	table.RegisterCallback(fsm.AfterEvent, StateIdle, EventStart, func(f *fsm.FSM, from, to fsm.State, event fsm.Event, args ...any) {
		afterEventCalled = true
		if from != StateIdle || to != StateRunning || event != EventStart {
			t.Error("AfterEvent callback received wrong parameters")
		}
	})

	table.RegisterCallback(fsm.LeaveState, StateIdle, EventStart, func(f *fsm.FSM, from, to fsm.State, event fsm.Event, args ...any) {
		leaveStateCalled = true
		if from != StateIdle || to != StateRunning || event != EventStart {
			t.Error("LeaveState callback received wrong parameters")
		}
	})

	table.RegisterCallback(fsm.EnterState, StateRunning, EventStart, func(f *fsm.FSM, from, to fsm.State, event fsm.Event, args ...any) {
		enterStateCalled = true
		if from != StateIdle || to != StateRunning || event != EventStart {
			t.Error("EnterState callback received wrong parameters")
		}
	})

	// 触发事件
	fsmInstance.Trigger(EventStart)

	// 验证回调函数被调用
	if !beforeEventCalled {
		t.Error("BeforeEvent callback was not called")
	}
	if !afterEventCalled {
		t.Error("AfterEvent callback was not called")
	}
	if !leaveStateCalled {
		t.Error("LeaveState callback was not called")
	}
	if !enterStateCalled {
		t.Error("EnterState callback was not called")
	}
}

// 测试FSM池
func TestFsmPool(t *testing.T) {
	table := createTestTransitionTable()
	pool := fsm.NewFsmPool(10, StateIdle, table)

	// 分配FSM实例
	fsm1 := pool.Allocate()
	if fsm1 == nil {
		t.Error("Failed to allocate FSM from pool")
	}
	if pool.AllocatedCount() != 1 {
		t.Errorf("Expected 1 allocated FSM, got %d", pool.AllocatedCount())
	}

	// 测试分配到的FSM
	if fsm1.CurrentState() != StateIdle {
		t.Errorf("Expected initial state %d, got %d", StateIdle, fsm1.CurrentState())
	}

	// 释放FSM实例
	pool.Release(fsm1)
	if pool.AllocatedCount() != 0 {
		t.Errorf("Expected 0 allocated FSM after release, got %d", pool.AllocatedCount())
	}

	// 测试池满情况
	var fsms []*fsm.FSM
	for range 11 {
		f := pool.Allocate()
		if f != nil {
			fsms = append(fsms, f)
		}
	}

	if len(fsms) != 10 {
		t.Errorf("Expected 10 FSMs allocated, got %d", len(fsms))
	}
}

// 测试并发安全性
func TestConcurrentAccess(t *testing.T) {
	table := createTestTransitionTable()
	fsmInstance := fsm.NewFSM(0, StateIdle, table)

	// 启动多个goroutine同时触发事件
	done := make(chan bool, 10)
	for range 10 {
		go func() {
			// 每个goroutine尝试多次触发事件
			for range 100 {
				fsmInstance.Trigger(EventStart)
				fsmInstance.Trigger(EventPause)
				fsmInstance.Trigger(EventResume)
				fsmInstance.Trigger(EventStop)
			}
			done <- true
		}()
	}

	// 等待所有goroutine完成
	for range 10 {
		<-done
	}

	// 最终状态应该是有效的
	finalState := fsmInstance.CurrentState()
	if finalState != StateIdle && finalState != StateRunning &&
		finalState != StatePaused && finalState != StateStopped {
		t.Errorf("Invalid final state: %d", finalState)
	}
}

// 基准测试：状态转移性能
func BenchmarkStateTransition(b *testing.B) {
	table := createTestTransitionTable()
	fsmInstance := fsm.NewFSM(0, StateIdle, table)
	for b.Loop() {
		fsmInstance.Trigger(EventStart)
		fsmInstance.Trigger(EventPause)
		fsmInstance.Trigger(EventResume)
		fsmInstance.Trigger(EventStop)
	}
}

// 基准测试：并发状态转移性能
func BenchmarkConcurrentStateTransition(b *testing.B) {
	table := createTestTransitionTable()
	fsmInstance := fsm.NewFSM(0, StateIdle, table)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			fsmInstance.Trigger(EventStart)
			fsmInstance.Trigger(EventPause)
			fsmInstance.Trigger(EventResume)
			fsmInstance.Trigger(EventStop)
		}
	})
}

// 基准测试：FSM池分配性能
func BenchmarkFsmPoolAllocation(b *testing.B) {
	table := createTestTransitionTable()
	pool := fsm.NewFsmPool(6500, StateIdle, table)
	i, j := 0, 0
	for b.Loop() {
		i++
		fsmInstance := pool.Allocate()
		if fsmInstance != nil {
			j++
			pool.Release(fsmInstance)
		}
	}
	b.ReportMetric(float64(j)/float64(i), "allocated")
}

func TestCreateFsmPool(t *testing.T) {
	pool := fsm.NewFsmPool(10000, StateIdle, createTestTransitionTable())
	_ = pool
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	t.Logf("Allocated: %d KB", stats.Alloc/1024)
}
