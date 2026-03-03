package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	const trials = 1_000_000

	keep3, change3 := simulateMontyHall(3, 1, trials, rng)
	fmt.Printf("Wersja 3 pudel (%d prob):\n", trials)
	fmt.Printf("  Strategia bez zmiany wyboru: %.2f%%\n", keep3*100)
	fmt.Printf("  Strategia zmiany wyboru:     %.2f%%\n\n", change3*100)

	const N = 10
	const k = 5
	keepN, changeN := simulateMontyHall(N, k, trials, rng)
	fmt.Printf("Wersja rozszerzona (N=%d, k=%d, %d prob):\n", N, k, trials)
	fmt.Printf("  Strategia bez zmiany wyboru: %.2f%%\n", keepN*100)
	fmt.Printf("  Strategia zmiany wyboru:     %.2f%%\n", changeN*100)
}

func simulateMontyHall(n, k, trials int, rng *rand.Rand) (keepWinRate, switchWinRate float64) {
	if n < 3 {
		panic("n musi byc >= 3")
	}
	if k < 1 || k > n-2 {
		panic("k musi byc z zakresu [1, n-2]")
	}

	keepWins := 0
	switchWins := 0

	for range trials {
		prize := rng.Intn(n)
		playerChoice := rng.Intn(n)

		if playerChoice == prize {
			keepWins++
		}

		// Host opens k boxes (not the prize and not the player's choice)
		openedByHost := make([]bool, n)
		opened := 0
		for opened < k {
			box := rng.Intn(n)
			if box == playerChoice || box == prize || openedByHost[box] {
				continue
			}
			openedByHost[box] = true
			opened++
		}

		// Enumerate possible switches
		var possibleSwitches []int
		for box := range n {
			if box == playerChoice || openedByHost[box] {
				continue
			}
			possibleSwitches = append(possibleSwitches, box)
		}

		// Pick a box randomly
		switchChoice := possibleSwitches[rng.Intn(len(possibleSwitches))]
		if switchChoice == prize {
			switchWins++
		}
	}

	return float64(keepWins) / float64(trials), float64(switchWins) / float64(trials)
}
