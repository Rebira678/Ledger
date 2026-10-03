package main

import (
	"fmt"
	"regexp"
)

func main() {
	cbeBalanceRe := regexp.MustCompile(`(?i)balance\s*(?:is\s+|[:#]\s*)?(?:ETB\s*)?([0-9][0-9,]*(?:\.[0-9]{0,2})?)`)
	
	s1 := "Your balance is 16,032.76Br."
	s2 := "Your current balance is ETB1,204.70. Thanks"
	
	m1 := cbeBalanceRe.FindStringSubmatch(s1)
	m2 := cbeBalanceRe.FindStringSubmatch(s2)
	
	fmt.Printf("M1: %q\n", m1)
	fmt.Printf("M2: %q\n", m2)
}
