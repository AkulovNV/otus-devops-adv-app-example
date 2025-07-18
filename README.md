
# DevOps Advanced Application Example
```bash
go mod tidy
go get github.com/stretchr/testify
go test ./...
mkdir -p ./bin
go build -o ./bin/app ./cmd/main.go
```
# Jenkins Helm Chart Deployment

```bash
helm repo add jenkins https://charts.jenkins.io
helm repo update
helm upgrade --install jenkins jenkins/jenkins \
  --namespace jenkins \
  --create-namespace \
  -f jenkins/values.yaml
```