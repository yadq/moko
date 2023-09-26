FROM registry.cn-zhangjiakou.aliyuncs.com/alpd/alpine:3.17.2

COPY moko /usr/bin/moko

RUN chmod +x /usr/bin/moko

ENTRYPOINT [ "/usr/bin/moko" ]
