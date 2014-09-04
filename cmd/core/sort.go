package main

import (
	"sort"

	"github.com/UniversityofTromso/mrclean"
)

//coypasta from stdlib

// By is the type of a "less" function that defines the ordering of its mrclean.Visual arguments.
type By func(p1, p2 *mrclean.Visual) bool

// Sort is a method on the function type, By, that sorts the argument slice according to the function.
func (by By) Sort(visuals []mrclean.Visual) {
	ps := &visualSorter{
		visuals: visuals,
		by:      by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(ps)
}

// visualSorter joins a By function and a slice of Visuals to be sorted.
type visualSorter struct {
	visuals []mrclean.Visual
	by      func(p1, p2 *mrclean.Visual) bool // Closure used in the Less method.
}

// Len is part of sort.Interface.
func (s *visualSorter) Len() int {
	return len(s.visuals)
}

// Swap is part of sort.Interface.
func (s *visualSorter) Swap(i, j int) {
	s.visuals[i], s.visuals[j] = s.visuals[j], s.visuals[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (s *visualSorter) Less(i, j int) bool {
	return s.by(&s.visuals[i], &s.visuals[j])
}

//var visuals = []mrclean.Visual{
//	{"Mercury", 0.055, 0.4},
//	{"Venus", 0.815, 0.7},
//	{"Earth", 1.0, 1.0},
//	{"Mars", 0.107, 1.5},
//}

// SortKeys demonstrates a technique for sorting a struct type using programmable sort criteria.
// Closures that order the mrclean.Visual structure.
var (
	namef = func(v1, v2 *mrclean.Visual) bool {
		return v1.Name < v2.Name
	}
	idf = func(v1, v2 *mrclean.Visual) bool {
		return v1.ID < v2.ID
	}
	urlf = func(v1, v2 *mrclean.Visual) bool {
		return v1.URL < v2.URL
	}
	metaf = func(v1, v2 *mrclean.Visual) bool {
		return v1.Meta < v2.Meta
	}
)

// Sort the visuals by the various criteria.
//By(name).Sort(visuals)
//fmt.Println("By name:", visuals)

//By(mass).Sort(visuals)
//fmt.Println("By mass:", visuals)

//By(distance).Sort(visuals)
//fmt.Println("By distance:", visuals)

//By(decreasingDistance).Sort(visuals)
//fmt.Println("By decreasing distance:", visuals)
