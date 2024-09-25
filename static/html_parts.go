package static

var (
	Part1 = `
    <!DOCTYPE html>
    <html>
    <head>
        <title>Диаграмма Вороного</title>
		<style>
			body {
				background-color: #1F1F1F; /* Темный фон для всей страницы */
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
				border-left: 5px solid #757575; /* Темная граница для правого контейнера */
				overflow-y: auto; /* Вертикальная прокрутка для логов */
				overflow-x: auto; /* Вертикальная прокрутка для логов */
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
    `

	Part2 = `
            </div>
            <div id="right-container">
                <h1>Логи</h1>
                <div id="logs">`

	Part3 = `
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
    `
)
