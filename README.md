# Transaction Processing System

## Описание

Это приложение реализует систему обработки транзакций с использованием Go и Gin. Приложение работает с базой данных SQLite и предоставляет REST API для отправки транзакций, получения баланса кошельков и истории транзакций.

API защищено ограничением по количеству запросов (1 запрос в секунду на IP) и включает проверку валидности данных для предотвращения ошибок при обработке транзакций.

## Особенности

- **SQLite**: Хранение информации о кошельках и транзакциях.
- **Gin**: Web-фреймворк для создания REST API.
- **Tollbooth**: Ограничение количества запросов на единицу времени (лимит 1 запрос в секунду).
- **Slog**: Логирование с возможностью выводить данные в читабельном формате.

## Установка

1. Склонируйте репозиторий:
   ```bash
   git clone https://github.com/karmin000/infotecs-tz.git
   cd infotecs-tz
   ```
2. Установите зависимости
   ```bash
   go mod tidy
   ```
3. Запустите приложение:
   ```bash
   go run main.go
   ```
Приложение будет доступно по адресу http://localhost:8080

## API
1. **Send** POST /api/send
   Отправляет средства с одного кошшелька на другой

   Параметры запроса:
   - **from**: Адрес отправителя (HEX, 64 символа).
   - **to**: Адрес получателя (HEX, 64 символа).
   - **amount**: Сумма перевода (например, 3.50).

Пример запроса:
```json
{
  "from": "e240d825d255af751f5f55af8d9671beabdf2236c0a3b4e2639b3e182d994c88",
  "to": "f440d825d255af751f5f55af8d9671beabdf2236c0a3b4e2639b3e182d994c88",
  "amount": 10.0
}
```
Пример ответа:
```json
{
  "id": 1,
  "from": "e240d825d255af751f5f55af8d9671beabdf2236c0a3b4e2639b3e182d994c88",
  "to": "f440d825d255af751f5f55af8d9671beabdf2236c0a3b4e2639b3e182d994c88",
  "amount": 10.0,
  "timestamp": "2025-02-24T15:00:00Z"
}
```
  Ошибки:
   - **400 Bad Request**: Неверный формат запроса или адреса.
   - **400 Bad Request**: Недостаточно средств на счёте.
   - **400 Bad Request**: Отправитель и получатель не могут быть одинаковыми.

2. **GetLast** GET /api/transactions?count=N
   Получить список последних транзакций
   Параметры запроса:
   - **count** (необязательный): Количество транзакций для получения (по умолчанию 10).
Пример ответа:
```json
[
  {
    "id": 1,
    "from": "e240d825d255af751f5f55af8d9671beabdf2236c0a3b4e2639b3e182d994c88",
    "to": "f440d825d255af751f5f55af8d9671beabdf2236c0a3b4e2639b3e182d994c88",
    "amount": 10.0,
    "timestamp": "2025-02-24T15:00:00Z"
  },
  ...
]
```
3. **GetBalance** GET /api/wallet/{address}/balance
   Получить баланс кошелька по адресу
   Параметры запроса:
   - **address**: Адрес кошелька.
Пример ответа:
```json
{
  "address": "e240d825d255af751f5f55af8d9671beabdf2236c0a3b4e2639b3e182d994c88",
  "balance": 100.0
}
```


   


