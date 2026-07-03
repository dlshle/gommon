package uri_trie

import (
	"fmt"
	"sync"
	"testing"

	test_utils "github.com/dlshle/gommon/testutils"
)

func TestTrieTree(t *testing.T) {
	tree := NewTrieTree()
	test_utils.NewTestGroup("trie tree", "").Cases([]*test_utils.Assertion{
		test_utils.NewTestCase("Add wildcard", "", func() bool {
			err := tree.Add("/x/*z", true, true)
			if err != nil {
				return false
			}
			return true
		}),
		test_utils.NewTestCase("Match wildcard", "", func() bool {
			return tree.SupportsUri("/x/asd")
		}),
		test_utils.NewTestCase("Add const over wildcard", "", func() bool {
			err := tree.Add("/x/z", true, true)
			if err != nil {
				return false
			}
			return true
		}),
		test_utils.NewTestCase("Add param over wildcard", "", func() bool {
			err := tree.Add("/x/:z", true, true)
			if err != nil {
				return true
			}
			return false
		}),
		test_utils.NewTestCase("Add wildcard over wildcard", "", func() bool {
			err := tree.Add("/x/*aaa", true, true)
			if err != nil {
				return true
			}
			return false
		}),
		test_utils.NewTestCase("match const and then match wildcard", "", func() bool {
			ctx, err := tree.Match("/x/z")
			if err != nil {
				return false
			}
			if !ctx.Value.(bool) {
				return false
			}
			ctx, err = tree.Match("/x/asd")
			return ctx.PathParams["z"] == "asd"
		}),
		test_utils.NewTestCase("clear", "", func() bool {
			tree.RemoveAll()
			return !tree.SupportsUri("/x/asd")
		}),
		test_utils.NewTestCase("Add short wildcard", "", func() bool {
			err := tree.Add("/*z", true, true)
			if err != nil {
				return false
			}
			return true
		}),
		test_utils.NewTestCase("Match short wildcard", "", func() bool {
			ctx, err := tree.Match("/xyz")
			if err != nil {
				return false
			}
			return ctx.PathParams["z"] == "xyz"
		}),
		test_utils.NewTestCase("Add const", "", func() bool {
			tree.RemoveAll()
			err := tree.Add("/x/y/z", true, true)
			if err != nil {
				return false
			}
			return true
		}),
		test_utils.NewTestCase("Test const", "", func() bool {
			return tree.SupportsUri("/x/y/z")
		}),
		test_utils.NewTestCase("Add param", "", func() bool {
			err := tree.Add("/x/y/z/:p/end", true, true)
			if err != nil {
				return false
			}
			return true
		}),
		test_utils.NewTestCase("Match param", "", func() bool {
			ctx, err := tree.Match("/x/y/z/param/end")
			if err != nil {
				return false
			}
			return ctx.PathParams["p"] == "param"
		}),
		test_utils.NewTestCase("Add double param", "", func() bool {
			err := tree.Add("/x/y/z/:p/end/:pp", true, true)
			if err != nil {
				return false
			}
			return true
		}),
		test_utils.NewTestCase("Match param", "", func() bool {
			ctx, err := tree.Match("/x/y/z/param0/end/param1")
			if err != nil {
				return false
			}
			return ctx.PathParams["p"] == "param0" && ctx.PathParams["pp"] == "param1"
		}),
		test_utils.NewTestCase("Add wildcard over const", "", func() bool {
			return tree.Add("/x/*stuff", true, true) == nil
		}),
		test_utils.NewTestCase("test match again", "", func() bool {
			res := tree.SupportsUri("/x/qwe")
			if !res {
				t.Log("do not support /x/qwe!")
				return false
			}
			res = tree.SupportsUri("/x/y/z")
			if !res {
				t.Log("do not support /x/y/z")
				return false
			}
			ctx, err := tree.Match("/x/y/z/param0/end/param1")
			if err != nil {
				t.Log("do not match previous pattern!")
				return false
			}
			return ctx.PathParams["p"] == "param0" && ctx.PathParams["pp"] == "param1"
		}),
		test_utils.NewTestCase("/x/:y, and then /x should not return err", "", func() bool {
			tree.RemoveAll()
			err := tree.Add("/x/:y", true, true)
			if err != nil {
				return false
			}
			err = tree.Add("/x", true, true)
			if err != nil {
				return false
			}
			return true
		}),
		test_utils.NewTestCase("/x/:y, and then /x/:y/z", "", func() bool {
			tree.RemoveAll()
			tree.Add("/x/:y", true, true)
			err := tree.Add("/x/:y/z", true, true)
			if err != nil {
				return false
			}
			if !tree.SupportsUri("/x/1") {
				return false
			}
			if !tree.SupportsUri("/x/5qwe/z") {
				return false
			}
			return !tree.SupportsUri("/x")
		}),
		test_utils.NewTestCase("/x/:y/z, and then /x/:y", "", func() bool {
			tree.RemoveAll()
			tree.Add("/x/:y/z", true, true)
			err := tree.Add("/x/:y", true, true)
			if err != nil {
				return false
			}
			if !tree.SupportsUri("/x/1") {
				return false
			}
			if !tree.SupportsUri("/x/5qwe/z") {
				return false
			}
			return !tree.SupportsUri("/x")
		}),
		test_utils.NewTestCase("/x/:y, and then /x/z", "", func() bool {
			tree.RemoveAll()
			tree.Add("/x/:y", true, true)
			err := tree.Add("/x/z", true, true)
			if err != nil {
				return false
			}
			return true
		}),
		test_utils.NewTestCase("/projects?pageSize=10&pageToken=123==xyz=", "", func() bool {
			tree.RemoveAll()
			err := tree.Add("/projects", true, true)
			if err != nil {
				return false
			}
			ctx, err := tree.Match("/projects?pageSize=10&pageToken=123==xyz=")
			if err != nil {
				return false
			}
			return ctx.QueryParams["pageSize"] == "10" && ctx.QueryParams["pageToken"] == "123==xyz="
		}),
	}).Do(t)
}

func TestTrieTreeFixes(t *testing.T) {
	tree := NewTrieTree()
	test_utils.NewTestGroup("trie tree fixes", "").Cases([]*test_utils.Assertion{
		test_utils.NewTestCase("Add root path", "", func() bool {
			return tree.Add("/", "root", true) == nil && tree.SupportsUri("/")
		}),
		test_utils.NewTestCase("Match root path", "", func() bool {
			ctx, err := tree.Match("/")
			if err != nil {
				return false
			}
			return ctx.Value.(string) == "root"
		}),
		test_utils.NewTestCase("Root path size", "", func() bool {
			return tree.Size() == 1
		}),
		test_utils.NewTestCase("Add root path without override fails", "", func() bool {
			return tree.Add("/", "root2", false) != nil
		}),
		test_utils.NewTestCase("Add and remove const path", "", func() bool {
			tree.RemoveAll()
			if tree.Add("/a/b", true, true) != nil {
				return false
			}
			if !tree.SupportsUri("/a/b") {
				return false
			}
			if !tree.Remove("/a/b") {
				return false
			}
			return !tree.SupportsUri("/a/b") && tree.Size() == 0
		}),
		test_utils.NewTestCase("Remove intermediate keeps children", "", func() bool {
			tree.RemoveAll()
			tree.Add("/a/b", true, true)
			tree.Add("/a/b/c", true, true)
			if !tree.Remove("/a/b") {
				return false
			}
			if tree.SupportsUri("/a/b") {
				return false
			}
			return tree.SupportsUri("/a/b/c") && tree.Size() == 1
		}),
		test_utils.NewTestCase("Size does not increment on failed add", "", func() bool {
			tree.RemoveAll()
			tree.Add("/x", true, true)
			before := tree.Size()
			err := tree.Add("/x", false, false)
			return err != nil && tree.Size() == before
		}),
		test_utils.NewTestCase("Size does not increment on override", "", func() bool {
			tree.RemoveAll()
			tree.Add("/x", true, true)
			before := tree.Size()
			err := tree.Add("/x", false, true)
			return err == nil && tree.Size() == before
		}),
		test_utils.NewTestCase("Query param empty value", "", func() bool {
			tree.RemoveAll()
			tree.Add("/projects", true, true)
			ctx, err := tree.Match("/projects?k=")
			if err != nil {
				return false
			}
			return ctx.QueryParams["k"] == ""
		}),
		test_utils.NewTestCase("Query param empty segment", "", func() bool {
			ctx, err := tree.Match("/projects?a=1&&b=2")
			if err != nil {
				return false
			}
			return ctx.QueryParams["a"] == "1" && ctx.QueryParams["b"] == "2"
		}),
		test_utils.NewTestCase("Duplicate param name rejected", "", func() bool {
			tree.RemoveAll()
			return tree.Add("/x/:a/:a", true, true) != nil
		}),
		test_utils.NewTestCase("Concurrent add and match", "", func() bool {
			tree.RemoveAll()
			tree.Add("/concurrent/:id", true, true)
			var wg sync.WaitGroup
			ok := true
			var mu sync.Mutex
			for i := 0; i < 50; i++ {
				wg.Add(2)
				go func(i int) {
					defer wg.Done()
					err := tree.Add(fmt.Sprintf("/c/%d", i), true, true)
					if err != nil {
						mu.Lock()
						ok = false
						mu.Unlock()
					}
				}(i)
				go func(i int) {
					defer wg.Done()
					_, err := tree.Match(fmt.Sprintf("/concurrent/%d", i))
					if err != nil {
						mu.Lock()
						ok = false
						mu.Unlock()
					}
				}(i)
			}
			wg.Wait()
			return ok
		}),
	}).Do(t)
}
