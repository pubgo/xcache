package xcache_test

import (
	"fmt"
	"testing"
)

func TestName(t *testing.T) {
	var d interface{} = ([]byte)(nil)
	fmt.Println(d)
	fmt.Println(d.([]byte) == nil)
}
