package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/0x0FACED/go-fortune/pkg/logger"
	"github.com/0x0FACED/go-fortune/pkg/voronoi"

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

// Преобразуем voronoi границы в Echarts для отображения
func voronoiToEcharts(stations []Station, diagram *voronoi.Diagram) *charts.Scatter {
	scatter := charts.NewScatter()

	points := make([]opts.ScatterData, 0)
	for _, station := range stations {
		points = append(points, opts.ScatterData{
			Value: []float64{station.X, station.Y},
		})
	}

	scatter.SetGlobalOptions(
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
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Type: "value",
			Name: "Высота",
			AxisLabel: &opts.AxisLabel{
				Color: "white",
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

	scatter.AddSeries("Станции", points).
		SetSeriesOptions(
			charts.WithItemStyleOpts(opts.ItemStyle{
				Color: "red",
			}),
		)

	for _, edge := range diagram.Edges {
		if &edge.Va != nil && &edge.Vb != nil {
			line := charts.NewLine()
			line.SetGlobalOptions(
				charts.WithXAxisOpts(opts.XAxis{Show: opts.Bool(true)}),
				charts.WithYAxisOpts(opts.YAxis{Show: opts.Bool(true)}),
			)

			// Добавляем серию для границ с синим цветом
			line.AddSeries("Границы", []opts.LineData{
				{Value: []float64{edge.Va.X, edge.Va.Y}},
				{Value: []float64{edge.Vb.X, edge.Vb.Y}},
			}).SetSeriesOptions(
				charts.WithLineStyleOpts(opts.LineStyle{
					Width: 2, // Толщина линий
				}),
			)

			scatter.Overlap(line)
		}
	}

	return scatter
}

func flushLogs() {
	GlobalLogs = make([]string, 0)
}

// http обработчик страницы с диаграмой и формой для ввода данных
func diagramHandler(w http.ResponseWriter, r *http.Request) {
	flushLogs()
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
	fmt.Fprintln(w, `
    <!DOCTYPE html>
    <html>
    <head>
        <title>Диаграмма Вороного</title>
		<style>
			body {
				background-color: #1E1E1E; /* Темный фон для всей страницы */
				color: #d3d3d3; /* Светло-серый текст */
				font-family: Consolas, monospace;
				overflow: hidden; /* Запретить прокрутку */
			}

			#container {
				display: flex;
				width: 100%;
				height: 100vh;
				box-sizing: border-box;
			}

			#left-container {
				width: 50%;
				padding: 10px;
				box-sizing: border-box;
			}

			#right-container {
				width: 50%;
				padding: 10px;
				box-sizing: border-box;
				border-left: 1px solid #444; /* Темная граница для правого контейнера */
				overflow-y: auto; /* Вертикальная прокрутка для логов */
				background-color: #1e1e1e; /* Темный фон для контейнера логов */
			}

			#logs {
				white-space: pre-wrap; /* Сохраняем пробелы и переносим строки */
				word-wrap: break-word; /* Перенос длинных слов */
				color: #d3d3d3; /* Цвет текста в логах — светло-серый */
				font-family: Consolas, monospace; /* Моноширинный шрифт для логов */
			}

			#chart-container {
				width: 100%;
				height: 400px;
			}

			input[type="number"],
			input[type="submit"] {
				background-color: #2b2b2b; /* Темный фон для полей ввода */
				color: #d3d3d3; /* Светло-серый текст для полей */
				border: 1px solid #444; /* Темная граница */
				padding: 5px;
				margin: 5px 0;
				border-radius: 4px;
			}

			label {
				color: #d3d3d3; /* Светло-серый цвет для текста меток */
			}

			h1 {
				color: #d3d3d3; /* Цвет заголовка светло-серый */
			}

			input[type="submit"]:hover {
				background-color: #444; /* Немного светлее при наведении */
				cursor: pointer;
			}

			/* Добавление стилей для темной темы */
			::-webkit-scrollbar {
				width: 8px;
			}

			::-webkit-scrollbar-thumb {
				background-color: #444; /* Цвет ползунка */
				border-radius: 10px;
			}

			::-webkit-scrollbar-track {
				background-color: #2b2b2b; /* Цвет области прокрутки */
			}
        </style>
    </head>
    <body>
        <div id="container">
            <div id="left-container">
                <h1>Параметры для диаграммы Вороного</h1>
                <form id="diagram-form" method="POST">
                    <label for="width">Ширина (W):</label>
                    <input type="number" id="width" name="width" value="1000" min="100" max="5000"><br><br>
                    <label for="height">Высота (H):</label>
                    <input type="number" id="height" name="height" value="1000" min="100" max="5000"><br><br>
                    <label for="stations">Количество станций (n):</label>
                    <input type="number" id="stations" name="stations" value="10" min="1" max="200"><br><br>
                    <input type="submit" value="Построить">
                </form>
    `)

	// Встраиваем диаграмму в HTML
	err := scatter.Render(w)
	if err != nil {
		fmt.Println("Ошибка рендеринга диаграммы:", err)
	}

	fmt.Fprintln(w, `
            </div>
            <div id="right-container">
                <h1>Логи</h1>
                <div id="logs">`)

	// Вставляем логи в HTML
	for _, log := range logger.Logs {
		fmt.Fprintln(w, log)
	}

	fmt.Fprintln(w, `
                </div>
            </div>
        </div>

        <script>
            document.getElementById('diagram-form').addEventListener('submit', function (e) {
                e.preventDefault();
                const formData = new FormData(this);
                const params = new URLSearchParams(formData).toString();

                // Отправка данных формы
                fetch('/', {
                    method: 'POST',
                    body: params,
                    headers: {
                        'Content-Type': 'application/x-www-form-urlencoded'
                    }
                })
                .then(response => {
                    if (!response.ok) {
                        throw new Error('Ошибка при отправке данных');
                    }
                    return response.text(); // Получаем HTML-ответ с обновленной диаграммой и логами
                })
                .then(html => {
                    document.open(); // Очищаем текущую страницу
                    document.write(html); // Записываем обновленный HTML
                    document.close(); // Закрываем поток
                })
                .catch(error => {
                    console.error('Ошибка:', error);
                });
            });
        </script>
    </body>
    </html>
    `)
}

var GlobalLogs []string

func logHandler(w http.ResponseWriter, r *http.Request) {
	AddLog("Лог 1: Выполнен запрос на построение диаграммы")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs": GlobalLogs,
	})
}

func AddLog(message string) {
	GlobalLogs = append(GlobalLogs, message)
}

func main() {
	http.HandleFunc("/", diagramHandler)
	fmt.Println("Сервер запущен на http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Err ListenAndServe", err)
	}
}
