package voronoi

import (
	"math"
)

type Vertex struct {
	X float64
	Y float64
}

var NO_VERTEX = Vertex{math.Inf(1), math.Inf(1)}

type vetrices []Vertex

func (s vetrices) Len() int      { return len(s) }
func (s vetrices) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type verticesByY struct{ vetrices }

func (s verticesByY) Less(i, j int) bool { return s.vetrices[i].Y < s.vetrices[j].Y }

type edgeVertex struct {
	Vertex
	Edges []*edge
}

type edge struct {
	LeftCell  *cell
	RightCell *cell
	Va        edgeVertex
	Vb        edgeVertex
}

func newEdge(LeftCell, RightCell *cell) *edge {
	return &edge{
		LeftCell:  LeftCell,
		RightCell: RightCell,
		Va:        edgeVertex{NO_VERTEX, nil},
		Vb:        edgeVertex{NO_VERTEX, nil},
	}
}

type halfEdge struct {
	Cell  *cell
	Edge  *edge
	Angle float64
}

type halfEdges []*halfEdge

func (s halfEdges) Len() int      { return len(s) }
func (s halfEdges) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type halfEdgesByAngle struct{ halfEdges }

func (s halfEdgesByAngle) Less(i, j int) bool { return s.halfEdges[i].Angle > s.halfEdges[j].Angle }

func newHalfEdge(edge *edge, LeftCell, RightCell *cell) *halfEdge {
	ret := &halfEdge{
		Cell: LeftCell,
		Edge: edge,
	}

	if RightCell != nil {
		ret.Angle = math.Atan2(RightCell.site.Y-LeftCell.site.Y, RightCell.site.X-LeftCell.site.X)
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

func (h *halfEdge) startPoint() Vertex {
	if h.Edge.LeftCell == h.Cell {
		return h.Edge.Va.Vertex
	}
	return h.Edge.Vb.Vertex

}

func (h *halfEdge) endPoint() Vertex {
	if h.Edge.LeftCell == h.Cell {
		return h.Edge.Vb.Vertex
	}
	return h.Edge.Va.Vertex
}
