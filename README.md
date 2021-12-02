# dapr-middleware-http-filter
a customized middleware of DAPR. 


filter 集成测试步骤
1. 运行 webserver.go。 端口8001，只能通过IP访问，不能使用localhost,127.0.0.1访问。
2. filter yaml的filter_url 指向上一步的webserver的地址。
3. postman 访问 filter部署到的dapr环境。

Docker Image的创建备注：
命令： make docker-build
1. env   
export DAPR_REGISTRY=gatty.pcauto 
export DAPR_TAG=dev
(a.本地设置了gatty.pcauto， 通过本地镜像，直接在k8s运行, 在dapr用户build， 导到gatty用户minik8s 运行. ）

b.dapr用户是使用host方式运行dapr，开发和调试的.
c.在k8s上部署自己定制dapr的middleware时，只需修改业务微服务上的dapr sidecar-image的注入就可以了，dapr-system namespace下面的DAPR 原始deploy是不需要任何修改的。
d.业务微服务注入包含了filter的daprd的写法：
annotations:
    dapr.io/sidecar-image: gatty.pcauto:dev...
    dapr.io/sidecar-listen-addresses: 0.0.0.0


2. docker 设置
sudo vi /etc/systemd/system/docker.service.d/http-proxy.conf

[Service]
Environment="HTTP_PROXY=192.168.11.210:9090" "HTTPS_PROXY=192.168.11.210:9090" "NO_PROXY=localhost,127.0.0.1,192.168.49.2"

启动minikube 时，如果要访问墙外url，加上proxy env, 再启动minikube。

export HTTP_PROXY=http://192.168.11.210:9090
export HTTPS_PROXY=http://192.168.11.210:9090
export NO_PROXY=localhost,127.0.0.1,192.168.49.2(缺省已经有了，可以不运行)


3. 注意：
自定义http的header时， 不能使用"_", 应该使用"-"， 这个才有更好兼容性，否则会给过滤掉。 例如"Test-Header-Key"。
