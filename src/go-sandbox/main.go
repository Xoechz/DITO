package main

import (
	"fmt"
	"sync"
	"time"
)

type Matrix struct {
	rows, cols int
	data       []int
}

func (m Matrix) get(row, col int) int {
	if row < 0 || row >= m.rows || col < 0 || col >= m.cols {
		panic("Index out of bounds")
	}
	return m.data[row*m.cols+col]
}

func (m Matrix) set(row, col, value int) {
	if row < 0 || row >= m.rows || col < 0 || col >= m.cols {
		panic("Index out of bounds")
	}
	m.data[row*m.cols+col] = value
}

func multiply(a, b Matrix) Matrix {
	if a.cols != b.rows {
		panic("Incompatible matrices")
	}

	result := Matrix{
		rows: a.rows,
		cols: b.cols,
		data: make([]int, a.rows*b.cols),
	}

	for i := 0; i < a.rows; i++ {
		for j := 0; j < b.cols; j++ {
			sum := 0
			for k := 0; k < a.cols; k++ {
				sum += a.get(i, k) * b.get(k, j)
			}
			result.set(i, j, sum)
		}
	}

	return result
}

func multiplyAsync(a, b Matrix) Matrix {
	if a.cols != b.rows {
		panic("Incompatible matrices")
	}

	result := Matrix{
		rows: a.rows,
		cols: b.cols,
		data: make([]int, a.rows*b.cols),
	}

	var wg = sync.WaitGroup{}

	for i := 0; i < a.rows; i++ {
		for j := 0; j < b.cols; j++ {
			wg.Add(1)
			go getMatrixMultiplicationCell(a, b, i, j, &result, &wg)
		}
	}

	wg.Wait()
	return result
}

func getMatrixMultiplicationCell(a Matrix, b Matrix, row int, col int, result *Matrix, wg *sync.WaitGroup) {
	sum := 0
	for k := 0; k < a.cols; k++ {
		sum += a.get(row, k) * b.get(k, col)
	}
	result.set(row, col, sum)
	wg.Done()
}

func (m Matrix) String() string {
	result := ""
	for i := 0; i < m.rows; i++ {
		for j := 0; j < m.cols; j++ {
			result += fmt.Sprintf("%6d ", m.get(i, j))
		}
		result += "\n"
	}
	return result
}

func main() {
	const testSize = 100
	const outputSize = 100

	a := Matrix{
		rows: outputSize,
		cols: testSize,
		data: make([]int, outputSize*testSize),
	}

	for i := 0; i < a.rows; i++ {
		for j := 0; j < a.cols; j++ {
			a.data[i*a.cols+j] = (i + j) % 10
		}
	}

	b := Matrix{
		rows: testSize,
		cols: outputSize,
		data: make([]int, testSize*outputSize),
	}

	for i := 0; i < b.rows; i++ {
		for j := 0; j < b.cols; j++ {
			b.data[i*b.cols+j] = (i - j) % 10
		}
	}

	startTime := time.Now()
	result := multiply(a, b)
	endTime := time.Now()

	startTimeAsync := time.Now()
	resultAsync := multiplyAsync(a, b)
	endTimeAsync := time.Now()
	fmt.Println("Resulting Matrix (Sync):")
	fmt.Println(result)
	fmt.Println("Resulting Matrix (Async):")
	fmt.Println(resultAsync)
	fmt.Printf("Sync Multiplication took %v\n", endTime.Sub(startTime))
	fmt.Printf("Async Multiplication took %v\n", endTimeAsync.Sub(startTimeAsync))
}
