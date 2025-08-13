# 项目根目录 Makefile，便于一键编译和管理
.PHONY: all clean build backend frontend run

all: build

build: 
	$(MAKE) -C backend build

clean:
	$(MAKE) -C backend clean

backend:
	$(MAKE) -C backend build-backend

frontend:
	$(MAKE) -C backend build-frontend

run:
	$(MAKE) -C backend run
