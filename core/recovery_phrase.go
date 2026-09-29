package core

import (
	"crypto/ed25519"
	"crypto/sha256"
	_ "embed"
	"errors"
	"fmt"
	"strings"
)

const RecoveryPhraseWords = 24

//go:embed wordlist_english.txt
var wordlistEnglish string

var (
	phraseWords = strings.Split(strings.TrimSuffix(wordlistEnglish, "\n"), "\n")
	phraseIndex = func() map[string]int {
		m := make(map[string]int, len(phraseWords))
		for i, w := range phraseWords {
			m[w] = i
		}
		return m
	}()
)

var ErrPhraseWordCount = errors.New("recovery phrase must have 24 words")

var ErrPhraseUnknownWord = errors.New("unknown recovery phrase word")

var ErrPhraseChecksum = errors.New("recovery phrase checksum does not match")

var ErrPhraseSeedLength = errors.New("recovery phrase seed must be 32 bytes")

func PhraseFromSeed(seed []byte) (string, error) {
	if len(seed) != ed25519.SeedSize {
		return "", fmt.Errorf("%w, got %d", ErrPhraseSeedLength, len(seed))
	}
	sum := sha256.Sum256(seed)
	data := append(append(make([]byte, 0, ed25519.SeedSize+1), seed...), sum[0])

	words := make([]string, 0, RecoveryPhraseWords)
	var acc, bits uint
	for _, b := range data {
		acc = acc<<8 | uint(b)
		bits += 8
		if bits >= 11 {
			bits -= 11
			words = append(words, phraseWords[acc>>bits])
			acc &= 1<<bits - 1
		}
	}
	return strings.Join(words, " "), nil
}

func SeedFromPhrase(phrase string) ([]byte, error) {
	words := strings.Fields(strings.ToLower(phrase))
	if len(words) != RecoveryPhraseWords {
		return nil, fmt.Errorf("%w, got %d", ErrPhraseWordCount, len(words))
	}

	data := make([]byte, 0, ed25519.SeedSize+1)
	var acc, bits uint
	for i, w := range words {
		idx, ok := phraseIndex[w]
		if !ok {
			return nil, fmt.Errorf("%w at position %d", ErrPhraseUnknownWord, i+1)
		}
		acc = acc<<11 | uint(idx)
		bits += 11
		for bits >= 8 {
			bits -= 8
			data = append(data, byte(acc>>bits))
			acc &= 1<<bits - 1
		}
	}

	seed := data[:ed25519.SeedSize:ed25519.SeedSize]
	if sum := sha256.Sum256(seed); sum[0] != data[ed25519.SeedSize] {
		return nil, fmt.Errorf("%w: check each word and its order", ErrPhraseChecksum)
	}
	return seed, nil
}
