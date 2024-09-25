package voronoi

import (
	"math"
)

type Vertex struct {
	X float64
	Y float64
}

var NO_VERTEX = Vertex{math.Inf(1), math.Inf(1)}

type Vertices []Vertex

func (s Vertices) Len() int      { return len(s) }
func (s Vertices) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type VerticesByY struct{ Vertices }

func (s VerticesByY) Less(i, j int) bool { return s.Vertices[i].Y < s.Vertices[j].Y }

type EdgeVertex struct {
	Vertex
	Edges []*Edge
}

type Edge struct {
	LeftCell  *Cell
	RightCell *Cell
	Va        EdgeVertex
	Vb        EdgeVertex
}

func (e *Edge) GetOtherCell(cell *Cell) *Cell {
	if cell == e.LeftCell {
		return e.RightCell
	} else if cell == e.RightCell {
		return e.LeftCell
	}
	return nil
}

func (e *Edge) GetOtherEdgeVertex(v Vertex) EdgeVertex {
	if v == e.Va.Vertex {
		return e.Vb
	} else if v == e.Vb.Vertex {
		return e.Va
	}
	return EdgeVertex{NO_VERTEX, nil}
}

func newEdge(LeftCell, RightCell *Cell) *Edge {
	return &Edge{
		LeftCell:  LeftCell,
		RightCell: RightCell,
		Va:        EdgeVertex{NO_VERTEX, nil},
		Vb:        EdgeVertex{NO_VERTEX, nil},
	}
}

type Halfedge struct {
	Cell  *Cell
	Edge  *Edge
	Angle float64
}

type Halfedges []*Halfedge

func (s Halfedges) Len() int      { return len(s) }
func (s Halfedges) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type halfedgesByAngle struct{ Halfedges }

func (s halfedgesByAngle) Less(i, j int) bool { return s.Halfedges[i].Angle > s.Halfedges[j].Angle }

func newHalfedge(edge *Edge, LeftCell, RightCell *Cell) *Halfedge {
	ret := &Halfedge{
		Cell: LeftCell,
		Edge: edge,
	}

	if RightCell != nil {
		ret.Angle = math.Atan2(RightCell.Site.Y-LeftCell.Site.Y, RightCell.Site.X-LeftCell.Site.X)
	} else {
		va := edge.Va
		vb := edge.Vb

		if edge.LeftCell == LeftCell {
			ret.Angle = math.Atan2(vb.X-va.X, va.Y-vb.Y)
		} else {
			ret.Angle = math.Atan2(va.X-vb.X, vb.Y-va.Y)
		}
	}
	return ret
}

func (h *Halfedge) GetStartpoint() Vertex {
	if h.Edge.LeftCell == h.Cell {
		return h.Edge.Va.Vertex
	}
	return h.Edge.Vb.Vertex

}

func (h *Halfedge) GetEndpoint() Vertex {
	if h.Edge.LeftCell == h.Cell {
		return h.Edge.Vb.Vertex
	}
	return h.Edge.Va.Vertex
}
