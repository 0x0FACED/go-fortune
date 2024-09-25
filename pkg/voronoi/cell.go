package voronoi

import "sort"

type Cell struct {
	Site      Vertex
	Halfedges []*Halfedge
}

func newCell(site Vertex) *Cell {
	return &Cell{Site: site}
}

func (t *Cell) prepare() int {
	halfedges := t.Halfedges
	iHalfedge := len(halfedges) - 1

	for ; iHalfedge >= 0; iHalfedge-- {
		edge := halfedges[iHalfedge].Edge

		if edge.Vb.Vertex == NO_VERTEX || edge.Va.Vertex == NO_VERTEX {
			halfedges[iHalfedge] = halfedges[len(halfedges)-1]
			halfedges = halfedges[:len(halfedges)-1]
		}
	}

	sort.Sort(halfedgesByAngle{halfedges})
	t.Halfedges = halfedges
	return len(halfedges)
}
