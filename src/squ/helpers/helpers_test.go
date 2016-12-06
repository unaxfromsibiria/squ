package helpers_test

import (
	"squ/helpers"
	"testing"
	"math"
)

const (
	deviation = 0.5 // %
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

func TestRandomQuestionEqualParts(t *testing.T) {
	worker := func(n int, resultChannel *chan byte) {
		index := 0
		rand := helpers.NewSystemRandom()
		for index < n {
			if rand.Question() {
				(*resultChannel) <- byte(1)
			} else {
				(*resultChannel) <- byte(0)
			}
			index++
		}
		(*resultChannel) <- byte(2)
	}
	countWorker := 10
	countIter := 4000
	resultReceiver := func(total int, resultChannel *chan byte, doneChannel *chan bool) {
		done := 0
		partTrue := 0
		partFalse := 0
		for done < total {
			res := <-(*resultChannel)
			switch int(res) {
			case 0:
				partFalse++
			case 1:
				partTrue++
			case 2:
				done++
			}
		}
		p := math.Abs(float64(partTrue) - float64(partFalse)) / float64(partFalse + partTrue) * 100.0
		t.Logf("%d == %d ? ~%f %%", partTrue, partFalse, p)
		(*doneChannel) <- p <= float64(deviation)
	}
	//
	resultChan := make(chan byte, countIter*countWorker/2)
	doneChan := make(chan bool, 1)
	go resultReceiver(countWorker, &resultChan, &doneChan)
	for i := 0; i < countWorker; i++ {
		go worker(countIter, &resultChan)
	}
	d := <-doneChan
	close(resultChan)
	close(doneChan)
	if !d {
		t.Error("failed")
	}
}


func TestSimpleFindTimeout(t *testing.T) {
	data1 := `{
		"p1": 4,
		"p2": 6,
		"timeauth": 100,
		"timeout": 20
	}`
	data2 := `{
		"p1": 4,
		"p2": 6,
		"timeauth": 100
	}`
	value1 := helpers.FindTimeout(&data1)
	value2 := helpers.FindTimeout(&data2)
	t.Logf("Results: %d, %d", value1, value2)
	if value1 != 20 * 1000 || value2 != helpers.DefaultCmdExecuteTimeOut * 1000 {
		t.Error("incorrect result")
	}
}
