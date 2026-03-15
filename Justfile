set dotenv-load

generate:
    go generate ./...

run:
    go run cmd/lazypubk/main.go

docker-build version="latest":
    docker build -t lazy-dvc:{{version}} .

docker-run version="latest":
    docker run -d -p 2222:22 --name lazy-dvc-{{version}} lazy-dvc:{{version}}

docker-stop version="latest":
    docker stop lazy-dvc-{{version}} && docker rm lazy-dvc-{{version}}

docker-logs version="latest":
    docker logs -f lazy-dvc-{{version}}

ssh-iter:
    ssh -p 2222 -o "UserKnownHostsFile=/dev/null" -o "StrictHostKeyChecking=no" dvc-storage@localhost "ls -la /home/dvc-storage/data"
