# Практическое занятие 14
## Шишков А.Д. ЭФМО-02-25
## Тема
Реализация очереди задач. Модель producer-consumer, повторные попытки обработки, DLQ и идемпотентность

## Цель
Освоить построение очереди задач по модели producer-consumer с использованием RabbitMQ, научиться организовывать повторные попытки обработки, настраивать очередь проблемных сообщений DLQ, а также реализовывать базовую идемпотентность обработчика для защиты от повторной обработки одного и того же сообщения.

---

## Краткое описание реализованного решения

В существующий проект добавлена очередь задач `task_jobs`. Сервис `tasks` принимает HTTP-запрос на постановку задачи в очередь и публикует сообщение в RabbitMQ. Отдельный сервис `worker` получает сообщение, проверяет его по `message_id`, имитирует обработку и затем:

- подтверждает сообщение через `Ack(false)`, если обработка успешна;
- повторно публикует сообщение в основную очередь, если произошла ошибка и лимит попыток еще не исчерпан;
- публикует сообщение в очередь проблемных сообщений `task_jobs_dlq`, если число попыток превышено.

Для учебной демонстрации используется простое правило:

- если `task_id == "t_fail"`, обработка всегда завершается ошибкой;
- для остальных значений `task_id` обработка считается успешной.

Идемпотентность реализована на учебном уровне: `worker` хранит уже обработанные `message_id` в памяти и не выполняет одну и ту же задачу повторно.

---

## Используемые компоненты

- `services/tasks` - HTTP-сервис, публикующий job в очередь;
- `services/worker` - отдельный consumer, обрабатывающий задачи из очереди;
- `services/auth` - сервис авторизации для доступа к `tasks`;
- `RabbitMQ` - брокер сообщений;
- `PostgreSQL` - хранилище задач;
- `Redis` - кэш в сервисе `tasks`.

---

## Запуск RabbitMQ

Для работы используется Docker-контейнер с management-интерфейсом.

```yaml
version: "3.9"

services:
  rabbitmq:
    image: rabbitmq:3-management
    container_name: pz14-rabbitmq
    ports:
      - "5672:5672"
      - "15672:15672"
    environment:
      RABBITMQ_DEFAULT_USER: guest
      RABBITMQ_DEFAULT_PASS: guest
```

Запуск:

```bash
cd deploy/rabbit
docker compose up -d
docker compose ps
```

После запуска доступны:

- AMQP-порт `5672` для приложений;
- RabbitMQ Management UI: [http://localhost:15672](http://localhost:15672);
- логин и пароль: `guest / guest`.


<img width="2538" height="904" alt="image" src="https://github.com/user-attachments/assets/44e057c0-f883-4f6b-8187-5700b7485ef7" /> 


---

## Формат сообщения

Сообщение задачи публикуется в формате JSON и содержит обязательные поля:

```json
{
  "job": "process_task",
  "task_id": "t_001",
  "attempt": 1,
  "message_id": "uuid-here"
}
```

Структура сообщения:

```go
type TaskJob struct {
    Job       string `json:"job"`
    TaskID    string `json:"task_id"`
    Attempt   int    `json:"attempt"`
    MessageID string `json:"message_id"`
}
```

Описание полей:

- `job` - тип выполняемой задачи;
- `task_id` - идентификатор бизнес-объекта;
- `attempt` - номер текущей попытки обработки;
- `message_id` - уникальный идентификатор сообщения.

---

## Очереди `task_jobs` и `task_jobs_dlq`

В проекте используются две очереди:

- основная очередь задач `task_jobs`;
- очередь проблемных сообщений `task_jobs_dlq`.

Очереди объявляются как `durable` и используются сервисами `tasks` и `worker`.

Параметры очередей:

- `durable = true`
- `autoDelete = false`
- `exclusive = false`

Это означает:

- очереди сохраняются после перезапуска RabbitMQ;
- очереди не удаляются автоматически;
- очереди не привязаны к одному клиентскому соединению.

---

## Producer: постановка задачи в очередь

В сервисе `tasks` реализован endpoint:

```text
POST /v1/jobs/process-task
```

Тело запроса:

```json
{
  "task_id": "t_001"
}
```

После получения запроса сервис:

- проверяет входные данные;
- формирует сообщение `TaskJob`;
- назначает `attempt = 1`;
- генерирует `message_id`;
- публикует сообщение в `task_jobs`;
- возвращает клиенту подтверждение.

Пример ответа:

```json
{
  "status": "accepted",
  "task_id": "t_001",
  "message_id": "uuid-here",
  "attempt": 1
}
```

---

## Consumer: обработка задач в сервисе `worker`

Для обработки сообщений реализован отдельный сервис `worker`.

Основные характеристики consumer:

- подключается к RabbitMQ;
- читает сообщения из `task_jobs`;
- проверяет `message_id`;
- имитирует обработку;
- при ошибке делает повторную попытку;
- после превышения лимита переводит сообщение в `task_jobs_dlq`.

---

## Повторные попытки обработки

В проекте реализована простая стратегия retry.

Если обработка завершилась ошибкой:

- `worker` увеличивает `attempt`;
- если лимит попыток не превышен, публикует сообщение заново в `task_jobs`;
- если лимит превышен, публикует сообщение в `task_jobs_dlq`;
- исходное сообщение подтверждает через `Ack(false)`.

По умолчанию используется:

```text
MAX_ATTEMPTS=3
```

Это означает:

- первая попытка;
- вторая попытка после ошибки;
- третья попытка после повторной ошибки;
- после следующей ошибки сообщение переводится в DLQ.

---

## Идемпотентная проверка по `message_id`

Перед выполнением обработки `worker` проверяет, не было ли сообщение уже обработано ранее.

Для учебной версии используется хранение обработанных `message_id` в памяти.

Если сообщение с тем же `message_id` приходит повторно:

- обработчик не выполняет задачу повторно;
- сообщение сразу подтверждается через `Ack(false)`.

---

## Переменные окружения

### Для сервиса `tasks`

- `TASKS_PORT` - порт сервиса, по умолчанию `8082`
- `DATABASE_URL` - строка подключения к PostgreSQL
- `AUTH_MODE` - режим авторизации: `http` или `grpc`
- `AUTH_BASE_URL` - адрес HTTP auth-сервиса
- `AUTH_GRPC_ADDR` - адрес gRPC auth-сервиса
- `REDIS_ADDR` - адрес Redis
- `RABBIT_URL` - адрес RabbitMQ, по умолчанию `amqp://guest:guest@localhost:5672/`
- `EVENT_QUEUE_NAME` - очередь событий, по умолчанию `task_events`
- `JOB_QUEUE_NAME` - имя основной очереди, по умолчанию `task_jobs`
- `JOB_DLQ_NAME` - имя DLQ, по умолчанию `task_jobs_dlq`

### Для сервиса `worker`

- `RABBIT_URL` - адрес RabbitMQ, по умолчанию `amqp://guest:guest@localhost:5672/`
- `JOB_QUEUE_NAME` - имя основной очереди, по умолчанию `task_jobs`
- `JOB_DLQ_NAME` - имя DLQ, по умолчанию `task_jobs_dlq`
- `WORKER_PREFETCH` - значение `prefetch`, по умолчанию `1`
- `MAX_ATTEMPTS` - максимальное число попыток, по умолчанию `3`

---

## Демонстрация работы

### Подготовка зависимостей

Перед запуском сервисов должны быть доступны:

- PostgreSQL на `5432`;
- Redis на `6379`;
- RabbitMQ на `5672` и `15672`.

### Запуск auth-сервиса

```bash
go run ./services/auth/cmd/auth
```

### Запуск worker

```powershell
$env:RABBIT_URL="amqp://guest:guest@localhost:5672/"
$env:JOB_QUEUE_NAME="task_jobs"
$env:JOB_DLQ_NAME="task_jobs_dlq"
$env:WORKER_PREFETCH="1"
$env:MAX_ATTEMPTS="3"
go run ./services/worker/cmd/worker
```

### Запуск tasks

```powershell
$env:RABBIT_URL="amqp://guest:guest@localhost:5672/"
$env:JOB_QUEUE_NAME="task_jobs"
$env:JOB_DLQ_NAME="task_jobs_dlq"
$env:REDIS_ADDR="127.0.0.1:6379"
go run ./services/tasks/cmd/tasks
```

### Получение токена

Параметры запроса:

- метод: `POST`
- URL: `http://localhost:8081/v1/auth/login`
- заголовок: `Content-Type: application/json`
- тело:

```json
{
  "username": "student",
  "password": "student"
}
```

Ожидаемый ответ:

```json
{
  "access_token": "demo-token",
  "token_type": "Bearer"
}
```


<img width="1463" height="911" alt="image" src="https://github.com/user-attachments/assets/39bfc57a-b163-4d56-b2f4-ba15eddbf138" /> 


### Успешная обработка job

Команда:

```powershell
curl -i -X POST http://localhost:8082/v1/jobs/process-task `
  -H "Content-Type: application/json" `
  -H "Authorization: Bearer demo-token" `
  -d "{\"task_id\":\"t_001\"}"
```

Ожидаемый результат:

- сервис `tasks` возвращает `202 Accepted`;
- сообщение публикуется в `task_jobs`;
- `worker` получает job;
- обработка завершается успешно;
- сообщение подтверждается через `Ack(false)`.

Пример ответа:

```json
{
  "status": "accepted",
  "task_id": "t_001",
  "message_id": "uuid-here",
  "attempt": 1
}
```

<img width="1448" height="949" alt="image" src="https://github.com/user-attachments/assets/d0f5392d-8117-4077-80cb-c41a0876bd95" /> 


<img width="2078" height="51" alt="image" src="https://github.com/user-attachments/assets/78a12c2a-d038-48b8-9cc2-dcf420588ac9" /> 



### Обработка с ошибкой, retries и DLQ

Команда:

```powershell
curl -i -X POST http://localhost:8082/v1/jobs/process-task `
  -H "Content-Type: application/json" `
  -H "Authorization: Bearer demo-token" `
  -d "{\"task_id\":\"t_fail\"}"
```

Ожидаемое поведение:

- первая попытка завершается ошибкой;
- `worker` публикует сообщение повторно с увеличенным `attempt`;
- вторая попытка завершается ошибкой;
- третья попытка завершается ошибкой;
- после превышения лимита сообщение публикуется в `task_jobs_dlq`.

В логах `worker` видны строки:

```text
task job received
task job failed
job scheduled for retry
job published to dlq
```

<img width="2247" height="202" alt="image" src="https://github.com/user-attachments/assets/eab33724-aaf3-4a52-bfd5-6bb65e4c8b87" />


### Проверка через RabbitMQ Management UI

Открыть [http://localhost:15672](http://localhost:15672), войти под `guest / guest`, затем перейти в раздел `Queues and Streams`.

На странице видно:

- очередь `task_jobs`;
- очередь `task_jobs_dlq`;
- наличие consumer у основной очереди;
- попадание сообщения в DLQ после неудачной обработки.

<img width="1899" height="774" alt="image" src="https://github.com/user-attachments/assets/b333c269-cc7a-4dc7-a897-66f72010c322" /> 


## Выводы

- В проект успешно добавлена очередь задач `task_jobs`.
- Отдельный сервис `worker` обрабатывает задачи по модели producer-consumer.
- Реализовано ограничение числа попыток обработки через поле `attempt`.
- После превышения лимита сообщение переводится в очередь `task_jobs_dlq`.
- Реализована учебная идемпотентность по `message_id`.
- Показан полный сценарий прохождения сообщения: от HTTP-запроса до успешной обработки или отправки в DLQ.
