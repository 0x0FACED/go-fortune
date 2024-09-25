package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/0x0FACED/go-fortune/pkg/logger"
	"github.com/0x0FACED/go-fortune/pkg/voronoi"
	"github.com/0x0FACED/go-fortune/static"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

type Station struct {
	X, Y float64
}

// Генерируем случайные точки для станций
func generateStations(n int, width, height int) []Station {
	stations := make([]Station, n)
	rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < n; i++ {
		stations[i] = Station{
			X: float64(rand.Intn(width)),
			Y: float64(rand.Intn(height)),
		}
	}
	return stations
}

func prepareScatter(scatter *charts.Scatter) {
	scatter.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Height: "580px",
			Width:  "1020px",
		}),
		charts.WithLegendOpts(opts.Legend{
			TextStyle: &opts.TextStyle{
				Color: "white",
			},
			Right: "10%",
		}),
		charts.WithTitleOpts(opts.Title{
			Title:                "Диаграмма Вороного (Форчун)",
			TitleBackgroundColor: "white",
			Left:                 "10%",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Type: "value",
			Name: "Ширина",
			AxisLabel: &opts.AxisLabel{
				Color: "white",
			},
			SplitLine: &opts.SplitLine{
				Show: opts.Bool(false),
			},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Type: "value",
			Name: "Высота",
			AxisLabel: &opts.AxisLabel{
				Color: "white",
			},
			SplitLine: &opts.SplitLine{
				Show: opts.Bool(false),
			},
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Type:       "inside",
			Start:      0,
			End:        100,
			FilterMode: "none",
			Orient:     "horizontal",
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Type:       "inside",
			Start:      0,
			End:        100,
			FilterMode: "none",
			Orient:     "vertical",
		}),
	)
}

// Преобразуем voronoi границы в Echarts для отображения
func voronoiToEcharts(stations []Station, diagram *voronoi.Diagram) *charts.Scatter {
	scatter := charts.NewScatter()

	points := make([]opts.ScatterData, 0)
	for _, station := range stations {
		points = append(points, opts.ScatterData{
			Value: []float64{station.X, station.Y},
		})
	}

	// Дизайним скаттер
	prepareScatter(scatter)

	scatter.AddSeries("Станции", points).
		SetSeriesOptions(
			charts.WithItemStyleOpts(opts.ItemStyle{
				Color: "lightgreen",
			}),
		)

	for _, edge := range diagram.Edges {
		line := charts.NewLine()
		line.SetGlobalOptions(
			charts.WithXAxisOpts(opts.XAxis{Show: opts.Bool(true)}),
			charts.WithYAxisOpts(opts.YAxis{Show: opts.Bool(true)}),
		)

		line.AddSeries("Границы", []opts.LineData{
			{Value: []float64{edge.Va.X, edge.Va.Y}},
			{Value: []float64{edge.Vb.X, edge.Vb.Y}},
		}).SetSeriesOptions(
			charts.WithLineStyleOpts(opts.LineStyle{
				Width: 2,
			}),
		)

		scatter.Overlap(line)
	}

	return scatter
}

// http обработчик страницы с диаграмой и формой для ввода данных
func diagramHandler(w http.ResponseWriter, r *http.Request) {
	width := 1000
	height := 1000
	numStations := 10

	if r.Method == http.MethodPost {
		r.ParseForm()
		width, _ = strconv.Atoi(r.FormValue("width"))
		height, _ = strconv.Atoi(r.FormValue("height"))
		numStations, _ = strconv.Atoi(r.FormValue("stations"))
		fmt.Println("Data: ", width, height, numStations)
	}

	// Генерация станций
	stations := generateStations(numStations, width, height)

	// Генерация точек для алгоритма Форчуна
	var points []voronoi.Vertex
	for _, station := range stations {
		points = append(points, voronoi.Vertex{X: station.X, Y: station.Y})
	}

	// Создаем bounding box
	bbox := voronoi.NewBoundingBox(0, float64(width), 0, float64(height))

	// Создаем логгер для записи логов и вывода их на страницу
	logger := logger.New()
	defer logger.ClearLogs()

	// Создаем диаграмму Вороного с логгером
	diagram := voronoi.CreateDiagram(points, bbox, true, logger)

	// Конвертация диаграммы в HTML через Echarts
	scatter := voronoiToEcharts(stations, diagram)

	// Генерация HTML-страницы с обновленной диаграммой и логами
	fmt.Fprintln(w, static.Part1)

	// Встраиваем диаграмму в HTML
	err := scatter.Render(w)
	if err != nil {
		fmt.Println("Ошибка рендеринга диаграммы:", err)
	}

	fmt.Fprintln(w, static.Part2)

	// Вставляем логи в HTML
	for _, log := range logger.Logs {
		fmt.Fprintln(w, log)
	}

	fmt.Fprintln(w, static.Part3)
}

func main() {
	http.HandleFunc("/", diagramHandler)
	fmt.Println("Сервер запущен на http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Err ListenAndServe", err)
	}
}
