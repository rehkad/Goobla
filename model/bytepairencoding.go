package model

import (
	"cmp"
	"context"
	"fmt"
	"iter"
	"log/slog"
	"strings"

	"github.com/dlclark/regexp2"
	heap "github.com/emirpasic/gods/v2/trees/binaryheap"
	"github.com/goobla/goobla/logutil"
)

type BytePairEncoding struct {
	pre   *regexp2.Regexp
	vocab *Vocabulary
}

var _ TextProcessor = (*BytePairEncoding)(nil)

func NewBytePairEncoding(pre string, vocab *Vocabulary) BytePairEncoding {
	return BytePairEncoding{
		pre:   regexp2.MustCompile(pre, regexp2.Unicode|regexp2.RE2),
		vocab: vocab,
	}
}

func (bpe BytePairEncoding) Vocabulary() *Vocabulary {
	return bpe.vocab
}

func (bpe BytePairEncoding) Is(id int32, special Special) bool {
	return bpe.vocab.Is(id, special)
}

func (bpe *BytePairEncoding) split(s string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for m, _ := bpe.pre.FindStringMatch(s); m != nil; m, _ = bpe.pre.FindNextMatch(m) {
			if !yield(m.String()) {
				break
			}
		}
	}
}

// fragment is a string fragment and their corresponding token IDs
type fragment struct {
	value string
	ids   []int32
}

// pair is a pair of runes and its rank
type pair struct {
	a, b  int
	rank  int
	value string
}

type merge struct {
	p, n  int
	runes []rune
}

func (bpe BytePairEncoding) Encode(s string, addSpecial bool) ([]int32, error) {
	fragments := []fragment{{value: s}}
	for _, special := range bpe.vocab.SpecialVocabulary() {
		id := bpe.vocab.Encode(special)
		for i := 0; i < len(fragments); i++ {
			frag := fragments[i]
			if len(frag.ids) > 0 {
				continue
			}

			var middle []fragment
			switch i := strings.Index(frag.value, special); {
			case i < 0:
				middle = append(middle, frag)
			case i > 0:
				middle = append(middle, fragment{value: frag.value[:i]})
				fallthrough
			default:
				middle = append(middle, fragment{value: special, ids: []int32{id}})
				if rest := frag.value[i+len(special):]; rest != "" {
					middle = append(middle, fragment{value: rest})
				}
			}

			fragments = append(fragments[:i], append(middle, fragments[i+1:]...)...)
		}
	}

	var ids []int32
	for _, frag := range fragments {
		if len(frag.ids) > 0 {
			ids = append(ids, frag.ids...)
			continue
		}

		for split := range bpe.split(frag.value) {
			var sb strings.Builder
			for _, b := range []byte(split) {
				r := rune(b)
				switch {
				case r == 0x00ad:
					r = 0x0143
				case r <= 0x0020:
					r = r + 0x0100
				case r >= 0x007e && r <= 0x00a0:
					r = r + 0x00a2
				}

				sb.WriteRune(r)
			}

			// short circuit if the fragment is in the vocabulary
			if id := bpe.vocab.Encode(sb.String()); id >= 0 {
				ids = append(ids, id)
				continue
			}

			runes := []rune(sb.String())
			merges := make([]merge, len(runes))
			for r := range runes {
				merges[r] = merge{
					p:     r - 1,
					n:     r + 1,
					runes: []rune{runes[r]},
				}
			}

			pairwise := func(a, b int) *pair {
				if a < 0 || b >= len(runes) {
					return nil
				}

				left, right := string(merges[a].runes), string(merges[b].runes)
				rank := bpe.vocab.Merge(left, right)
				if rank < 0 {
					return nil
				}

				return &pair{
					a:     a,
					b:     b,
					rank:  rank,
					value: left + right,
				}
			}

			pairs := heap.NewWith(func(i, j *pair) int {
				return cmp.Compare(i.rank, j.rank)
			})

			for i := range len(runes) - 1 {
				if pair := pairwise(i, i+1); pair != nil {
					pairs.Push(pair)
				}
			}

			for !pairs.Empty() {
				pair, _ := pairs.Pop()

				left, right := merges[pair.a], merges[pair.b]
				if len(left.runes) == 0 || len(right.runes) == 0 ||
					string(left.runes)+string(right.runes) != pair.value {
					continue
				}

				if id := bpe.vocab.Encode(pair.value); id < 0 {
					continue
				}

				merges[pair.a].runes = append(left.runes, right.runes...)
				merges[pair.b].runes = nil

				merges[pair.a].n = right.n
				if right.n < len(merges) {
					merges[right.n].p = pair.a
				}

				if pair := pairwise(merges[pair.a].p, pair.a); pair != nil {
					pairs.Push(pair)
				}

				if pair := pairwise(pair.a, merges[pair.a].n); pair != nil {
					pairs.Push(pair)
				}
			}

			for _, merge := range merges {
				if len(merge.runes) > 0 {
					if id := bpe.vocab.Encode(string(merge.runes)); id >= 0 {
						ids = append(ids, id)
					}
				}
			}
		}
	}

	slog.Log(context.Background(), logutil.LevelTrace, "encoded", "string", s, "ids", ids)

	if addSpecial && len(ids) > 0 {
		ids = bpe.vocab.addSpecials(ids)
	}

	return ids, nil
}

type lazyIdsString struct {
	ids []int32
}

func (l lazyIdsString) LogValue() slog.Value {
	return slog.AnyValue(fmt.Sprint(l.ids))
}

func (bpe BytePairEncoding) Decode(ids []int32) (string, error) {
	var sb strings.Builder
	for _, id := range ids {
		for _, r := range bpe.vocab.Decode(id) {
			switch {
			case r == 0x0100:
				// this produces 0x00 aka NULL
				continue
			case r == 0x0143:
				r = 0x00ad
			case r > 0x0100 && r <= 0x0120:
				r = r - 0x0100
			case r > 0x0120 && r <= 0x0142:
				r = r - 0x00a2
			}

			// NOTE: not using WriteRune here because it writes the UTF-8
			// encoding of the rune which is _not_ what we want
			if err := sb.WriteByte(byte(r)); err != nil {
				return "", err
			}
		}
	}

	slog.Log(context.Background(), logutil.LevelTrace, "decoded", "string", sb.String(), "from", lazyIdsString{ids: ids})
	return sb.String(), nil
}
