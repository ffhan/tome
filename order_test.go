package tome

import (
	"testing"
	"unsafe"
)

func TestOrderSize(t *testing.T) {
	o := Order{}
	t.Logf("sizeof: %db", unsafe.Sizeof(o))
}
