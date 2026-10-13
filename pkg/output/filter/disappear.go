package filter

import (
	"context" //nolint: gosec
	"fmt"
	"iter"
	"slices"
	"strings"

	"github.com/outscale/octl/pkg/debug"
	"github.com/outscale/octl/pkg/output/result"
	"github.com/outscale/octl/pkg/style"
	"github.com/samber/lo"
)

const (
	disappearBuffer = 100
)

type Disappear struct {
	resetOnNew bool

	iter           int
	last           []string
	lastResults    []result.Result
	current        []string
	currentResults []result.Result
}

func (d *Disappear) Filter(ctx context.Context, seq iter.Seq[result.Result]) iter.Seq[result.Result] {
	return func(yield func(result.Result) bool) {
		for v := range seq {
			if v.Error != nil {
				_ = yield(v)
				return
			}

			if v.Iter != d.iter {
				for i := range d.last {
					if d.last[i] != "" && !slices.Contains(d.current, d.last[i]) {
						debug.Println(d.last[i], "has disappeared")
						if !yield(disappear(d.lastResults[i])) {
							return
						}
					} else {
						debug.Println(d.last[i], "still exists")
					}
				}
				d.iter = v.Iter
				d.current, d.last = d.last, d.current
				d.currentResults, d.lastResults = d.lastResults, d.currentResults
				d.current = d.current[:0]
				d.currentResults = d.currentResults[:0]
			}
			if id, found := getID(v.Ok); found {
				d.current = append(d.current, id)
				d.currentResults = append(d.currentResults, v)
			}
			if !yield(v) {
				return
			}
		}
	}
}

func getID(v any) (string, bool) {
	var id string
	switch val := v.(type) {
	case map[string]any:
		ids := lo.FilterKeys(val, func(k string, _ any) bool {
			return strings.HasSuffix(k, "Id")
		})
		if len(ids) > 0 {
			id, _ = val[ids[0]].(string)
		}
	case []string:
		id = val[0]
	case []any:
		id, _ = val[0].(string)
	default:
		debug.Println(fmt.Sprintf("invalid type %T, no disappear", v))
	}
	return id, id != ""
}

func disappear(r result.Result) result.Result {
	// guarded by canDisappear
	return result.New(r, lo.Map(r.Ok.([]string), func(s string, _ int) string {
		return style.Strikethrough.Render(s)
	}))
}

func NewDisappear() *Disappear {
	return &Disappear{
		last:           make([]string, 0, dedupBuffer),
		lastResults:    make([]result.Result, 0, dedupBuffer),
		current:        make([]string, 0, dedupBuffer),
		currentResults: make([]result.Result, 0, dedupBuffer),
	}
}

var _ Interface = (*Disappear)(nil)
