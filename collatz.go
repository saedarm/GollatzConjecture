package main

// Sequence returns every number visited from n down to 1, including n and 1.
func Sequence(n int) []int {
	seq := []int{n}
	for n != 1 {
		if n%2 == 0 {
			n /= 2
		} else {
			n = 3*n + 1
		}
		seq = append(seq, n)
	}
	return seq
}

// StoppingTimes returns how many steps each starting number from 1 to limit
// takes to reach 1. steps[i] is the stopping time for i; steps[0] is unused.
//
// It walks each number only until it drops below where it started, then
// reuses the answer already worked out for that smaller number.
func StoppingTimes(limit int) []int {
	steps := make([]int, limit+1)
	for i := 2; i <= limit; i++ {
		n, count := i, 0
		for n >= i {
			if n%2 == 0 {
				n /= 2
			} else {
				n = 3*n + 1
			}
			count++
		}
		steps[i] = count + steps[n]
	}
	return steps
}
