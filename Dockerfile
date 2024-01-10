# Build stage

# 这一行从Docker Hub上拉取一个带有Go语言1.21版本和Alpine Linux 3.18的基础镜像，
# 并命名这个构建阶段为builder。
# WORKDIR /app: 设置工作目录为/app，如果不存在会创建它。
# COPY . .: 将当前目录下的所有文件和目录复制到容器中的/app目录。
# RUN go build -o main main.go: 执行go build命令来编译main.go文件，
# 并生成执行文件命名为main。

FROM golang:1.21-alpine3.18 AS builder
WORKDIR /app
COPY . .
RUN go build -o main main.go
# RUN apk add curl
# RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz
        

# 这一行开始新的阶段，从Docker Hub上拉取一个Alpine Linux 3.18的基础镜像用于运行应用。
# WORKDIR /app: 同样设置工作目录为/app。
# COPY --from=builder /app/main .: 从前一个构建阶段（builder）中复制编译好的main执行文件到当前工作目录下。
# COPY app.env .: 复制app.env文件到容器中的/app目录。
# COPY start.sh .: 复制start.sh脚本到容器中的/app目录。
# COPY wait-for.sh .: 复制wait-for.sh脚本到容器中的/app目录，这个脚本通常用于等待数据库等服务启动后再启动应用。
# COPY db/migration ./db/migration: 复制本地的db/migration目录及其内容到容器中的/app/db/migration目录。

# Run stage
FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/main .
# COPY --from=builder /app/migrate ./migrate
COPY app.env .
COPY start.sh .
COPY wait-for.sh .
COPY db/migration ./db/migration


# EXPOSE 8080 9090: 声明容器在运行时监听8080和9090端口，这通常是应用服务的端口。
# CMD [ "/app/main" ]: 指定容器启动时默认执行的命令，这里是执行/app/main程序。
# ENTRYPOINT [ "/app/start.sh" ]: 指定容器启动时要运行的脚本，这里是start.sh。
# 这个脚本会被设置为容器的默认入口点，意味着它会首先被执行，
# 并且CMD中的/app/main会作为参数传递给start.sh脚本。

EXPOSE 8080 9090
CMD [ "/app/main" ]
ENTRYPOINT [ "/app/start.sh" ]





# Docker是一个开源平台，用于自动化开发、部署和运行应用程序的容器化。
# 容器化是一种轻量级的虚拟化方法，可以让你的应用和服务在隔离的环境中运行，
# 这使得部署和迁移更加简单和高效。

# 通过执行docker build -t bank:latest .命令，你就创建了一个Docker镜像，
# 这个镜像现在可以用来运行你的Go应用程序。
# 创建这个镜像的目的通常是为了确保应用程序可以在任何提供Docker支持的环境中以相同的方式运行，
# 从而简化部署和运维工作。

# 镜像构建完成后，你可以通过类似docker run的命令来启动一个基于这个镜像的容器。
# 如果一切设置正确，你的Go应用程序将在容器中启动并开始服务，监听在指定的端口上。
# 这样，你就能够在开发、测试或生产环境中以一致、可预测的方式部署你的应用了。