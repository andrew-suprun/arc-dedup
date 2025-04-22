package app

import (
	"fmt"
	"testing"
)

func TestPadLeft(t *testing.T) {
	fmt.Printf("%q\n", padLeft("abc", 4))
	fmt.Printf("%q\n", padLeft("abc", 2))
}

func TestPadRight(t *testing.T) {
	fmt.Printf("%q\n", padRight("abc", 4))
	fmt.Printf("%q\n", padRight("abc", 2))
}
