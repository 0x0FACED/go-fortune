package voronoi

import (
	"math"
	"sort"

	"github.com/0x0FACED/go-fortune/pkg/logger"
	"go.uber.org/zap"
)

// Основная функция - база
// Это основной алгоритм, где вызываются остальные функции/методы
func CreateDiagram(sites []Vertex, bbox BoundingBox, closeCells bool, logger *logger.ZapLogger) *Diagram {
	// sites - точки (вершины)
	v := &Voronoi{
		cellsMap: make(map[Vertex]*cell),
		Logger:   logger,
	}

	logger.Info("[f] Алгоритм Форчуна запущен")

	// сортируем по Y, чтобы гарантировать обработку сверху вниз (от меньших к большим)
	sort.Sort(verticesByY{sites})

	logger.Info("[f] Сайты (точки) отсортированы по Y", zap.Any("sites", sites))
	// функция для имитации очереди
	// получаем первую вершину и удаляем ее из слайса
	pop := func() *Vertex {
		if len(sites) == 0 {
			return nil
		}

		site := sites[0]
		sites = sites[1:]
		return &site
	}

	// берем первую вершину
	site := pop()

	logger.Info("[f] Первая вершина", zap.Any("site", site))
	// предыдущие точки
	prevSiteX := math.SmallestNonzeroFloat64
	prevSiteY := math.SmallestNonzeroFloat64
	var circle *circleEvent

	var counter int
	logger.Info("[f] Основной цикл начат")
	// основной цикл
	for {
		v.Logger.Info("[f-for] ===============================================================================================")
		v.Logger.Info("[f-for] Текущая итерация", zap.Int("c", counter))
		v.Logger.Info("[f-for] Осталось сайтов", zap.Int("sites", len(sites)))
		counter++
		// site event - когда мы пересекаем точку
		// circle event - когда три параболы пересекаются и образуют вершину (пересечение)
		// надо узнать, какое событие мы обрабатываем, поэтому мы узнаем,
		// есть ли site event И ПОСТУПИЛО ЛИ ОНО РАНЬШЕ
		circle = v.firstCircleEvent

		// добавляем beachsectiob

		//Если site не nil, и либо события круга нет, либо Y новой точки меньше, чем Y круга,
		// или Y совпадает, но X точки меньше, чем X круга — обрабатывается событие точки.
		if site != nil && (circle == nil || site.Y < circle.y || (site.Y == circle.y && site.X < circle.x)) {
			// Проверка на дубликат (нет смысла строить линии для точек, которые расположены
			// на одинаковых координатах)
			if site.X != prevSiteX || site.Y != prevSiteY {
				logger.Info("[f-for-site] Не дубликат", zap.Any("site", site))
				// создаем ячейку для точки
				nCell := newCell(*site)
				logger.Info("[f-for-site] Новая ячейка", zap.Any("cell", nCell))
				// добавляем в структуру вороного в ячейки новую ячейку
				v.cells = append(v.cells, nCell)
				// добавляем в мапу
				v.cellsMap[*site] = nCell
				// создаем beachsection
				logger.Info("[f-for-site] Создаем beach section")
				v.addBeachSection(*site)
				// запоминаем эти координаты для проверки на дубликаты
				prevSiteY = site.Y
				prevSiteX = site.X
			} else {
				logger.Error("[f-for-site] Найден дубликат!", zap.Any("site", site))
			}
			// достаем следующую точку
			site = pop()
			logger.Info("[f-for-site] Следующая точка", zap.Any("site", site))
		} else if circle != nil { // убираем beachsection, если круг не nil
			logger.Info("[f-for-circle] Данные круга", zap.Float64("x", circle.x), zap.Float64("y", circle.y), zap.Any("arc-site", circle.arc.site))
			v.removeBeachSection(circle.arc)

		} else { // конец
			break
		}
	}

	logger.Info("[f] Алгоритм завершен!")

	v.clipEdges(bbox)

	logger.Info("[f] Остатки соединены")

	if closeCells {
		v.closeCells(bbox)
	} else {
		for _, cell := range v.cells {
			cell.prepare()
		}
	}

	//v.gatherVertexEdges()

	return &Diagram{Edges: v.edges, Cells: v.cells}
}
