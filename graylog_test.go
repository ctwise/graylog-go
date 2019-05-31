package main

import (
	"fmt"
	"os/user"
	"testing"
)

func ExampleExpand() {
	fmt.Println(expand("line1\\nthen line2"))
	// Output:
	// line1
	// then line2
}

func TestExpandPath(t *testing.T) {
	path1 := expandPath("~/.graylog")

	usr, _ := user.Current()
	dir := usr.HomeDir

	if path1 != dir + "/.graylog" {
		t.Errorf("expandPath(\"~/.graylog\") = %s", path1)
	}
}
