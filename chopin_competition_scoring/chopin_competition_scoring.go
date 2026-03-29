package main

import (
	"fmt"
	"sort"
)

// Contestant stores participant data.
type Contestant struct {
	Name       string
	Repertoire []string
	Scores     map[string][]int
}

// AddRepertoire returns a new contestant with updated repertoire.
func AddRepertoire(c Contestant, pieces []string) Contestant {
	newC := c
	newC.Repertoire = append([]string{}, c.Repertoire...)
	newC.Repertoire = append(newC.Repertoire, pieces...)
	return newC
}

// AddScore returns a new contestant with score added for a specific piece.
func AddScore(c Contestant, piece string, score int) Contestant {
	if score < 0 || score > 25 {
		return c
	}

	newC := c
	newC.Repertoire = append([]string{}, c.Repertoire...)

	newScores := make(map[string][]int, len(c.Scores))
	for k, v := range c.Scores {
		cp := append([]int{}, v...)
		newScores[k] = cp
	}
	newC.Scores = newScores

	newC.Scores[piece] = append(newC.Scores[piece], score)
	return newC
}

// CorrectedAverage calculates average with simple correction:
// if there are at least 3 scores, remove one min and one max, then average the rest.
func CorrectedAverage(scores []int) float64 {
	if len(scores) == 0 {
		return 0
	}

	cp := append([]int{}, scores...)
	sort.Ints(cp)

	if len(cp) >= 3 {
		cp = cp[1 : len(cp)-1]
	}

	sum := 0
	for _, s := range cp {
		sum += s
	}
	return float64(sum) / float64(len(cp))
}

// TotalPoints returns total corrected points for contestant (sum of piece averages).
func TotalPoints(c Contestant) float64 {
	total := 0.0
	for _, piece := range c.Repertoire {
		total += CorrectedAverage(c.Scores[piece])
	}
	return total
}

// Ranking returns contestants sorted by total points descending.
func Ranking(contestants []Contestant) []Contestant {
	cp := append([]Contestant{}, contestants...)
	sort.Slice(cp, func(i, j int) bool {
		return TotalPoints(cp[i]) > TotalPoints(cp[j])
	})
	return cp
}

// WinnerForPiece returns contestant with highest corrected score for given piece.
func WinnerForPiece(contestants []Contestant, piece string) (Contestant, bool) {
	if len(contestants) == 0 {
		return Contestant{}, false
	}

	best := contestants[0]
	bestScore := CorrectedAverage(best.Scores[piece])

	for i := 1; i < len(contestants); i++ {
		current := contestants[i]
		score := CorrectedAverage(current.Scores[piece])
		if score > bestScore {
			best = current
			bestScore = score
		}
	}
	return best, true
}

func printContestant(c Contestant) {
	fmt.Printf("Contestant: %s\n", c.Name)
	for _, piece := range c.Repertoire {
		fmt.Printf("  Piece: %-22s Scores: %-18v Corrected avg: %.2f\n",
			piece, c.Scores[piece], CorrectedAverage(c.Scores[piece]))
	}
	fmt.Printf("  TOTAL: %.2f\n\n", TotalPoints(c))
}

func main() {
	// 3 contestants, repertoire of 3 pieces, 5 judges
	pieces := []string{
		"Etude Op.10 No.1",
		"Nocturne Op.9 No.2",
		"Ballade No.1",
	}

	charles := Contestant{Name: "Charles Leclerc", Scores: map[string][]int{}}
	max := Contestant{Name: "Max Verstappen", Scores: map[string][]int{}}
	lewis := Contestant{Name: "Lewis Hamilton", Scores: map[string][]int{}}

	charles = AddRepertoire(charles, pieces)
	max = AddRepertoire(max, pieces)
	lewis = AddRepertoire(lewis, pieces)

	// Alex scores
	charles = AddScore(charles, "Etude Op.10 No.1", 23)
	charles = AddScore(charles, "Etude Op.10 No.1", 24)
	charles = AddScore(charles, "Etude Op.10 No.1", 22)
	charles = AddScore(charles, "Etude Op.10 No.1", 25)
	charles = AddScore(charles, "Etude Op.10 No.1", 21)

	charles = AddScore(charles, "Nocturne Op.9 No.2", 20)
	charles = AddScore(charles, "Nocturne Op.9 No.2", 21)
	charles = AddScore(charles, "Nocturne Op.9 No.2", 22)
	charles = AddScore(charles, "Nocturne Op.9 No.2", 19)
	charles = AddScore(charles, "Nocturne Op.9 No.2", 23)

	charles = AddScore(charles, "Ballade No.1", 24)
	charles = AddScore(charles, "Ballade No.1", 23)
	charles = AddScore(charles, "Ballade No.1", 25)
	charles = AddScore(charles, "Ballade No.1", 22)
	charles = AddScore(charles, "Ballade No.1", 24)

	// Bella scores
	max = AddScore(max, "Etude Op.10 No.1", 22)
	max = AddScore(max, "Etude Op.10 No.1", 23)
	max = AddScore(max, "Etude Op.10 No.1", 24)
	max = AddScore(max, "Etude Op.10 No.1", 21)
	max = AddScore(max, "Etude Op.10 No.1", 23)

	max = AddScore(max, "Nocturne Op.9 No.2", 24)
	max = AddScore(max, "Nocturne Op.9 No.2", 23)
	max = AddScore(max, "Nocturne Op.9 No.2", 22)
	max = AddScore(max, "Nocturne Op.9 No.2", 25)
	max = AddScore(max, "Nocturne Op.9 No.2", 24)

	max = AddScore(max, "Ballade No.1", 20)
	max = AddScore(max, "Ballade No.1", 21)
	max = AddScore(max, "Ballade No.1", 22)
	max = AddScore(max, "Ballade No.1", 23)
	max = AddScore(max, "Ballade No.1", 24)

	// Chris scores
	lewis = AddScore(lewis, "Etude Op.10 No.1", 25)
	lewis = AddScore(lewis, "Etude Op.10 No.1", 24)
	lewis = AddScore(lewis, "Etude Op.10 No.1", 25)
	lewis = AddScore(lewis, "Etude Op.10 No.1", 23)
	lewis = AddScore(lewis, "Etude Op.10 No.1", 22)

	lewis = AddScore(lewis, "Nocturne Op.9 No.2", 19)
	lewis = AddScore(lewis, "Nocturne Op.9 No.2", 20)
	lewis = AddScore(lewis, "Nocturne Op.9 No.2", 21)
	lewis = AddScore(lewis, "Nocturne Op.9 No.2", 22)
	lewis = AddScore(lewis, "Nocturne Op.9 No.2", 23)

	lewis = AddScore(lewis, "Ballade No.1", 25)
	lewis = AddScore(lewis, "Ballade No.1", 25)
	lewis = AddScore(lewis, "Ballade No.1", 24)
	lewis = AddScore(lewis, "Ballade No.1", 23)
	lewis = AddScore(lewis, "Ballade No.1", 24)

	contestants := []Contestant{charles, max, lewis}

	fmt.Println("=== CONTESTANTS DATA ===")
	for _, c := range contestants {
		printContestant(c)
	}

	fmt.Println("=== RANKING (TOTAL POINTS) ===")
	ranking := Ranking(contestants)
	for i, c := range ranking {
		fmt.Printf("%d. %s - %.2f\n", i+1, c.Name, TotalPoints(c))
	}

	fmt.Println("\n=== BEST CONTESTANT FOR EACH PIECE ===")
	for _, piece := range pieces {
		winner, ok := WinnerForPiece(contestants, piece)
		if !ok {
			fmt.Printf("%s: no data\n", piece)
			continue
		}
		fmt.Printf("%s -> %s (%.2f)\n", piece, winner.Name, CorrectedAverage(winner.Scores[piece]))
	}
}
