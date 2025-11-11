package task

import (
	"math/rand"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func shuffle[T any](src []T) []T {
	dst := slices.Clone(src)
	rand.Shuffle(len(dst), func(i, j int) {
		dst[i], dst[j] = dst[j], dst[i]
	})
	return dst
}

func shuffleCount[T any](src []T, count int) [][]T {
	var out [][]T
	for range count {
		out = append(out, shuffle(src))
	}
	return out
}

func permutations[T any](src []T) [][]T {
	n := len(src)
	switch n {
	case 0:
		return [][]T{}
	case 1:
		return [][]T{{src[0]}}
	}
	var result [][]T
	for i := 0; i < n; i++ {
		fixed := src[i]
		rest := make([]T, 0, n-1)
		rest = append(rest, src[:i]...)
		rest = append(rest, src[i+1:]...)
		for _, perm := range permutations(rest) {
			newPerm := make([]T, 0, n)
			newPerm = append(newPerm, fixed)
			newPerm = append(newPerm, perm...)
			result = append(result, newPerm)
		}
	}
	return result
}

func TestSortTask(t *testing.T) {
	assert := assert.New(t)
	var arr []*Task
	path, _ := parseFilepath("test")
	prep := func(lines []string) {
		Lists.Empty(path)
		for ndx, line := range lines {
			task, err := ParseTask(&ndx, line)
			assert.NoError(err)
			assert.NotNil(task)
			Lists.Append(path, task)
		}
		CleanupRelations(path)
		Lists.Sort(path)
		arr = Lists[path].Tasks
	}
	t.Run("progress", func(t *testing.T) {
		for _, lineSet := range permutations([]string{
			"$p=unit/0/100/cat", "$p=unit/0/100",
			"$p=unit/0/100/catA", "$p=unit/0/100/catB",
		}) {
			prep(lineSet)
			assert.Equal("$p=unit/0/100/cat", arr[0].Norm())
			assert.Equal("$p=unit/0/100", arr[3].Norm())
			assert.Equal("$p=unit/0/100/catA", arr[1].Norm())
			assert.Equal("$p=unit/0/100/catB", arr[2].Norm())
		}
		for _, lineSet := range permutations([]string{
			"$p=unit/0/300", "$p=unit/10/100",
			"$p=unit/20/100", "$p=unit/200/1000",
		}) {
			prep(lineSet)
			assert.Equal("$p=unit/200/1000", arr[0].Norm())
			assert.Equal("$p=unit/20/100", arr[1].Norm())
			assert.Equal("$p=unit/10/100", arr[2].Norm())
			assert.Equal("$p=unit/0/300", arr[3].Norm())
		}
	})
	t.Run("priority", func(t *testing.T) {
		for _, lineSet := range permutations([]string{
			"(A) s", "s",
			"(B) s", "(C) s",
		}) {
			prep(lineSet)
			assert.Equal("(A) s", arr[0].Norm())
			assert.Equal("s", arr[3].Norm())
			assert.Equal("(B) s", arr[1].Norm())
			assert.Equal("(C) s", arr[2].Norm())
		}
	})
	t.Run("anti-priority", func(t *testing.T) {
		for _, lineSet := range shuffleCount([]string{
			"[0] $p=unit/0/100/cat", "$p=unit/0/100/cat",
			"(z) $p=unit/0/100", "$p=unit/0/100",
			"$p=unit/0/100/catA", "$p=unit/0/100/catB",
			"(A) s", "s", "[D] s",
			"(B) s", "(C) s",
			"+b +a", "s",
			"+b +a +z", "+z +b +0",
			"#a", "#ab", "#a #b #c",
			"@a", "@ab", "@a @b @c",
			"#a @a", "#a @b", "#b @a",
			"@a #b", "@a #b", "@b #a",
			"a +b", "+b",
			"c", "d",
			"a",
			"b $id=1",
			"(z) x $P=1",
			"z.1 $P=1",
			"[0] a.2 $P=1",
			"c",
			"$due=1w $dead=1w", "$due=1w $dead=2w",
			"$due=1w $end=1w", "$due=1w $end=2w",
		}, 5) {
			prep(lineSet)
			assert.Equal("$p=unit/0/100/cat", arr[0].Norm())
			assert.Equal("[0] $p=unit/0/100/cat", arr[1].Norm())
			assert.Equal("$p=unit/0/100/catA", arr[2].Norm())
			assert.Equal("$p=unit/0/100/catB", arr[3].Norm())
			assert.Equal("(z) $p=unit/0/100", arr[4].Norm())
			assert.Equal("$p=unit/0/100", arr[5].Norm())
			assert.Equal("$due=1w $end=1w", arr[6].Norm())
			assert.Equal("$due=1w $dead=1w", arr[7].Norm())
			assert.Equal("$due=1w $end=2w", arr[8].Norm())
			assert.Equal("$due=1w $dead=2w", arr[9].Norm())
			assert.Equal("(A) s", arr[10].Norm())
			assert.Equal("(B) s", arr[11].Norm())
			assert.Equal("(C) s", arr[12].Norm())
			assert.Equal("+z +b +0", arr[13].Norm())
			assert.Equal("+b +a", arr[14].Norm())
			assert.Equal("+b +a +z", arr[15].Norm())
			assert.Equal("a +b", arr[16].Norm())
			assert.Equal("+b", arr[17].Norm())
			assert.Equal("#a", arr[18].Norm())
			assert.Equal("#a #b #c", arr[19].Norm())
			assert.Equal("#a @a", arr[20].Norm())
			assert.Equal("#a @b", arr[21].Norm())
			assert.Equal("#ab", arr[22].Norm())
			assert.Equal("#b @a", arr[23].Norm())
			assert.Equal("@a", arr[24].Norm())
			assert.Equal("@a #b", arr[25].Norm())
			assert.Equal("@a #b", arr[26].Norm())
			assert.Equal("@a @b @c", arr[27].Norm())
			assert.Equal("@ab", arr[28].Norm())
			assert.Equal("@b #a", arr[29].Norm())
			assert.Equal("a", arr[30].Norm())
			assert.Equal("b $id=1", arr[31].Norm())
			assert.Equal("(z) x $P=1", arr[32].Norm())
			assert.Equal("z.1 $P=1", arr[33].Norm())
			assert.Equal("[0] a.2 $P=1", arr[34].Norm())
			assert.Equal("c", arr[35].Norm())
			assert.Equal("c", arr[36].Norm())
			assert.Equal("d", arr[37].Norm())
			assert.Equal("s", arr[38].Norm())
			assert.Equal("s", arr[39].Norm())
			assert.Equal("[D] s", arr[40].Norm())
		}
	})
	t.Run("hints", func(t *testing.T) {
		for _, lineSet := range shuffleCount([]string{
			"+b +a", "s",
			"+b +a +z", "+z +b +0",
		}, 25) {
			prep(lineSet)
			assert.Equal("+z +b +0", arr[0].Norm())
			assert.Equal("+b +a", arr[1].Norm())
			assert.Equal("+b +a +z", arr[2].Norm())
			assert.Equal("s", arr[3].Norm())
		}
		for _, lineSet := range shuffleCount([]string{
			"#a", "#ab", "#a #b #c",
			"@a", "@ab", "@a @b @c",
			"#a @a", "#a @b", "#b @a",
			"@a #b", "@a #b", "@b #a",
		}, 20) {
			prep(lineSet)
			assert.Equal("#a", arr[0].Norm())
			assert.Equal("#a #b #c", arr[1].Norm())
			assert.Equal("#a @a", arr[2].Norm())
			assert.Equal("#a @b", arr[3].Norm())
			assert.Equal("#ab", arr[4].Norm())
			assert.Equal("#b @a", arr[5].Norm())
			assert.Equal("@a", arr[6].Norm())
			assert.Equal("@a #b", arr[7].Norm())
			assert.Equal("@a #b", arr[8].Norm())
			assert.Equal("@a @b @c", arr[9].Norm())
			assert.Equal("@ab", arr[10].Norm())
		}
	})
	t.Run("texts", func(t *testing.T) {
		for _, lineSet := range permutations([]string{
			"a +b", "+b",
			"c", "d",
		}) {
			prep(lineSet)
			assert.Equal("a +b", arr[0].Norm())
			assert.Equal("+b", arr[1].Norm())
			assert.Equal("c", arr[2].Norm())
			assert.Equal("d", arr[3].Norm())
		}
	})
	t.Run("parent children", func(t *testing.T) {
		for _, lineSet := range permutations([]string{
			"a",
			"b $id=1",
			"z.1 $P=1",
			"a.2 $P=1",
			"c",
		}) {
			prep(lineSet)
			assert.Equal("a", arr[0].Norm())
			assert.Equal("b $id=1", arr[1].Norm())
			assert.Equal("a.2 $P=1", arr[2].Norm())
			assert.Equal("z.1 $P=1", arr[3].Norm())
			assert.Equal("c", arr[4].Norm())
		}
		for _, lineSet := range shuffleCount([]string{
			"$id=0", "$id=1", "$id=2",
			"$P=0", "$P=1", "$P=2",
		}, 20) {
			prep(lineSet)
			assert.Equal("$id=0", arr[0].Norm())
			assert.Equal("$P=0", arr[1].Norm())
			assert.Equal("$id=1", arr[2].Norm())
			assert.Equal("$P=1", arr[3].Norm())
			assert.Equal("$id=2", arr[4].Norm())
			assert.Equal("$P=2", arr[5].Norm())
		}
		for _, lineSet := range shuffleCount([]string{
			"$id=0", "$id=1", "$id=2",
			"$P=0", "$P=1", "$P=2",
			"$id=-1", "$id=4", "$id=5",
			"$P=-2", "$P=-3", "$P=-4",
		}, 20) {
			path, _ := parseFilepath("test")
			prep(lineSet)
			assert.Equal("$id=-1", Lists[path].Tasks[0].Norm())
			assert.Equal("$id=0", Lists[path].Tasks[1].Norm())
			assert.Equal("$P=0", Lists[path].Tasks[2].Norm())
			assert.Equal("$id=1", Lists[path].Tasks[3].Norm())
			assert.Equal("$P=1", Lists[path].Tasks[4].Norm())
			assert.Equal("$id=2", Lists[path].Tasks[5].Norm())
			assert.Equal("$P=2", Lists[path].Tasks[6].Norm())
			assert.Equal("$id=4", Lists[path].Tasks[7].Norm())
			assert.Equal("$id=5", Lists[path].Tasks[8].Norm())
			assert.Equal("$P=-2", Lists[path].Tasks[9].Norm())
			assert.Equal("$P=-3", Lists[path].Tasks[10].Norm())
			assert.Equal("$P=-4", Lists[path].Tasks[11].Norm())
		}
	})
	t.Run("datetime", func(t *testing.T) {
		for _, lineSet := range shuffleCount([]string{
			"$due=1w $dead=1w", "$due=1w $dead=2w",
			"$due=1w $end=1w", "$due=1w $end=2w",
			"$due=2d", "$due=4d",
			"$c=2024-feb", "$c=2024-mar",
			"$r=20d $r=1w $r=1h", "$r=1w $r=1h", "$r=10d $r=1h", "$r=1h $r=2d",
		}, 5) {
			prep(lineSet)
			assert.Equal("$c=2024-02", arr[6].Raw())
			assert.Equal("$c=2024-03", arr[7].Raw())
			assert.Equal("$r=1h $r=2d", arr[8].Norm())
			assert.Equal("$r=2w6d $r=1w $r=1h", arr[9].Norm())
			assert.Equal("$r=1w $r=1h", arr[10].Norm())
			assert.Equal("$r=1w3d $r=1h", arr[11].Norm())
		}
	})
	t.Run("urgent", func(t *testing.T) {
		for _, lineSet := range shuffleCount([]string{
			"$mit=-1", "$mit=0", "$mit=1",
			"$due=2m $end=1w", "$due=2m $end=2w",
			"$due=2m $dead=1w", "$due=2m $dead=2w",
			"$due=2m", "$due=3m",
			"$urgent", "$urgent $due=2m",
			"$due=1w $end=1w", "$due=1w $end=2w",
			"$due=1w $dead=1w", "$due=1w $dead=2w",
			"$due=1w", "$due=2w",
			"$urgent", "not so urgent",
		}, 5) {
			prep(lineSet)
			assert.Equal("$mit=0", arr[0].Norm())
			assert.Equal("$mit=1", arr[1].Norm())
			assert.Equal("$due=1w $end=1w", arr[2].Norm())
			assert.Equal("$due=1w $dead=1w", arr[3].Norm())
			assert.Equal("$due=1w $end=2w", arr[4].Norm())
			assert.Equal("$due=1w $dead=2w", arr[5].Norm())
			assert.Equal("$due=1w", arr[6].Norm())
			assert.Equal("$due=2w", arr[7].Norm())
			assert.Equal("$urgent $due=2m", arr[8].Norm())
			assert.Equal("$urgent", arr[9].Norm())
			assert.Equal("$urgent", arr[10].Norm())
			assert.Equal("$mit=-1", arr[11].Norm())
			assert.Equal("not so urgent", arr[12].Norm())
			assert.Equal("$due=2m $end=1w", arr[13].Norm())
			assert.Equal("$due=2m $end=2w", arr[14].Norm())
			assert.Equal("$due=2m $dead=1w", arr[15].Norm())
			assert.Equal("$due=2m $dead=2w", arr[16].Norm())
			assert.Equal("$due=2m", arr[17].Norm())
			assert.Equal("$due=3m", arr[18].Norm())
		}
	})
	t.Run("every", func(t *testing.T) {
		for _, lineSet := range permutations([]string{
			"$every=6m $due=3w", "$every=1y $due=3w",
		}) {
			prep(lineSet)
			assert.Equal("$every=6m $due=3w", arr[0].Norm())
			assert.Equal("$every=1y $due=3w", arr[1].Norm())
		}
	})
}
