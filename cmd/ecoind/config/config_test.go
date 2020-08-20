package config

import (
	"fmt"
	"testing"
)

func TestParseConfig(t *testing.T) {
	c, err := ParseConfig("./config.json")
	if err != nil {
		t.Errorf("error: %v\n", err)
	}
	fmt.Println(c)
	fmt.Println(c.CC)
	fmt.Println(c.PC.Seeds)
}
