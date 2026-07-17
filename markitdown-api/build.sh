docker build -t markitdown-api:v1.0.0 .
docker run -d -p 8001:8001 --env-file .env markitdown-api