package progress

import (
	"fmt"
	"testing"
	"time"
)

func TestCalculate(t *testing.T) {
	start := time.Now()
	total := 10
	finished := 5
	time.Sleep(2 * time.Second)

	doneString, doneFloat, remainingFloat := Calculate(total, finished, start)
	fmt.Println(doneString, doneFloat, remainingFloat)
	expectedDoneString := "0.50"
	expectedDoneFloat := 0.5
	expectedRemainingFloat := 4.02

	if doneString != expectedDoneString {
		t.Errorf("Expected doneString to be %s, but got %s", expectedDoneString, doneString)
	}

	if doneFloat != expectedDoneFloat {
		t.Errorf("Expected doneFloat to be %f, but got %f", expectedDoneFloat, doneFloat)
	}

	if remainingFloat != expectedRemainingFloat {
		t.Errorf("Expected remainingFloat to be %f, but got %f", expectedRemainingFloat, remainingFloat)
	}
}

func TestName(t *testing.T) {
	t.Run("0.99", func(t *testing.T) {
		start := time.Now()
		time.Sleep(2 * time.Second)
		finished := 996
		total := 1000
		a, b, c := Calculate(total, finished, start)
		fmt.Println(a, b, c)
	})

	t.Run("1.00", func(t *testing.T) {
		start := time.Now()
		time.Sleep(2 * time.Second)
		finished := 1
		total := 126
		a, b, c := Calculate(total, finished, start)
		fmt.Println(a, b, c)
	})
}
