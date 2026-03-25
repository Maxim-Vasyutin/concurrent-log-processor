# Log Processor

## 📋 Описание проекта

Log Processor — это CLI-инструмент для анализа логов в микросервисной архитектуре.

Он помогает быстро находить причины сбоев, объединяя разрозненные записи из разных сервисов в единую хронологию по request_id. Это особенно полезно при расследовании инцидентов, когда один запрос проходит через несколько сервисов.

Инструмент решает проблемы:
- ручного поиска по логам (grep/awk)
- восстановления цепочки событий
- локализации точки отказа

Технологии:
- Go
- конкурентная обработка (goroutines, channels, worker pool)
- потоковая обработка данных

---

## 🛠 Основные функции

- Сопоставление логов по request_id  
- Обнаружение ошибок (ERROR / WARN)  
- Восстановление хронологии событий  
- Конкурентная обработка файлов  
- Генерация JSON-отчетов  
- CLI-интерфейс для автоматизации  

---

## 🚀 Установка и запуск

### Требования

- Go 1.21+

### Установка

git clone https://github.com/your-repo/log-processor.git  
cd log-processor  
go build -o log-processor  

### Запуск

# базовый запуск
./log-processor

# указание директории с логами
./log-processor --input-dir ./logs

# указание выходного файла
./log-processor --input-dir ./logs --output-file result.json

---

## 🎮 Примеры использования

./log-processor --input-dir /var/log/microservices/

Вывод:

Processing 50 files with 8 workers...  
✓ Completed successfully in 2.1 seconds  

При прерывании:

^C  
Received interrupt signal, shutting down gracefully...  
✓ Partial results saved  

---

## 📊 Формат вывода

Пример JSON-отчета:

{
  "total_entries_processed": 1250,
  "failed_requests_found": 23,
  "processing_time_seconds": 2.3,
  "failed_requests": [
    {
      "request_id": "req_abc123",
      "failing_service": "payment-service",
      "error_message": "Card declined",
      "timeline": [
        "2023-12-25T14:30:15.123Z [INFO] user-service: User authenticated",
        "2023-12-25T14:30:15.567Z [ERROR] payment-service: Card declined"
      ]
    }
  ]
}

---

## ⚡ Производительность

- Используется worker pool для параллельной обработки файлов  
- Ограниченная конкурентность (bounded concurrency)  
- Потоковая обработка без загрузки всех данных в память  
- Ускорение обработки в несколько раз по сравнению с последовательным режимом  

---

## 🏗 Архитектура

Проект построен по layered architecture:

cli → scanner → processor → reporter

Структура проекта:

internal/
  cli/        → обработка CLI аргументов  
  parser/     → парсинг логов  
  scanner/    → поиск файлов  
  processor/  → обработка и анализ  
  reporter/   → генерация JSON  

Основные принципы:

- разделение ответственности  
- потоковая обработка  
- bounded concurrency (worker pool)  
- graceful shutdown через context  

---

## 📌 Применение

Инструмент может использоваться для:

- анализа production-инцидентов  
- постмортем-разборов  
- локальной диагностики микросервисов  
- интеграции в CI/CD пайплайны  

---

## 🧠 Почему это важно

В распределённых системах невозможно понять причину ошибки без восстановления полной цепочки событий.

Log Processor автоматизирует этот процесс и сокращает время анализа с часов до секунд.