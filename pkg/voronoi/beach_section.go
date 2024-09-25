package voronoi

type BeachSection struct {
	node        *rbtNode
	site        Vertex
	circleEvent *circleEvent
	edge        *edge
}

func (s *BeachSection) bindToNode(node *rbtNode) {
	s.node = node
}

func (s *BeachSection) Node() *rbtNode {
	return s.node
}

type BeachSectionPtrs []*BeachSection

func (s *BeachSectionPtrs) appendLeft(b *BeachSection) {
	*s = append(*s, b)
	for id := len(*s) - 1; id > 0; id-- {
		(*s)[id] = (*s)[id-1]
	}
	(*s)[0] = b
}

func (s *BeachSectionPtrs) appendRight(b *BeachSection) {
	*s = append(*s, b)
}
