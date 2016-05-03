package server

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

// Test adding packages
func TestAdd(t *testing.T) {
	m := NewMapStore()
	p := NewPackageIndexer(1, m, 8080)

	tests := []struct {
		name         string
		dependencies []string
		success      bool
	}{
		{
			name:         "a",
			dependencies: []string{"b", "c"},
			success:      false,
		},
		{
			name:         "b",
			dependencies: []string{},
			success:      true,
		},
		{
			name:         "c",
			dependencies: []string{},
			success:      true,
		},

		{
			name:         "d",
			dependencies: []string{"b", "c"},
			success:      true,
		},
	}

	for _, test := range tests {
		newPkg := &Package{name: test.name, dependents: make(map[string]interface{}), dependencies: test.dependencies}
		success := p.Add(newPkg)
		pkg := m.m[test.name]
		if success != test.success {
			t.Errorf("expected %t, got %t", test.success, success)

		}
		if !success {
			fmt.Println("returning?")
			return
		}
		if !reflect.DeepEqual(pkg, newPkg) {
			t.Errorf("expected %v, got %v\n", newPkg, pkg)
		}
	}
}

// Test querying packages
func TestQuery(t *testing.T) {
	m := NewMapStore()
	p := NewPackageIndexer(1, m, 8080)

	p.Add(&Package{name: "b", dependencies: make([]string, 0), dependents: make(map[string]interface{})})
	tests := []struct {
		request  string
		expected bool
	}{
		{
			request:  "z",
			expected: false,
		},
		{
			request:  "b",
			expected: true,
		},
		{
			request:  "",
			expected: false,
		},
	}

	for _, test := range tests {
		result := p.Query(test.request)
		if result != test.expected {
			t.Errorf("expected %t, got %t", test.expected, result)
		}

	}
}

// Test removing packages
func TestRemove(t *testing.T) {
	m := NewMapStore()
	p := NewPackageIndexer(1, m, 8080)

	p.Add(&Package{name: "b", dependencies: []string{}, dependents: make(map[string]interface{})})
	p.Add(&Package{name: "c", dependencies: []string{"b"}, dependents: make(map[string]interface{})})
	tests := []struct {
		request  string
		expected bool
	}{
		{
			request:  "z",
			expected: true,
		},
		{
			request:  "b",
			expected: false,
		},
		{
			request:  "c",
			expected: true,
		},
		{
			request:  "b",
			expected: true,
		},
	}

	for _, test := range tests {
		result := p.Remove(test.request)
		if result != test.expected {
			t.Errorf("expected %t, got %t", test.expected, result)
		}

	}
}

// Testing concurrent add, making sure that no dataraces occur
func TestConcurrentAdd(t *testing.T) {
	m := NewMapStore()
	p := NewPackageIndexer(1, m, 8080)

	a := &Package{
		name:         "a",
		dependencies: []string{},
		dependents:   map[string]interface{}{},
	}
	b := &Package{
		name:         "b",
		dependencies: []string{},
		dependents:   map[string]interface{}{},
	}
	c := &Package{
		name:         "c",
		dependencies: []string{},
		dependents:   map[string]interface{}{},
	}
	d := &Package{
		name:         "d",
		dependencies: []string{},
		dependents:   map[string]interface{}{},
	}
	p.Add(a)
	p.Add(b)
	p.Add(c)
	p.Add(d)

	e1 := &Package{
		name:         "e",
		dependencies: []string{"a", "b"},
		dependents:   map[string]interface{}{},
	}
	e2 := &Package{
		name:         "e",
		dependencies: []string{"b", "c"},
		dependents:   map[string]interface{}{},
	}
	e3 := &Package{
		name:         "e",
		dependencies: []string{"c", "d"},
		dependents:   map[string]interface{}{},
	}
	wg := sync.WaitGroup{}

	for _, pkg := range []*Package{e1, e2, e3} {
		wg.Add(1)
		go func(e *Package) {
			p.Add(e)
			wg.Done()
		}(pkg)
	}

	wg.Wait()

	_, success := p.store.Get("e")

	if !success {
		t.Error("can't find e package")
	}
}

func addWithWaitGroup(p *PackageIndexer, pkg *Package, wg *sync.WaitGroup) {
	p.Add(pkg)
	wg.Done()
}

// Testing concurrent query and remove, making sure that no dataraces occur
func TestConcurrentQueryRemove(t *testing.T) {
	m := NewMapStore()
	p := NewPackageIndexer(1, m, 8080)

	a := &Package{
		name:         "a",
		dependencies: []string{},
		dependents:   map[string]interface{}{},
	}
	b := &Package{
		name:         "b",
		dependencies: []string{},
		dependents:   map[string]interface{}{},
	}
	c := &Package{
		name:         "c",
		dependencies: []string{},
		dependents:   map[string]interface{}{},
	}
	d := &Package{
		name:         "d",
		dependencies: []string{},
		dependents:   map[string]interface{}{},
	}
	wg := sync.WaitGroup{}

	p.Add(a)
	p.Add(b)
	p.Add(c)
	p.Add(d)

	wg.Add(1)
	go func() {
		p.Remove("a")
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		p.Query("a")
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		p.Remove("b")
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		p.Query("b")
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		p.Remove("c")
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		p.Query("c")
		wg.Done()
	}()

	wg.Wait()
}

// Testing the request parsing
func TestParseRequestString(t *testing.T) {
	tests := []struct {
		request  string
		expected *Request
		success  bool
	}{
		{
			request:  "INDEX|a|b,c",
			expected: &Request{command: CmdIndex, pkg: "a", dependencies: []string{"b", "c"}},
			success:  true,
		},
		{
			request:  "INDEX|a|b",
			expected: &Request{command: CmdIndex, pkg: "a", dependencies: []string{"b"}},
			success:  true,
		},

		{
			request:  "INDEX|a|",
			expected: &Request{command: CmdIndex, pkg: "a", dependencies: []string{}},
			success:  true,
		},
		{
			request:  "QUERY|a|",
			expected: &Request{command: CmdQuery, pkg: "a", dependencies: []string{}},
			success:  true,
		},
		{
			request:  "REMOVE|a|",
			expected: &Request{command: CmdRemove, pkg: "a", dependencies: []string{}},
			success:  true,
		},
		{
			request:  "BADCOMMAND1",
			expected: nil,
			success:  false,
		},
		{
			request:  "INDEX|a",
			expected: nil,
			success:  false,
		},
		{
			request:  "坏子",
			expected: nil,
			success:  false,
		},
	}

	for _, test := range tests {
		result, success := parseRequestString(test.request)

		if success == test.success && !reflect.DeepEqual(result, test.expected) {
			t.Errorf("expected %v, got %v", test.expected, result)
		}
	}
}

// Testing the mapKeys util function
func TestMapKeys(t *testing.T) {
	tests := []struct {
		m        map[string]interface{}
		expected []string
	}{
		{
			m:        map[string]interface{}{"a": nil, "b": nil},
			expected: []string{"a", "b"},
		},
		{
			m:        map[string]interface{}{"a": nil},
			expected: []string{"a"},
		},
		{
			m:        map[string]interface{}{},
			expected: []string{},
		},
	}

	for _, test := range tests {
		if !testEq(mapKeys(test.m), test.expected, t) {
			t.Errorf("expected %v, got %v", test.expected, mapKeys(test.m))
		}
	}
}

func testEq(a, b []string, t *testing.T) bool {

	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		t.Log("a or b are nill")
		return false
	}

	var found bool
	if len(a) != len(b) {

		t.Log("a and b are not the same length")
		return false
	}

	for _, v1 := range a {
		for _, v2 := range b {
			if v1 == v2 {
				found = true
			}
		}
		if !found {
			return false
		}
		found = false
	}

	return true
}

// Testing the sliceToMap util function
func TestSliceToMap(t *testing.T) {
	tests := []struct {
		expected map[string]interface{}
		s        []string
	}{
		{
			expected: map[string]interface{}{"a": struct{}{}, "b": struct{}{}},
			s:        []string{"a", "b"},
		},
		{
			expected: map[string]interface{}{"a": struct{}{}},
			s:        []string{"a"},
		},
		{
			expected: map[string]interface{}{},
			s:        []string{},
		},
	}

	for _, test := range tests {
		result := sliceToMap(test.s)
		if !reflect.DeepEqual(result, test.expected) {
			t.Errorf("expected %v, got %v", test.expected, result)
		}
	}
}
