Usage
=====
```
docker-compose up
```

Methods
=====
```
POST /create RETURN id
Example:
curl \
-X POST http://localhost:8080/create \
-H 'Content-Type: application/json' \
-d '{"name": "test", "desc": "test", "images": ["url1", "url2"], "price": 30}'
```
```
GET /getall
Params:
- pagination=0 - offset 0*10, 1*10 and so on
- price=[desc/asc]
- created_at=[desc/asc]
Example:
curl \
-X GET 
http://localhost:8080/getall\?pagination\=0\&price\=desc
```
```
GET /get
Params:
- id
- fields=[description,images]
Example:
curl \
-X GET \
http://localhost:8080/get\?id\=4\&fields\=description
```
