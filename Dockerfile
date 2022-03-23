FROM golang
MAINTAINER xuri.li

RUN mkdir /usr/local/goc
ADD . /usr/local/goc

RUN go env -w GO111MODULE=on
RUN go env -w GOPROXY=https://goproxy.cn

# RUN go get github.com/aliyun/aliyun-oss-go-sdk
# RUN go get github.com/astaxie/beego
# RUN go get github.com/go-sql-driver/mysql
# RUN go get github.com/pkg/errors
# RUN go get github.com/satori/go.uuid
# RUN go get go.mongodb.org/mongo-driver
# RUN go get golang.org/x/time

WORKDIR /usr/local/goc

RUN go build

CMD ["go", "run",  "/usr/local/goc/"]
