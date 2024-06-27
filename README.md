
目录

- [MOREDOC - 在线文库](#intro)
  - [技术栈](#stack)
  - [开源地址](#opensource)
  - [使用手册](#manual)
  - [演示站点](#demo)
  - [微信交流群](#wechatgroup)
  - [页面预览](#preview)
    - [首页](#preview-index)
    - [列表页](#preview-category)
    - [文档详情页](#preview-read)
    - [文档上传页](#preview-upload)
    - [搜索结果页](#preview-search)
    - [管理后台](#preview-dashboard)
  - [二次开发](#dev)
    - [环境要求](#dev-env)
    - [目录结构](#dev-tree)
    - [app.toml](#dev-config)
    - [初始化](#dev-init)
    - [管理员初始账号密码](#dev-account)
    - [发布版本](#dev-release)
  - [License](#license)
  - [鸣谢](#thanks)

<a name="intro"></a>

<a name="stack"></a>

## 技术栈

- Golang ：gin + gRPC + GORM
- Vue.js : nuxt2 + element-ui
- Database : MySQL 5.7



<a name="dev-tree"></a>

### 目录结构

> 部分目录，在程序运行时自动生成，不需要手动创建

```bash
.
├── LICENSE                 # 开源协议
├── Makefile                # 编译脚本
├── README.md               # 项目说明
├── api                     # proto api， API协议定义
├── app.example.toml        # 配置文件示例，需要复制为 app.toml
├── biz                     # 业务逻辑层，主要处理业务逻辑，实现api接口
├── cmd                     # 命令行工具
├── cache                   # 缓存相关
├── conf                    # 配置定义
├── dict                    # 结巴分词字典，用于给文档自动进行分词
├── docs                    # API文档等
├── documents               # 用户上传的文档存储目录
├── go.mod                  # go依赖管理
├── go.sum                  # go依赖管理
├── main.go                 # 项目入口
├── middleware              # 中间件
├── model                   # 数据库模型，使用gorm对数据库进行操作
├── release                 # 版本发布生成的版本会放到这里
├── service                 # 服务层，衔接cmd与biz
├── sitemap                 # 站点地图
├── third_party             # 第三方依赖，主要是proto文件
├── uploads                 # 文档文件之外的其他文件存储目录
└── util                    # 工具函数
```

<a name="dev-config"></a>

### app.toml

```
# 程序运行级别：debug、info、warn、error
level="debug"

# 日志编码方式，支持：json、console
logEncoding="console"

# 后端监听端口
port="8880"

# 数据库配置
[database]
    driver="mysql"
    dsn="root:root@tcp(localhost:3306)/moredoc?charset=utf8mb4&loc=Local&parseTime=true"
    # 生产环境，请将showSQL设置为false
    showSQL=true
    maxOpen=10
    maxIdle=10

# jwt 配置
[jwt]
    secret="moredoc"
    expireDays=365
```

<a name="dev-init"></a>

### 初始化

**后端初始化**

```
# 安装go依赖
go mod tidy

# 初始化工程依赖
make init

# 编译proto api
make api

# 修改 app.toml 文件配置
cp app.example.toml app.toml

# 编译后端
go build -o moredoc main.go

# 初始化数据库结构
./moredoc syncdb

# 运行后端(可用其他热编译工具)，监听8880端口
go run main.go serve
```

**前端初始化**

```bash
# 切换到web目录
cd web

# 安装依赖
npm install

# 运行前端，监听3000端口，浏览器访问 http://localhost:3000
npm run dev
```

<a name="dev-account"></a>

### 管理员初始账号密码

```
admin
mnt.ltd
```

<a name="dev-release"></a>

### 发布版本

以下为示例

```
# 打标签
git tag -a v1.0.0 -m "release v1.0.0"

# 推送标签
git push origin v1.0.0

# 编译前端
cd web && npm run generate

# 编译后端，编译好了的版本会放到release目录下
# 编译linux版本（Windows版本用 make buildwin）
make buildlinux
```

<a name="license"></a>

## License

开源版本基于 [Apache License 2.0](./LICENSE) 协议发布。

<a name="thanks"></a>
