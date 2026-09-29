package perf

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	PoWPuzzles = uint64(1024)

	PoWDifficultyBits = uint(19)

	PoWTarget = ^uint64(0) >> PoWDifficultyBits

	MemoryTableSize = uint64(8_388_608)

	MemoryReads = uint64(10_000_000)

	WorkloadVersion = uint64(4)
)

type BenchmarkResult struct {
	Seed        [32]byte
	CPUHash     [32]byte
	MemoryProof [32]byte
	FinalHash   [32]byte

	Nonces  []uint64
	Puzzles uint64
	Elapsed time.Duration
}

func (r *BenchmarkResult) Score() float64 {
	secs := r.Elapsed.Seconds()
	if secs <= 0 {
		return 0
	}
	return 1.0 / secs
}

func DeriveSeed(startTimestamp int64, providerPubkey []byte) [32]byte {
	h := sha256.New()
	var ts [8]byte
	binary.BigEndian.PutUint64(ts[:], uint64(startTimestamp))
	h.Write(ts[:])
	h.Write(providerPubkey)
	var seed [32]byte
	copy(seed[:], h.Sum(nil))
	return seed
}

func puzzleHash(state [32]byte, index, nonce uint64) [32]byte {
	var buf [48]byte
	copy(buf[0:32], state[:])
	binary.BigEndian.PutUint64(buf[32:40], index)
	binary.BigEndian.PutUint64(buf[40:48], nonce)
	return sha256.Sum256(buf[:])
}

func solvePuzzle(state [32]byte, index, target uint64) (uint64, [32]byte) {
	workers := uint64(runtime.NumCPU())
	if workers == 0 {
		workers = 1
	}

	var (
		done     atomic.Bool
		mu       sync.Mutex
		winNonce uint64
		winHash  [32]byte
		wg       sync.WaitGroup
	)

	for w := uint64(0); w < workers; w++ {
		wg.Add(1)
		go func(w uint64) {
			defer wg.Done()
			for nonce := w; !done.Load(); nonce += workers {
				h := puzzleHash(state, index, nonce)
				if binary.BigEndian.Uint64(h[:8]) <= target {
					mu.Lock()
					if !done.Load() {
						winNonce, winHash = nonce, h
						done.Store(true)
					}
					mu.Unlock()
					return
				}
			}
		}(w)
	}
	wg.Wait()

	return winNonce, winHash
}

func runCPUChain(seed [32]byte, puzzles, target uint64) ([32]byte, []uint64) {
	state := seed
	nonces := make([]uint64, puzzles)
	for i := uint64(0); i < puzzles; i++ {
		nonce, h := solvePuzzle(state, i, target)
		nonces[i] = nonce
		state = h
	}
	return state, nonces
}

func RunBenchmark(seed [32]byte, puzzles uint64) BenchmarkResult {
	start := time.Now()

	cpuHash, nonces := runCPUChain(seed, puzzles, PoWTarget)

	table := buildMemoryTable(cpuHash, MemoryTableSize)

	memoryProof := randomReads(table, cpuHash, MemoryReads)

	combined := sha256.Sum256(append(cpuHash[:], memoryProof[:]...))

	return BenchmarkResult{
		Seed:        seed,
		CPUHash:     cpuHash,
		MemoryProof: memoryProof,
		FinalHash:   combined,
		Nonces:      nonces,
		Puzzles:     puzzles,
		Elapsed:     time.Since(start),
	}
}

func buildMemoryTable(seed [32]byte, entries uint64) [][32]byte {
	table := make([][32]byte, entries)
	h := sha256.New()
	prev := seed
	var idx [8]byte
	for i := uint64(0); i < entries; i++ {
		h.Reset()
		h.Write(prev[:])
		binary.BigEndian.PutUint64(idx[:], i)
		h.Write(idx[:])

		_ = h.Sum(table[i][:0])
		prev = table[i]
	}
	return table
}

func randomReads(table [][32]byte, seed [32]byte, reads uint64) [32]byte {
	n := uint64(len(table))
	h := sha256.New()

	idx := binary.BigEndian.Uint64(seed[:8]) % n
	for i := uint64(0); i < reads; i++ {
		entry := table[idx]
		h.Write(entry[:])

		idx = binary.BigEndian.Uint64(entry[:8]) % n
	}

	var proof [32]byte
	copy(proof[:], h.Sum(nil))
	return proof
}

func MemoryPhase(seed [32]byte) [32]byte {
	table := buildMemoryTable(seed, MemoryTableSize)
	return randomReads(table, seed, MemoryReads)
}

func VerifyCPUChain(seed [32]byte, nonces []uint64, puzzles uint64) ([32]byte, error) {
	return verifyCPUChain(seed, nonces, puzzles, PoWTarget)
}

func verifyCPUChain(seed [32]byte, nonces []uint64, puzzles, target uint64) ([32]byte, error) {
	if uint64(len(nonces)) != puzzles {
		return [32]byte{}, fmt.Errorf("nonce count mismatch: got %d, want %d", len(nonces), puzzles)
	}
	state := seed
	for i := uint64(0); i < puzzles; i++ {
		h := puzzleHash(state, i, nonces[i])
		if binary.BigEndian.Uint64(h[:8]) > target {
			return [32]byte{}, fmt.Errorf("puzzle %d: nonce %d does not meet target", i, nonces[i])
		}
		state = h
	}
	return state, nil
}

func VerifyWork(seed [32]byte, nonces []uint64, puzzles uint64) (cpuHash, finalHash [32]byte, err error) {
	cpuHash, err = VerifyCPUChain(seed, nonces, puzzles)
	if err != nil {
		return [32]byte{}, [32]byte{}, err
	}
	table := buildMemoryTable(cpuHash, MemoryTableSize)
	memoryProof := randomReads(table, cpuHash, MemoryReads)
	finalHash = sha256.Sum256(append(cpuHash[:], memoryProof[:]...))
	return cpuHash, finalHash, nil
}

const smallPoWTarget = ^uint64(0) >> 8

func RunBenchmarkSmall(seed [32]byte, puzzles uint64) BenchmarkResult {
	start := time.Now()

	cpuHash, nonces := runCPUChain(seed, puzzles, smallPoWTarget)

	table := buildMemoryTable(cpuHash, 1000)
	memoryProof := randomReads(table, cpuHash, 100)
	combined := sha256.Sum256(append(cpuHash[:], memoryProof[:]...))

	return BenchmarkResult{
		Seed:        seed,
		CPUHash:     cpuHash,
		MemoryProof: memoryProof,
		FinalHash:   combined,
		Nonces:      nonces,
		Puzzles:     puzzles,
		Elapsed:     time.Since(start),
	}
}

func VerifyCPUChainSmall(seed [32]byte, nonces []uint64, puzzles uint64) ([32]byte, error) {
	return verifyCPUChain(seed, nonces, puzzles, smallPoWTarget)
}

func SeedHex(seed [32]byte) string {
	return hex.EncodeToString(seed[:])
}
