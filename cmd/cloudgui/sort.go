package main

import (
	"fmt"
	"math"
	"sort"
)

var Task, Method, Iteration, Approach string = "Task", "Method", "Iteration", "Approach"
var lessMap map[string]lessFunc = map[string]lessFunc{
	Task:      lessTask,
	Method:    lessMethod,
	Iteration: lessIteration,
	Approach:  lessApproach,
}

type Visuals []*Visual

//partial implementationn of sort.Interface the rest is implementes
//in the other types
func (i Visuals) Len() int      { return len(i) }
func (i Visuals) Swap(p, q int) { i[p], i[q] = i[q], i[p] }

//sort by name
type ByName struct{ Visuals }

func (n ByName) Less(i, j int) bool { return n.Visuals[i].Name < n.Visuals[j].Name }

//sort by Iteration
type ByIteration struct{ Visuals }

func (n ByIteration) Less(i, j int) bool { return n.Visuals[i].Iteration < n.Visuals[j].Iteration }

//sort by Approach
type ByApproach struct{ Visuals }

func (n ByApproach) Less(i, j int) bool { return n.Visuals[i].Approach < n.Visuals[j].Approach }

//sort by Method
type ByMethod struct{ Visuals }

type lessFunc func(p1, p2 *Visual) bool

// multiSorter implements the Sort interface, sorting the visuals within.
type multiSorter struct {
	visuals []*Visual
	less    []lessFunc
}

// Sort sorts the argument slice according to the less functions passed to OrderedBy.
func (ms *multiSorter) Sort(visuals []*Visual) {
	ms.visuals = visuals
	sort.Sort(ms)
}

// OrderedBy returns a Sorter that sorts using the less functions, in order.
// Call its Sort method to sort the data.
func OrderedBy(less ...lessFunc) *multiSorter {
	return &multiSorter{
		less: less,
	}
}

// Len is part of sort.Interface.
func (ms *multiSorter) Len() int {
	return len(ms.visuals)
}

// Swap is part of sort.Interface.
func (ms *multiSorter) Swap(i, j int) {
	ms.visuals[i], ms.visuals[j] = ms.visuals[j], ms.visuals[i]
}

// Less is part of sort.Interface. It is implemented by looping along the
// less functions until it finds a comparison that is either Less or
// !Less. Note that it can call the less functions twice per call. We
// could change the functions to return -1, 0, 1 and reduce the
// number of calls for greater efficiency: an exercise for the reader.
//TODO the exercise because i am the reader!
func (ms *multiSorter) Less(i, j int) bool {
	p, q := ms.visuals[i], ms.visuals[j]
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(ms.less)-1; k++ {
		less := ms.less[k]
		switch {
		case less(p, q):
			// p < q, so we have a decision.
			return true
		case less(q, p):
			// p > q, so we have a decision.
			return false
		}
		// p == q; try the next comparison.
	}
	// All comparisons to here said "equal", so just return whatever
	// the final comparison reports.
	//fmt.Println("last RE-sort")
	return ms.less[k](p, q)
}

//Less functions, later we put them in a map to mach them with the
//sort command order
func lessTask(v1, v2 *Visual) bool {
	return v1.Task < v2.Task
}

func lessApproach(v1, v2 *Visual) bool {
	return v1.Approach < v2.Approach
}

func lessIteration(v1, v2 *Visual) bool {
	return v1.Iteration < v2.Iteration
}

func lessMethod(v1, v2 *Visual) bool {
	return v1.Method < v2.Method
}

//this function sorts stuff, we have 4 dimension for sorting it's a mess
//func SortVisuals(imgs map[string]*Visual, disp *Display, order []string) ([]RpcReq, error) {
func SortVisuals(imgs []*Visual, disp *Display, order []string) ([]RpcReq, error) {
	//check the  orrder is about fields that exist
	var fncs []lessFunc
	//just the right order without wrong stuff
	//var ord []string
	for _, o := range order {
		f, ok := lessMap[o]
		if ok {
			fmt.Println("found function for: ", o)
			fncs = append(fncs, f)
			//ord = append(ord, o)
		}
	}
	if len(fncs) == 0 {
		return nil, fmt.Errorf("invalid sorting order %v", order)
	}
	//fmt.Println("sorting functions: ", len(fncs))
	//get the bigger image, it will be reference block
	//for the sorting
	//maxr := Rectangle{}
	offset := Point{}
	vizs := make(Visuals, len(imgs))
	c := 0
	for _, v := range imgs {
		//maxr = maxr.Union(v.rect)
		offset.X = math.Max(v.Size[0], offset.X)
		offset.Y = math.Max(v.Size[1], offset.Y)
		//copy of the visuals to work the sorting magic
		vizs[c] = v
		c++
	}
	//magic sorting here
	OrderedBy(fncs...).Sort(vizs)
	//now we put stuff on the screen in order
	fmt.Println("Display: ", disp.Size, disp.rect)
	var (
		margin float64 = 0.05 //5 cm
		row    float64 = 1
		ret    []RpcReq
	)
	//dx, dy := maxr.Dx()+margin, maxr.Dy()+margin
	dx, dy := offset.X+margin, offset.Y+margin
	//lastpos := Point{X: disp.rect.Min.X, Y: disp.rect.Max.Y - dy}
	lastpos := Point{
		X: disp.rect.Min.X + dx*0.5,
		Y: disp.rect.Max.Y - dy + dy*0.5,
	}
	fmt.Println("laspos ", lastpos)
	for _, v := range vizs {
		v.Origin[0], v.Origin[1] = lastpos.X, lastpos.Y
		//fmt.Println("before: ", v.rect, v.rect.Center())
		//fmt.Println("after: ", v.rect, v.rect.Center())
		ret = append(ret, SetVisualOrigin(v.ID, v.Origin))
		//HERE rpc
		lastpos.X += dx
		if lastpos.X+dx*0.5 > disp.rect.Max.X {
			lastpos.X = disp.rect.Min.X + dx*0.5
			row += 1
			lastpos.Y = disp.rect.Max.Y - row*dy
		}
	}
	return ret, nil
}

/*
for _, v := range vizs {
	fmt.Println("before: ", v.rect, v.rect.Center())
	v.rect.Max.X = v.rect.Max.X - (lastpos.X - v.rect.Min.X)
	v.rect.Max.Y = v.rect.Max.Y - (lastpos.Y - v.rect.Min.Y)
	v.rect.Min = lastpos
	fmt.Println("after: ", v.rect, v.rect.Center())
	ret = append(ret, SetVisualOrigin(v.ID, v.rect.Center().Slice()))
	//HERE rpc
	lastpos.X += dx
	if lastpos.X+dx > disp.rect.Max.X {
		lastpos.X = 0
		row += 1
		lastpos.Y = disp.rect.Max.Y - row*dy
	}
}*/
