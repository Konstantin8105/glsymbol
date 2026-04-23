package glsymbol

import (
	"fmt"
	"os/exec"
	"testing"
)

func Test(t *testing.T) {
	_, err := exec.Command("go", "run", "demo.go", "-test=true").Output()
	if err != nil {
		t.Fatal(err)
		fmt.Println("Error:", err)
	}
}
