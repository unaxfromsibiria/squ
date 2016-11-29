package helpers_test

import (
	"squ/helpers"
	"testing"
)

func TestUidConcurrencyCreation(t *testing.T) {

	ptest := func(n int, resultChannel *chan []string) {
		result := make([]string, n)
		index := 0
		rand := helpers.NewSystemRandom()
		for index < n {
			result[index] = rand.Uid()
			index++
		}
		(*resultChannel) <- result
	}

	total := 15000
	n := 12
	i := 0
	results := make(chan []string, n)
	for i < n {
		t.Logf("Start channel %d", i+1)
		go ptest(total, &results)
		i++
	}
	i = 0
	variants := make(map[string]bool)
	for i < n {
		i++
		result := <-results
		t.Logf("End channel %d data size %d", i, len(result))
		for _, h := range result {
			variants[h] = true
		}
	}
	if len(variants) != n*total {
		t.Errorf("%d values ​​are repeated!", n*total-len(variants))
	}

}
