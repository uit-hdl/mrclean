package main

import (
	"fmt"

	geom "github.com/folago/googlmath"
	"github.com/folago/mrclean"
)

func main() {}

//
type Display struct {
	Name string
	geom.Rectangle
	Visuals []mrclean.Visual
}

// The Display methods signatures follow the RPC rules
// See http://golang.org/pkg/net/rpc/
func (d *Display) AddVisual(vis mrclean.Visual, reply *int) error {
	return fmt.Errorf("not implemented")
}
func (d *Display) RemoveVisualByID(vid int, reply *int) error {
	return fmt.Errorf("not implemented")
}
func (d *Display) RemoveVisualByName(vname string, reply *int) error {
	return fmt.Errorf("not implemented")
}
func (d *Display) MoveVisual(vid int, reply *int) error {
	return fmt.Errorf("not implemented")
}

func (d *Display) SetVisualRect(vis mrclean.Visual, reply *int) error {
	return fmt.Errorf("not implemented")
}
