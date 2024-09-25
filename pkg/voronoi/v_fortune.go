package voronoi

import (
	"fmt"
	"math"
	"sort"

	"github.com/0x0FACED/go-fortune/pkg/logger"
	"go.uber.org/zap"
)

// Основная структура
type Voronoi struct {
	// ячейки диаграммы Вороного
	cells []*Cell
	// ребра диаграммы Вороного
	edges []*Edge

	// мапа для быстрого доступа к ячейке по координатам (ключу)
	cellsMap map[Vertex]*Cell

	// Пляжная линия (красно-черное дерево)
	// динамические меняется при продвижении, охватывает всю высоту от 0 до H
	beachline rbt
	// События круга (для отслеживания, когда пляжная линия исчезнет)
	circleEvents rbt
	// следующее событие круга
	firstCircleEvent *circleEvent

	Logger *logger.ZapLogger
}

// Структура диаграммы
type Diagram struct {
	Cells []*Cell
	Edges []*Edge
}

func (s *Voronoi) cell(site Vertex) *Cell {
	ret := s.cellsMap[site]
	if ret == nil {
		panic(fmt.Sprintf("Couldn't find cell for site %v", site))
	}
	return ret
}

// Создание ребра
func (s *Voronoi) createEdge(LeftCell, RightCell *Cell, va, vb Vertex) *Edge {
	edge := newEdge(LeftCell, RightCell)
	s.edges = append(s.edges, edge)
	if va != NO_VERTEX {
		s.setEdgeStartpoint(edge, LeftCell, RightCell, va)
	}

	if vb != NO_VERTEX {
		s.setEdgeEndpoint(edge, LeftCell, RightCell, vb)
	}

	lCell := LeftCell
	rCell := RightCell

	lCell.Halfedges = append(lCell.Halfedges, newHalfedge(edge, LeftCell, RightCell))
	rCell.Halfedges = append(rCell.Halfedges, newHalfedge(edge, RightCell, LeftCell))
	return edge
}

func (s *Voronoi) createBorderEdge(LeftCell *Cell, va, vb Vertex) *Edge {
	edge := newEdge(LeftCell, nil)
	edge.Va.Vertex = va
	edge.Vb.Vertex = vb

	s.edges = append(s.edges, edge)
	return edge
}

func (s *Voronoi) setEdgeStartpoint(edge *Edge, LeftCell, RightCell *Cell, vertex Vertex) {
	if edge.Va.Vertex == NO_VERTEX && edge.Vb.Vertex == NO_VERTEX {
		edge.Va.Vertex = vertex
		edge.LeftCell = LeftCell
		edge.RightCell = RightCell
	} else if edge.LeftCell == RightCell {
		edge.Vb.Vertex = vertex
	} else {
		edge.Va.Vertex = vertex
	}
}

func (s *Voronoi) setEdgeEndpoint(edge *Edge, LeftCell, RightCell *Cell, vertex Vertex) {
	s.setEdgeStartpoint(edge, RightCell, LeftCell, vertex)
}

type BeachSection struct {
	node        *rbtNode
	site        Vertex
	circleEvent *circleEvent
	edge        *Edge
}

func (s *BeachSection) bindToNode(node *rbtNode) {
	s.node = node
}

func (s *BeachSection) Node() *rbtNode {
	return s.node
}

func leftBreakPoint(arc *BeachSection, directrix float64) float64 {
	site := arc.site
	rfocx := site.X
	rfocy := site.Y
	pby2 := rfocy - directrix
	if pby2 == 0 {
		return rfocx
	}

	lArc := arc.Node().previous
	if lArc == nil {
		return math.Inf(-1)
	}
	site = lArc.value.(*BeachSection).site
	lfocx := site.X
	lfocy := site.Y
	plby2 := lfocy - directrix
	if plby2 == 0 {
		return lfocx
	}
	hl := lfocx - rfocx
	aby2 := 1/pby2 - 1/plby2
	b := hl / plby2
	if aby2 != 0 {
		return (-b+math.Sqrt(b*b-2*aby2*(hl*hl/(-2*plby2)-lfocy+plby2/2+rfocy-pby2/2)))/aby2 + rfocx
	}
	return (rfocx + lfocx) / 2
}

func rightBreakPoint(arc *BeachSection, directrix float64) float64 {
	rArc := arc.Node().next
	if rArc != nil {
		return leftBreakPoint(rArc.value.(*BeachSection), directrix)
	}
	site := arc.site
	if site.Y == directrix {
		return site.X
	}
	return math.Inf(1)
}

func (s *Voronoi) detachBeachSection(arc *BeachSection) {
	s.detachCircleEvent(arc)
	s.beachline.removeNode(arc.node)
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

func (s *Voronoi) removeBeachSection(bs *BeachSection) {
	circle := bs.circleEvent
	x := circle.x
	y := circle.ycenter
	vertex := Vertex{x, y}
	previous := bs.node.previous
	next := bs.node.next
	disappearingTransitions := BeachSectionPtrs{bs}
	abs_fn := math.Abs

	s.detachBeachSection(bs)

	lArc := previous.value.(*BeachSection)
	for lArc.circleEvent != nil &&
		abs_fn(x-lArc.circleEvent.x) < 1e-9 &&
		abs_fn(y-lArc.circleEvent.ycenter) < 1e-9 {

		previous = lArc.node.previous
		disappearingTransitions.appendLeft(lArc)
		s.detachBeachSection(lArc)
		lArc = previous.value.(*BeachSection)
	}

	disappearingTransitions.appendLeft(lArc)
	s.detachCircleEvent(lArc)

	var rArc = next.value.(*BeachSection)
	for rArc.circleEvent != nil &&
		abs_fn(x-rArc.circleEvent.x) < 1e-9 &&
		abs_fn(y-rArc.circleEvent.ycenter) < 1e-9 {
		next = rArc.node.next
		disappearingTransitions.appendRight(rArc)
		s.detachBeachSection(rArc) // mark for reuse
		rArc = next.value.(*BeachSection)
	}

	disappearingTransitions.appendRight(rArc)
	s.detachCircleEvent(rArc)

	nArcs := len(disappearingTransitions)

	for iArc := 1; iArc < nArcs; iArc++ {
		rArc = disappearingTransitions[iArc]
		lArc = disappearingTransitions[iArc-1]

		lSite := s.cell(lArc.site)
		rSite := s.cell(rArc.site)

		s.setEdgeStartpoint(rArc.edge, lSite, rSite, vertex)
	}

	lArc = disappearingTransitions[0]
	rArc = disappearingTransitions[nArcs-1]
	lSite := s.cell(lArc.site)
	rSite := s.cell(rArc.site)

	rArc.edge = s.createEdge(lSite, rSite, NO_VERTEX, vertex)

	s.attachCircleEvent(lArc)
	s.attachCircleEvent(rArc)
}

func (v *Voronoi) addBeachsection(site Vertex) {
	v.Logger.Info("[f-add-bs] Входные параметры", zap.Any("site", site))
	// позиция по X
	x := site.X
	// линия текущей позиции прямого сканирования
	directrix := site.Y

	// В целом, парабола - это кривая, которая описывает все точки, находящиеся на ОДИНАКОВОМ
	// расстоянии от site (точки) и ПРЯМОЙ СКАНИРОВАНИЯ (directrix)

	// Параболы сайтов ОТРАЖАЮТ ВЛИЯНИЕ ЭТИХ САЙТОВ на ближайшие области
	// Чем дальше сайт от directrix, тем ШИРЕ парабола, ибо сайт меньше влияет на области
	// Когда параболы пересекаются, строится прямая (граница двух областей)
	// Когда 3 параболы пересекаются, тогда границы смыкаются и ставится точка.

	// lNode и rNode - узлы rbt текущего сайта, которые хранят ссылки на левые и правые дуги
	var lNode, rNode *rbtNode
	// расстояния между новым сайтом и точками пересечения парабол пляжной линии
	var dxl, dxr float64
	node := v.beachline.root

	v.Logger.Info("[f-add-bs] Текущая нода", zap.Any("node", node))
	// пока нода не равна nil
	for node != nil {
		// ищем, какие дуги находятся слева от линии, а какие справа
		nodeBeachline := node.value.(*BeachSection)
		v.Logger.Info("[f-add-bs-for] Точка из ноды", zap.Any("site", nodeBeachline.site))
		dxl = leftBreakPoint(nodeBeachline, directrix) - x
		v.Logger.Info("[f-add-bs-for] Левая break point", zap.Float64("dxl", dxl))
		if dxl > 1e-9 {
			node = node.left
		} else {
			dxr = x - rightBreakPoint(nodeBeachline, directrix)
			if dxr > 1e-9 {
				if node.right == nil {
					lNode = node
					break
				}
				node = node.right
			} else {
				if dxl > -1e-9 {
					lNode = node.previous
					rNode = node
				} else if dxr > -1e-9 {
					lNode = node
					rNode = node.next
				} else {
					lNode = node
					rNode = node
				}
				break
			}
		}
	}

	var lArc, rArc *BeachSection

	if lNode != nil {
		lArc = lNode.value.(*BeachSection)
	}
	if rNode != nil {
		rArc = rNode.value.(*BeachSection)
	}

	newArc := &BeachSection{site: site}
	if lArc == nil {
		v.beachline.insertSuccessor(nil, newArc)
	} else {
		v.beachline.insertSuccessor(lArc.node, newArc)
	}

	if lArc == nil && rArc == nil {
		return
	}

	if lArc == rArc {
		v.detachCircleEvent(lArc)

		rArc = &BeachSection{site: lArc.site}
		v.beachline.insertSuccessor(newArc.node, rArc)

		lCell := v.cell(lArc.site)
		newCell := v.cell(newArc.site)
		newArc.edge = v.createEdge(lCell, newCell, NO_VERTEX, NO_VERTEX)
		rArc.edge = newArc.edge

		v.attachCircleEvent(lArc)
		v.attachCircleEvent(rArc)
		return
	}

	if lArc != nil && rArc == nil {
		lCell := v.cell(lArc.site)
		newCell := v.cell(newArc.site)
		newArc.edge = v.createEdge(lCell, newCell, NO_VERTEX, NO_VERTEX)
		return
	}

	if lArc != rArc {
		v.detachCircleEvent(lArc)
		v.detachCircleEvent(rArc)

		LeftSite := lArc.site
		ax := LeftSite.X
		ay := LeftSite.Y
		bx := site.X - ax
		by := site.Y - ay
		RightSite := rArc.site
		cx := RightSite.X - ax
		cy := RightSite.Y - ay
		d := 2 * (bx*cy - by*cx)
		hb := bx*bx + by*by
		hc := cx*cx + cy*cy
		vertex := Vertex{(cy*hb-by*hc)/d + ax, (bx*hc-cx*hb)/d + ay}

		lCell := v.cell(LeftSite)
		cell := v.cell(site)
		rCell := v.cell(RightSite)

		v.setEdgeStartpoint(rArc.edge, lCell, rCell, vertex)

		newArc.edge = v.createEdge(lCell, cell, NO_VERTEX, vertex)
		rArc.edge = v.createEdge(cell, rCell, NO_VERTEX, vertex)

		v.attachCircleEvent(lArc)
		v.attachCircleEvent(rArc)
		return
	}
}

type circleEvent struct {
	node    *rbtNode
	site    Vertex
	arc     *BeachSection
	x       float64
	y       float64
	ycenter float64
}

func (s *circleEvent) bindToNode(node *rbtNode) {
	s.node = node
}

func (s *circleEvent) Node() *rbtNode {
	return s.node
}

func (s *Voronoi) attachCircleEvent(arc *BeachSection) {
	lArc := arc.node.previous
	rArc := arc.node.next
	if lArc == nil || rArc == nil {
		return // does that ever happen?
	}
	LeftSite := lArc.value.(*BeachSection).site
	cSite := arc.site
	RightSite := rArc.value.(*BeachSection).site

	if LeftSite == RightSite {
		return
	}

	bx := cSite.X
	by := cSite.Y
	ax := LeftSite.X - bx
	ay := LeftSite.Y - by
	cx := RightSite.X - bx
	cy := RightSite.Y - by

	d := 2 * (ax*cy - ay*cx)
	if d >= -2e-12 {
		return
	}

	ha := ax*ax + ay*ay
	hc := cx*cx + cy*cy
	x := (cy*ha - ay*hc) / d
	y := (ax*hc - cx*ha) / d
	ycenter := y + by

	circleEventInst := &circleEvent{
		arc:     arc,
		site:    cSite,
		x:       x + bx,
		y:       ycenter + math.Sqrt(x*x+y*y),
		ycenter: ycenter,
	}

	arc.circleEvent = circleEventInst

	var predecessor *rbtNode = nil
	node := s.circleEvents.root
	for node != nil {
		nodeValue := node.value.(*circleEvent)
		if circleEventInst.y < nodeValue.y || (circleEventInst.y == nodeValue.y && circleEventInst.x <= nodeValue.x) {
			if node.left != nil {
				node = node.left
			} else {
				predecessor = node.previous
				break
			}
		} else {
			if node.right != nil {
				node = node.right
			} else {
				predecessor = node
				break
			}
		}
	}
	s.circleEvents.insertSuccessor(predecessor, circleEventInst)
	if predecessor == nil {
		s.firstCircleEvent = circleEventInst
	}
}

func (s *Voronoi) detachCircleEvent(arc *BeachSection) {
	circle := arc.circleEvent
	if circle != nil {
		if circle.node.previous == nil {
			if circle.node.next != nil {
				s.firstCircleEvent = circle.node.next.value.(*circleEvent)
			} else {
				s.firstCircleEvent = nil
			}
		}
		s.circleEvents.removeNode(circle.node)
		arc.circleEvent = nil
	}
}

// Bounding Box
type BoundingBox struct {
	Xl, Xr, Yt, Yb float64
}

// Create new Bounding Box
func NewBoundingBox(xl, xr, yt, yb float64) BoundingBox {
	return BoundingBox{xl, xr, yt, yb}
}

func connectEdge(edge *Edge, bbox BoundingBox) bool {
	vb := edge.Vb.Vertex
	if vb != NO_VERTEX {
		return true
	}

	va := edge.Va.Vertex
	xl := bbox.Xl
	xr := bbox.Xr
	yt := bbox.Yt
	yb := bbox.Yb
	LeftSite := edge.LeftCell.Site
	RightSite := edge.RightCell.Site
	lx := LeftSite.X
	ly := LeftSite.Y
	rx := RightSite.X
	ry := RightSite.Y
	fx := (lx + rx) / 2
	fy := (ly + ry) / 2

	var fm, fb float64

	if !equalWithEpsilon(ry, ly) {
		fm = (lx - rx) / (ry - ly)
		fb = fy - fm*fx
	}

	if equalWithEpsilon(ry, ly) {
		// doesn't intersect with viewport
		if fx < xl || fx >= xr {
			return false
		}
		// downward
		if lx > rx {
			if va == NO_VERTEX {
				va = Vertex{fx, yt}
			} else if va.Y >= yb {
				return false
			}
			vb = Vertex{fx, yb}
			// upward
		} else {
			if va == NO_VERTEX {
				va = Vertex{fx, yb}
			} else if va.Y < yt {
				return false
			}
			vb = Vertex{fx, yt}
		}

	} else if fm < -1 || fm > 1 {
		// downward
		if lx > rx {
			if va == NO_VERTEX {
				va = Vertex{(yt - fb) / fm, yt}
			} else if va.Y >= yb {
				return false
			}
			vb = Vertex{(yb - fb) / fm, yb}
			// upward
		} else {
			if va == NO_VERTEX {
				va = Vertex{(yb - fb) / fm, yb}
			} else if va.Y < yt {
				return false
			}
			vb = Vertex{(yt - fb) / fm, yt}
		}

	} else {
		// rightward
		if ly < ry {
			if va == NO_VERTEX {
				va = Vertex{xl, fm*xl + fb}
			} else if va.X >= xr {
				return false
			}
			vb = Vertex{xr, fm*xr + fb}
			// leftward
		} else {
			if va == NO_VERTEX {
				va = Vertex{xr, fm*xr + fb}
			} else if va.X < xl {
				return false
			}
			vb = Vertex{xl, fm*xl + fb}
		}
	}
	edge.Va.Vertex = va
	edge.Vb.Vertex = vb
	return true
}

func clipEdge(edge *Edge, bbox BoundingBox) bool {
	ax := edge.Va.X
	ay := edge.Va.Y
	bx := edge.Vb.X
	by := edge.Vb.Y
	t0 := float64(0)
	t1 := float64(1)
	dx := bx - ax
	dy := by - ay

	// left
	q := ax - bbox.Xl
	if dx == 0 && q < 0 {
		return false
	}
	r := -q / dx
	if dx < 0 {
		if r < t0 {
			return false
		} else if r < t1 {
			t1 = r
		}
	} else if dx > 0 {
		if r > t1 {
			return false
		} else if r > t0 {
			t0 = r
		}
	}
	// right
	q = bbox.Xr - ax
	if dx == 0 && q < 0 {
		return false
	}
	r = q / dx
	if dx < 0 {
		if r > t1 {
			return false
		} else if r > t0 {
			t0 = r
		}
	} else if dx > 0 {
		if r < t0 {
			return false
		} else if r < t1 {
			t1 = r
		}
	}

	// top
	q = ay - bbox.Yt
	if dy == 0 && q < 0 {
		return false
	}
	r = -q / dy
	if dy < 0 {
		if r < t0 {
			return false
		} else if r < t1 {
			t1 = r
		}
	} else if dy > 0 {
		if r > t1 {
			return false
		} else if r > t0 {
			t0 = r
		}
	}
	// bottom
	q = bbox.Yb - ay
	if dy == 0 && q < 0 {
		return false
	}
	r = q / dy
	if dy < 0 {
		if r > t1 {
			return false
		} else if r > t0 {
			t0 = r
		}
	} else if dy > 0 {
		if r < t0 {
			return false
		} else if r < t1 {
			t1 = r
		}
	}

	if t0 > 0 {
		edge.Va.Vertex = Vertex{ax + t0*dx, ay + t0*dy}
	}

	if t1 < 1 {
		edge.Vb.Vertex = Vertex{ax + t1*dx, ay + t1*dy}
	}

	return true
}

func equalWithEpsilon(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func lessThanWithEpsilon(a, b float64) bool {
	return b-a > 1e-9
}

func greaterThanWithEpsilon(a, b float64) bool {
	return a-b > 1e-9
}

func (s *Voronoi) clipEdges(bbox BoundingBox) {
	abs_fn := math.Abs

	for i := len(s.edges) - 1; i >= 0; i-- {
		edge := s.edges[i]

		if !connectEdge(edge, bbox) || !clipEdge(edge, bbox) || (abs_fn(edge.Va.X-edge.Vb.X) < 1e-9 && abs_fn(edge.Va.Y-edge.Vb.Y) < 1e-9) {
			edge.Va.Vertex = NO_VERTEX
			edge.Vb.Vertex = NO_VERTEX
			s.edges[i] = s.edges[len(s.edges)-1]
			s.edges = s.edges[0 : len(s.edges)-1]
		}
	}
}

func (s *Voronoi) closeCells(bbox BoundingBox) {
	xl := bbox.Xl
	xr := bbox.Xr
	yt := bbox.Yt
	yb := bbox.Yb
	cells := s.cells
	abs_fn := math.Abs

	for _, cell := range cells {
		if cell.prepare() == 0 {
			continue
		}

		halfedges := cell.Halfedges
		nHalfedges := len(halfedges)

		iLeft := 0
		for iLeft < nHalfedges {
			iRight := (iLeft + 1) % nHalfedges
			endpoint := halfedges[iLeft].GetEndpoint()
			startpoint := halfedges[iRight].GetStartpoint()
			if abs_fn(endpoint.X-startpoint.X) >= 1e-9 || abs_fn(endpoint.Y-startpoint.Y) >= 1e-9 {
				va := endpoint
				vb := endpoint
				if equalWithEpsilon(endpoint.X, xl) && lessThanWithEpsilon(endpoint.Y, yb) {
					if equalWithEpsilon(startpoint.X, xl) {
						vb = Vertex{xl, startpoint.Y}
					} else {
						vb = Vertex{xl, yb}
					}

					// walk rightward along bottom side
				} else if equalWithEpsilon(endpoint.Y, yb) && lessThanWithEpsilon(endpoint.X, xr) {
					if equalWithEpsilon(startpoint.Y, yb) {
						vb = Vertex{startpoint.X, yb}
					} else {
						vb = Vertex{xr, yb}
					}
					// walk upward along right side
				} else if equalWithEpsilon(endpoint.X, xr) && greaterThanWithEpsilon(endpoint.Y, yt) {
					if equalWithEpsilon(startpoint.X, xr) {
						vb = Vertex{xr, startpoint.Y}
					} else {
						vb = Vertex{xr, yt}
					}
					// walk leftward along top side
				} else if equalWithEpsilon(endpoint.Y, yt) && greaterThanWithEpsilon(endpoint.X, xl) {
					if equalWithEpsilon(startpoint.Y, yt) {
						vb = Vertex{startpoint.X, yt}
					} else {
						vb = Vertex{xl, yt}
					}
				} else {
					//			break
				}

				edge := s.createBorderEdge(cell, va, vb)
				cell.Halfedges = append(cell.Halfedges, nil)
				halfedges = cell.Halfedges
				nHalfedges = len(halfedges)

				copy(halfedges[iLeft+2:], halfedges[iLeft+1:len(halfedges)-1])
				halfedges[iLeft+1] = newHalfedge(edge, cell, nil)

			}
			iLeft++
		}
	}
}

func (s *Voronoi) gatherVertexEdges() {
	vertexEdgeMap := make(map[Vertex][]*Edge)

	for _, edge := range s.edges {
		vertexEdgeMap[edge.Va.Vertex] = append(
			vertexEdgeMap[edge.Va.Vertex], edge)
		vertexEdgeMap[edge.Vb.Vertex] = append(
			vertexEdgeMap[edge.Vb.Vertex], edge)
	}

	for vertex, edgeSlice := range vertexEdgeMap {
		for _, edge := range edgeSlice {
			if vertex == edge.Va.Vertex {
				edge.Va.Edges = edgeSlice
			}
			if vertex == edge.Vb.Vertex {
				edge.Vb.Edges = edgeSlice
			}
		}
	}
}
