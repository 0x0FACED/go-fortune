package voronoi

import "sort"

type cell struct {
	site      Vertex
	halfEdges []*halfEdge
}

func newCell(site Vertex) *cell {
	return &cell{site: site}
}

func (t *cell) prepare() int {
	halfedges := t.halfEdges
	iHalfedge := len(halfedges) - 1

	for ; iHalfedge >= 0; iHalfedge-- {
		edge := halfedges[iHalfedge].Edge

		if edge.Vb.Vertex == NO_VERTEX || edge.Va.Vertex == NO_VERTEX {
			halfedges[iHalfedge] = halfedges[len(halfedges)-1]
			halfedges = halfedges[:len(halfedges)-1]
		}
	}

	sort.Sort(halfEdgesByAngle{halfedges})
	t.halfEdges = halfedges
	return len(halfedges)
}
