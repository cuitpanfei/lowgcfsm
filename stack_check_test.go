package fsm_test

import (
	"testing"

	fsm "github.com/cuitpanfei/lowgcfsm"
)

func RecursiveCall(count int) int {
	if fsm.IsRecursiveCall() {
		return count
	}
	if count > 2 {
		return count
	}
	return RecursiveCall(count + 1)
}
func TestStackCheck(t *testing.T) {
	callTimes := RecursiveCall(1)
	if callTimes > 2 {
		t.Fail()
	}
}
