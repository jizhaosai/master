# mDNS 资产测绘系统

基于 mDNS/DNS-SD 协议的局域网资产发现与深度 Banner 识别工具。

## 技术栈

- Go 1.24.0 + Gin + GORM + MySQL 8.0
- 前端：原生 HTML/CSS/JS 单页面

## 快速开始

### 1. 初始化数据库

```bash
mysql -u root -p < scripts/init_db.sql
```

### 2. 修改配置

编辑 `configs/config.yaml`，设置数据库密码等参数。也可通过环境变量 `MYSQL_PASSWORD` 注入。

### 3. 编译

```bash
go build -o mdns-mapper.exe ./cmd/mdns-mapper
```

### 4. CLI 扫描模式

```bash
# 基本用法（需管理员权限绑定 5353 端口）
./mdns-mapper scan --cidr 192.168.1.0/24 --ports 1-65535

# 指定网卡、输出 JSON、写入数据库
./mdns-mapper scan --cidr 192.168.1.0/24 --ports 80,443,5000-5100 \
  --interface eth0 --format json --save-db
```

### 5. Web 服务模式

```bash
./mdns-mapper serve --addr :8080
```

启动后访问 `http://localhost:8080` 即可使用前端页面发起扫描、查看结果。

## 项目结构

```
cmd/mdns-mapper/       程序入口
internal/
  api/                 Gin HTTP 接口
  cli/                 Cobra CLI 命令
  config/              配置加载
  engine/              mDNS 扫描引擎核心
  logger/              日志
  model/               数据模型
  output/              输出格式化
  repository/          数据库仓储
  service/             业务服务层
pkg/netutil/           网卡工具
web/                   前端页面
configs/               配置文件
scripts/               数据库初始化脚本
docs/                  设计文档
```

## API 接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | /api/v1/scans | 发起扫描 |
| GET | /api/v1/scans | 任务列表 |
| GET | /api/v1/scans/:id | 任务详情 |
| GET | /api/v1/scans/:id/assets | 资产列表 |
| GET | /api/v1/scans/:id/export?format=text\|json | 导出 |
| GET | /api/v1/interfaces | 网卡列表 |

## 注意事项

- mDNS 组播仅在本地链路有效，无法穿透三层路由
- Windows 下需以管理员身份运行才能绑定 UDP 5353
- 端口范围指的是对 SRV 记录声明的服务端口做过滤，非逐端口探测
