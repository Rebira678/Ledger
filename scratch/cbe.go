package main

import (
	"fmt"
	"regexp"
)

func main() {
	cbeBalanceRe := regexp.MustCompile(`(?i)balance\s*(?:is\s+|[:#]\s*)?(?:ETB\s*)?([0-9][0-9,]*(?:\.[0-9]{0,2})?)`)
	
	s1 := "Your balance is 16,032.76Br."
	s2 := "Your current balance is ETB1,204.70. Thanks"
	s3 := "Balance: 4,820.50 ETB"
	s4 := "Balance 1,620.50 ETB"
	
	for i, s := range []string{s1, s2, s3, s4} {
		m := cbeBalanceRe.FindStringSubmatch(s)
		if m != nil {
			fmt.Printf("Match %d: %q\n", i+1, m[1])
		} else {
			fmt.Printf("Match %d: FAILED\n", i+1)
		}
	}
}
