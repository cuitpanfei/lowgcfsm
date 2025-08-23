package fsm

import (
	"fmt"
	"math"
	"strconv"
	"sync"
	"sync/atomic"
)

// State 表示状态机的状态类型
type State int32

const (
	StateInInit State = math.MaxInt32
)

// Event 表示状态机的事件类型
type Event int32

// Transition 表示状态转移
type Transition struct {
	From  State
	Event Event
	To    State
}

// Handler 业务逻辑处理函数类型
type Handler func(fsm *FSM, from State, to State, event Event, args ...interface{})

// CallbackType 回调类型
type CallbackType int

const (
	BeforeEvent CallbackType = iota
	AfterEvent
	LeaveState
	EnterState
)

// TransitionTable 状态转移表接口
type TransitionTable interface {
	GetNextState(from State, event Event) (State, bool)
	GetCallback(cbType CallbackType, state State, event Event) Handler
}

// ArrayTransitionTable 基于数组的状态转移表，GC友好
type ArrayTransitionTable struct {
	maxStates    int32
	maxEvents    int32
	table        []State // 二维数组扁平化存储: table[state][event] = nextState
	beforeEvents []Handler
	afterEvents  []Handler
	leaveStates  []Handler
	enterStates  []Handler
}

// NewArrayTransitionTable 创建新的数组状态转移表
func NewArrayTransitionTable(transitions []Transition) *ArrayTransitionTable {
	maxStates, maxEvents := getMaxStatesAndEvents(transitions)
	t := &ArrayTransitionTable{
		maxStates:    maxStates,
		maxEvents:    maxEvents,
		table:        make([]State, maxStates*maxEvents),
		beforeEvents: make([]Handler, maxStates*maxEvents),
		afterEvents:  make([]Handler, maxStates*maxEvents),
		leaveStates:  make([]Handler, maxStates),
		enterStates:  make([]Handler, maxStates),
	}

	// 初始化表格，默认无效状态
	for i := range t.table {
		t.table[i] = StateInInit
	}

	// 填充转移规则
	for _, trans := range transitions {
		if StateInInit == trans.From || StateInInit == trans.To {
			panic(strconv.Itoa(int(StateInInit)) + " is invalid state")
		}
		index := int32(trans.From)*maxEvents + int32(trans.Event)
		if index < int32(len(t.table)) {
			t.table[index] = trans.To
		}
	}

	return t
}

func getMaxStatesAndEvents(transitions []Transition) (maxStates, maxEvents int32) {
	for _, trans := range transitions {
		if trans.From > State(maxStates) {
			maxStates = int32(trans.From)
		}
		if trans.To > State(maxStates) {
			maxStates = int32(trans.To)
		}
		if trans.Event > Event(maxEvents) {
			maxEvents = int32(trans.Event)
		}
	}
	return maxStates + 1, maxEvents + 1
}
func (t *ArrayTransitionTable) PrintTable() {
	fmt.Println("Transition Table:")
	fmt.Println("From\tEvent\tTo")
	for i := range t.table {
		from := State(int32(i) / t.maxEvents)
		event := Event(int32(i) % t.maxEvents)
		to := t.table[i]
		fmt.Printf("%d\t%d\t%d\n", from, event, to)
	}
}

// RegisterCallback 注册回调函数
func (t *ArrayTransitionTable) RegisterCallback(cbType CallbackType, state State, event Event, handler Handler) {
	switch cbType {
	case BeforeEvent:
		index := int32(state)*t.maxEvents + int32(event)
		if index < int32(len(t.beforeEvents)) {
			t.beforeEvents[index] = handler
		}
	case AfterEvent:
		index := int32(state)*t.maxEvents + int32(event)
		if index < int32(len(t.afterEvents)) {
			t.afterEvents[index] = handler
		}
	case LeaveState:
		if int32(state) < int32(len(t.leaveStates)) {
			t.leaveStates[state] = handler
		}
	case EnterState:
		if int32(state) < int32(len(t.enterStates)) {
			t.enterStates[state] = handler
		}
	}
}

// GetNextState 获取下一个状态
func (t *ArrayTransitionTable) GetNextState(from State, event Event) (State, bool) {
	index := int32(from)*t.maxEvents + int32(event)
	if index >= int32(len(t.table)) || t.table[index] == StateInInit {
		return StateInInit, false
	}
	return t.table[index], true
}

// GetCallback 获取回调函数
func (t *ArrayTransitionTable) GetCallback(cbType CallbackType, state State, event Event) Handler {
	switch cbType {
	case BeforeEvent:
		index := int32(state)*t.maxEvents + int32(event)
		if index < int32(len(t.beforeEvents)) {
			return t.beforeEvents[index]
		}
	case AfterEvent:
		index := int32(state)*t.maxEvents + int32(event)
		if index < int32(len(t.afterEvents)) {
			return t.afterEvents[index]
		}
	case LeaveState:
		if int32(state) < int32(len(t.leaveStates)) {
			return t.leaveStates[state]
		}
	case EnterState:
		if int32(state) < int32(len(t.enterStates)) {
			return t.enterStates[state]
		}
	}
	return nil
}

// FSM 有限状态机实例
type FSM struct {
	state           int32 // 使用int32保证原子操作
	transitionTable *ArrayTransitionTable
	data            interface{} // 用户自定义数据，可用于存储业务状态
	id              string      // 状态机ID，用于标识
	mu              sync.Mutex  // 锁
}

// NewFSM 创建新的状态机实例
func NewFSM(id string, initialState State, transitionTable *ArrayTransitionTable, data interface{}) *FSM {
	return &FSM{
		state:           int32(initialState),
		transitionTable: transitionTable,
		id:              id,
	}
}

// CurrentState 获取当前状态（原子读取）
func (f *FSM) CurrentState() State {
	return State(atomic.LoadInt32(&f.state))
}

// ID 获取状态机ID
func (f *FSM) ID() string {
	return f.id
}

// Trigger 触发事件（原子状态切换）
func (f *FSM) Trigger(event Event, args ...interface{}) bool {
	// 先检查状态是否匹配，避免不必要的锁竞争
	current := f.CurrentState()
	if _, ok := f.transitionTable.GetNextState(current, event); !ok {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for {
		current := f.CurrentState()
		nextState, ok := f.transitionTable.GetNextState(current, event)
		if !ok {
			return false
		}

		// 执行before事件回调
		if handler := f.transitionTable.GetCallback(BeforeEvent, current, event); handler != nil {
			handler(f, current, nextState, event, args...)
		}

		// 执行leave状态回调
		if handler := f.transitionTable.GetCallback(LeaveState, current, event); handler != nil {
			handler(f, current, nextState, event, args...)
		}

		// 使用CAS原子操作确保状态切换的原子性
		if atomic.CompareAndSwapInt32(&f.state, int32(current), int32(nextState)) {
			// 执行enter状态回调
			if handler := f.transitionTable.GetCallback(EnterState, nextState, event); handler != nil {
				handler(f, current, nextState, event, args...)
			}

			// 执行after事件回调
			if handler := f.transitionTable.GetCallback(AfterEvent, current, event); handler != nil {
				handler(f, current, nextState, event, args...)
			}

			return true
		}
		// 如果CAS失败，说明状态已被其他goroutine修改，重试
	}
}

// FsmPool 状态机对象池，用于管理大量状态机实例
type FsmPool struct {
	pool            []FSM
	transitionTable *ArrayTransitionTable
	mu              sync.Mutex
	freeIndices     []int
	allocatedCount  int32
}

// NewFsmPool 创建状态机池
func NewFsmPool(size int, initialState State, transitionTable *ArrayTransitionTable) *FsmPool {
	pool := &FsmPool{
		pool:            make([]FSM, size),
		transitionTable: transitionTable,
		freeIndices:     make([]int, 0, size),
	}

	// 初始化所有状态机
	for i := range pool.pool {
		pool.pool[i] = FSM{
			state:           int32(initialState),
			transitionTable: transitionTable,
			id:              fmt.Sprintf("fsm-%d", i),
		}
		pool.freeIndices = append(pool.freeIndices, i)
	}

	return pool
}

// Allocate 从池中分配一个状态机实例
func (p *FsmPool) Allocate() *FSM {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.freeIndices) == 0 {
		return nil
	}

	index := p.freeIndices[len(p.freeIndices)-1]
	p.freeIndices = p.freeIndices[:len(p.freeIndices)-1]
	atomic.AddInt32(&p.allocatedCount, 1)

	return &p.pool[index]
}

// Release 释放状态机实例回池中
func (p *FsmPool) Release(fsm *FSM) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 找到FSM在池中的索引
	for i := range p.pool {
		if &p.pool[i] == fsm {
			p.freeIndices = append(p.freeIndices, i)
			atomic.AddInt32(&p.allocatedCount, -1)
			// 清空数据
			break
		}
	}
}

// AllocatedCount 获取已分配的状态机数量
func (p *FsmPool) AllocatedCount() int {
	return int(atomic.LoadInt32(&p.allocatedCount))
}

// Size 获取池大小
func (p *FsmPool) Size() int {
	return len(p.pool)
}
