### Описание запуска проекта
---
Нужно в папке rateLimiting запустить

```
docker-compose up --build
```
Сбилдятся 2 контейнера со своими переменными окружения.

По проекту:

1) Настроены middleware как декораторы
2) Настроены CRUD операции для работы с клиентами
3) Паники отлавливаются middleware