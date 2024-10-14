package voronoi

import (
	"fmt"
	"math"

	"github.com/0x0FACED/go-fortune/pkg/logger"
	"go.uber.org/zap"
)

// Основная структура
type Voronoi struct {
	// ячейки диаграммы Вороного
	cells []*cell
	// ребра диаграммы Вороного
	edges []*edge

	// мапа для быстрого доступа к ячейке по координатам (ключу)
	cellsMap map[Vertex]*cell

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
	Cells []*cell
	Edges []*edge
}

func (s *Voronoi) cell(site Vertex) *cell {
	ret := s.cellsMap[site]
	if ret == nil {
		panic(fmt.Sprintf("Couldn't find cell for site %v", site))
	}
	return ret
}

// Создание ребра
func (s *Voronoi) createEdge(LeftCell, RightCell *cell, va, vb Vertex) *edge {
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

	lCell.halfEdges = append(lCell.halfEdges, newHalfEdge(edge, LeftCell, RightCell))
	rCell.halfEdges = append(rCell.halfEdges, newHalfEdge(edge, RightCell, LeftCell))
	return edge
}

func (s *Voronoi) createBorderEdge(LeftCell *cell, va, vb Vertex) *edge {
	edge := newEdge(LeftCell, nil)
	edge.Va.Vertex = va
	edge.Vb.Vertex = vb

	s.edges = append(s.edges, edge)
	return edge
}

func (s *Voronoi) setEdgeStartpoint(edge *edge, LeftCell, RightCell *cell, vertex Vertex) {
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

func (s *Voronoi) setEdgeEndpoint(edge *edge, LeftCell, RightCell *cell, vertex Vertex) {
	s.setEdgeStartpoint(edge, RightCell, LeftCell, vertex)
}

func (v *Voronoi) leftBreakPoint(arc *BeachSection, directrix float64) float64 {
	// получаем сайт по дуге (сайт, с чьей дугой работаем)
	site := arc.site
	rfocx := site.X
	rfocy := site.Y
	pby2 := rfocy - directrix
	v.Logger.Info("\t[f-for-add-bs-for-left-bp] (Расстояние) Правая точка пересечения", zap.Float64("right", pby2))
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
	v.Logger.Info("[f-for-add-bs-for-left-bp] (Расстояние) Левая точка пересечения", zap.Float64("left", plby2))
	if plby2 == 0 {
		return lfocx
	}
	hl := lfocx - rfocx
	aby2 := 1/pby2 - 1/plby2
	b := hl / plby2
	var res float64
	if aby2 != 0 {
		res = (-b+math.Sqrt(b*b-2*aby2*(hl*hl/(-2*plby2)-lfocy+plby2/2+rfocy-pby2/2)))/aby2 + rfocx
		v.Logger.Info("[f-for-add-bs-for-left-bp] Результат", zap.Float64("res", res))
		return res
	}
	res = (rfocx + lfocx) / 2
	v.Logger.Info("[f-for-add-bs-for-left-bp] Результат", zap.Float64("res", res))
	return res
}

func (v *Voronoi) rightBreakPoint(arc *BeachSection, directrix float64) float64 {
	rArc := arc.Node().next
	if rArc != nil {
		v.Logger.Info("[f-for-add-bs-for-right-bp] Правая nil, идем налево")
		return v.leftBreakPoint(rArc.value.(*BeachSection), directrix)
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

func (v *Voronoi) removeBeachSection(bs *BeachSection) {
	v.Logger.Info("[f-for-rm-bs-for] Начало rm bs", zap.Any("site_bs", bs.circleEvent.site))
	circle := bs.circleEvent
	v.Logger.Info("[f-for-rm-bs-for] Текущее событие круга", zap.Float64("site_bs_x", bs.circleEvent.x), zap.Float64("site_bs_y", bs.circleEvent.y))
	x := circle.x
	y := circle.ycenter
	vertex := Vertex{x, y}
	previous := bs.node.previous
	next := bs.node.next
	disappearingTransitions := BeachSectionPtrs{bs}
	abs_fn := math.Abs

	v.detachBeachSection(bs)

	lArc := previous.value.(*BeachSection)
	for lArc.circleEvent != nil &&
		abs_fn(x-lArc.circleEvent.x) < 1e-9 &&
		abs_fn(y-lArc.circleEvent.ycenter) < 1e-9 {

		previous = lArc.node.previous
		disappearingTransitions.appendLeft(lArc)
		v.detachBeachSection(lArc)
		lArc = previous.value.(*BeachSection)
	}

	disappearingTransitions.appendLeft(lArc)
	v.detachCircleEvent(lArc)

	var rArc = next.value.(*BeachSection)
	for rArc.circleEvent != nil &&
		abs_fn(x-rArc.circleEvent.x) < 1e-9 &&
		abs_fn(y-rArc.circleEvent.ycenter) < 1e-9 {
		next = rArc.node.next
		disappearingTransitions.appendRight(rArc)
		v.detachBeachSection(rArc)
		rArc = next.value.(*BeachSection)
	}

	disappearingTransitions.appendRight(rArc)
	v.detachCircleEvent(rArc)

	nArcs := len(disappearingTransitions)

	for iArc := 1; iArc < nArcs; iArc++ {
		rArc = disappearingTransitions[iArc]
		lArc = disappearingTransitions[iArc-1]

		lSite := v.cell(lArc.site)
		rSite := v.cell(rArc.site)

		v.setEdgeStartpoint(rArc.edge, lSite, rSite, vertex)
	}

	lArc = disappearingTransitions[0]
	rArc = disappearingTransitions[nArcs-1]
	lSite := v.cell(lArc.site)
	rSite := v.cell(rArc.site)

	rArc.edge = v.createEdge(lSite, rSite, NO_VERTEX, vertex)

	v.attachCircleEvent(lArc)
	v.attachCircleEvent(rArc)
}

func (v *Voronoi) addBeachSection(site Vertex) {
	v.Logger.Info("[f-for-add-bs] Входные параметры", zap.Any("site", site))
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

	v.Logger.Info("[f-for-add-bs] Текущая нода", zap.Any("node", node))
	// пока нода не равна nil. Это поиск места для новой дуги
	// Цикл перебирает дуги на beach line (ДУГИ ПАРАБОЛ), чтобы найти место для новой дуги
	for node != nil {
		// вычисляем левую точку пересечения параболы
		nodeBeachline := node.value.(*BeachSection)
		// вычисляем разницу между координатой X новой точки
		// и левой точкой пересечения текущей параболы с прямой сканирования.
		dxl = v.leftBreakPoint(nodeBeachline, directrix) - x

		v.Logger.Info("[f-for-add-bs-for] Точка из ноды", zap.Any("site", nodeBeachline.site))
		v.Logger.Info("[f-for-add-bs-for] Левая точка пересечения параболы", zap.Float64("dxl", dxl))

		if dxl > 1e-9 {
			v.Logger.Info("[f-for-add-bs-for] Новая точка находится СЛЕВА от текущей дуги (параболы)",
				zap.Float64("dxl", dxl),
			)
			node = node.left
		} else {
			dxr = x - v.rightBreakPoint(nodeBeachline, directrix)
			if dxr > 1e-9 {
				v.Logger.Info("[f-for-add-bs-for] Новая точка находится СПРАВА от текущей дуги (параболы)",
					zap.Float64("dxr", dxr),
				)
				if node.right == nil {
					lNode = node
					break
				}
				node = node.right
			} else {
				v.Logger.Info("[f-for-add-bs-for] Новая точка находится МЕЖДУ ДУГАМИ",
					zap.Float64("dxr", dxr),
				)
				if dxl > -1e-9 {
					v.Logger.Info("[f-for-add-bs-for] Новая точка совпадает с ЛЕВОЙ границей дуги",
						zap.Float64("dxl", dxl),
					)
					lNode = node.previous
					rNode = node
				} else if dxr > -1e-9 {
					v.Logger.Info("[f-for-add-bs-for] Новая точка совпадает с ПРАВОЙ границей дуги",
						zap.Float64("dxr", dxr),
					)
					lNode = node
					rNode = node.next
				} else {
					v.Logger.Info("[f-for-add-bs-for] Новая точка находится ВНУТРИ текущей дуги",
						zap.Float64("dxl", dxl),
						zap.Float64("dxr", dxr),
					)
					lNode = node
					rNode = node
				}
				break
			}
		}
	}

	v.Logger.Info("[f-add-bs] Позиция для новой дуги найдена")
	var lArc, rArc *BeachSection

	// достаем левую и правую дуги (если имеются)
	if lNode != nil {
		lArc = lNode.value.(*BeachSection)
	}

	if rNode != nil {
		rArc = rNode.value.(*BeachSection)
	}

	// создаем новую дугу (параболу)
	newArc := &BeachSection{site: site}
	if lArc == nil {
		v.beachline.insertSuccessor(nil, newArc)
	} else {
		v.beachline.insertSuccessor(lArc.node, newArc)
	}

	// если обе неопределены, то возвращаемся, ибо наша дуга первая
	if lArc == nil && rArc == nil {
		return
	}

	// если новая дуга делит существующую на 2 (находимся внутри дуги)
	if lArc == rArc {
		// удаляем событие круга, связанное с lArc
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
	lSite := lArc.value.(*BeachSection).site
	cSite := arc.site
	rSite := rArc.value.(*BeachSection).site

	if lSite == rSite {
		return
	}

	bx := cSite.X
	by := cSite.Y
	ax := lSite.X - bx
	ay := lSite.Y - by
	cx := rSite.X - bx
	cy := rSite.Y - by

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

func (v *Voronoi) detachCircleEvent(arc *BeachSection) {
	circle := arc.circleEvent
	if circle != nil {
		if circle.node.previous == nil {
			if circle.node.next != nil {
				v.firstCircleEvent = circle.node.next.value.(*circleEvent)
				v.Logger.Info("[f-for-rm-bs-detach-ce] Первое событие круга", zap.Float64("ce_x", v.firstCircleEvent.x), zap.Float64("ce_y", v.firstCircleEvent.y))
			} else {
				v.firstCircleEvent = nil
			}
		}
		v.circleEvents.removeNode(circle.node)
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

// функция для дополнения всех ребер с bbox (в самом конце, когда еще параболы/дуги остались)
func connectEdge(edge *edge, bbox BoundingBox) bool {
	vb := edge.Vb.Vertex
	if vb != NO_VERTEX {
		return true
	}

	va := edge.Va.Vertex
	xl := bbox.Xl
	xr := bbox.Xr
	yt := bbox.Yt
	yb := bbox.Yb
	lSite := edge.LeftCell.site
	rSite := edge.RightCell.site
	lx := lSite.X
	ly := lSite.Y
	rx := rSite.X
	ry := rSite.Y
	fx := (lx + rx) / 2
	fy := (ly + ry) / 2

	var fm, fb float64

	// определение наклона ребра fm и смещения fb
	if !equalEps(ry, ly) {
		// если ry == ly, значит наклона нет (вертикальная линия)
		fm = (lx - rx) / (ry - ly)
		fb = fy - fm*fx
	}

	// вертикальное
	if equalEps(ry, ly) {
		// вышли за границы, не надо соедпинять
		if fx < xl || fx >= xr {
			return false
		}
		// вниз
		if lx > rx {
			if va == NO_VERTEX {
				va = Vertex{fx, yt}
			} else if va.Y >= yb {
				return false
			}
			vb = Vertex{fx, yb}
			// иначе вверх
		} else {
			if va == NO_VERTEX {
				va = Vertex{fx, yb}
			} else if va.Y < yt {
				return false
			}
			vb = Vertex{fx, yt}
		}

	} else if fm < -1 || fm > 1 {
		// вниз
		if lx > rx {
			if va == NO_VERTEX {
				va = Vertex{(yt - fb) / fm, yt}
			} else if va.Y >= yb {
				return false
			}
			vb = Vertex{(yb - fb) / fm, yb}
			// вверх
		} else {
			if va == NO_VERTEX {
				va = Vertex{(yb - fb) / fm, yb}
			} else if va.Y < yt {
				return false
			}
			vb = Vertex{(yt - fb) / fm, yt}
		}

	} else {
		// вправо
		if ly < ry {
			if va == NO_VERTEX {
				va = Vertex{xl, fm*xl + fb}
			} else if va.X >= xr {
				return false
			}
			vb = Vertex{xr, fm*xr + fb}
			// влево
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

// обрезаем ребро, если за границы вышло
// используется алгоритм Лианга-Барски
func clipEdge(edge *edge, bbox BoundingBox) bool {
	ax := edge.Va.X
	ay := edge.Va.Y
	bx := edge.Vb.X
	by := edge.Vb.Y
	t0 := float64(0)
	t1 := float64(1)
	dx := bx - ax
	dy := by - ay

	// влево
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
	// вправо
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

	// вверх
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
	// вниз
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

func equalEps(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func lessThanEps(a, b float64) bool {
	return b-a > 1e-9
}

func moreThanEps(a, b float64) bool {
	return a-b > 1e-9
}

// ограничиваем все ребра (отрезки), чтоб за гр bbox не вышли
func (v *Voronoi) clipEdges(bbox BoundingBox) {
	abs_fn := math.Abs

	for i := len(v.edges) - 1; i >= 0; i-- {
		edge := v.edges[i]

		if !connectEdge(edge, bbox) || !clipEdge(edge, bbox) || (abs_fn(edge.Va.X-edge.Vb.X) < 1e-9 && abs_fn(edge.Va.Y-edge.Vb.Y) < 1e-9) {
			edge.Va.Vertex = NO_VERTEX
			edge.Vb.Vertex = NO_VERTEX
			v.edges[i] = v.edges[len(v.edges)-1]
			v.edges = v.edges[0 : len(v.edges)-1]
		}
	}
	//v.Logger.Info("[f-for-rm-bs-detach-ce] Первое событие круга", zap.Float64("ce_x", v.firstCircleEvent.x), zap.Float64("ce_y", v.firstCircleEvent.y))
}

// закрываем ячейки, гарантируя, что каждая ячейка внутри bbox
func (v *Voronoi) closeCells(bbox BoundingBox) {
	left := bbox.Xl
	right := bbox.Xr
	top := bbox.Yt
	bottom := bbox.Yb
	cells := v.cells

	for _, cell := range cells {
		// Пропускаем ячейки без рёбер
		if cell.prepare() == 0 {
			continue
		}

		halfEdges := cell.halfEdges
		numHalfEdges := len(halfEdges)

		currentEdgeIdx := 0
		for currentEdgeIdx < numHalfEdges {
			nextEdgeIdx := (currentEdgeIdx + 1) % numHalfEdges
			endPoint := halfEdges[currentEdgeIdx].endPoint()
			startPoint := halfEdges[nextEdgeIdx].startPoint()

			// Проверка на наличие зазора между текущим и следующим полурёбрами
			if math.Abs(endPoint.X-startPoint.X) >= 1e-9 || math.Abs(endPoint.Y-startPoint.Y) >= 1e-9 {
				startVertex := endPoint
				endVertex := endPoint

				// Идём вниз вдоль левой границы
				if equalEps(endPoint.X, left) && lessThanEps(endPoint.Y, bottom) {
					if equalEps(startPoint.X, left) {
						endVertex = Vertex{left, startPoint.Y}
					} else {
						endVertex = Vertex{left, bottom}
					}

					// Идём вправо вдоль нижней границы
				} else if equalEps(endPoint.Y, bottom) && lessThanEps(endPoint.X, right) {
					if equalEps(startPoint.Y, bottom) {
						endVertex = Vertex{startPoint.X, bottom}
					} else {
						endVertex = Vertex{right, bottom}
					}

					// Идём вверх вдоль правой границы
				} else if equalEps(endPoint.X, right) && moreThanEps(endPoint.Y, top) {
					if equalEps(startPoint.X, right) {
						endVertex = Vertex{right, startPoint.Y}
					} else {
						endVertex = Vertex{right, top}
					}

					// Идём влево вдоль верхней границы
				} else if equalEps(endPoint.Y, top) && moreThanEps(endPoint.X, left) {
					if equalEps(startPoint.Y, top) {
						endVertex = Vertex{startPoint.X, top}
					} else {
						endVertex = Vertex{left, top}
					}
				}

				newEdge := v.createBorderEdge(cell, startVertex, endVertex)
				cell.halfEdges = append(cell.halfEdges, nil)
				halfEdges = cell.halfEdges
				numHalfEdges = len(halfEdges)

				// Вставляем новое полуребро для замыкания ячейки
				copy(halfEdges[currentEdgeIdx+2:], halfEdges[currentEdgeIdx+1:len(halfEdges)-1])
				halfEdges[currentEdgeIdx+1] = newHalfEdge(newEdge, cell, nil)
			}
			currentEdgeIdx++
		}
	}
}

func (v *Voronoi) gatherVertexEdges() {
	vertexEdgeMap := make(map[Vertex][]*edge)

	for _, edge := range v.edges {
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
