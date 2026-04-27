# 基金助手 MVP

这是一个围绕基金助手原型持续演进的 Go + Hertz 项目。当前版本在原有聊天骨架基础上，新增了：

- 注册与登录
- 用户资料与默认头像
- 更换头像
- 会话按用户隔离
- MySQL 存储用户、登录态、会话和消息
- 优化过的聊天界面比例和消息气泡布局
- 基金代码精确查询
- 基金名称模糊搜索与候选展示
- 基金基础信息卡片
- 基金近期走势展示与区间切换
- 推荐占位和分析占位区域

## 启动前准备

1. 准备一个 MySQL 数据库，例如 `stock_agent`
2. 配置 MySQL 连接相关环境变量
3. 启动服务

```bash
go mod tidy
MYSQL_HOST=127.0.0.1 \
MYSQL_PORT=3306 \
MYSQL_USER=root \
MYSQL_PASSWORD=root \
MYSQL_DATABASE=stock_agent \
go run .
```

默认启动地址为 `http://127.0.0.1:8888`。

## 环境变量

- `PORT`: 服务监听端口，默认 `8888`
- `MYSQL_HOST`: MySQL 主机，默认 `127.0.0.1`
- `MYSQL_PORT`: MySQL 端口，默认 `3306`
- `MYSQL_USER`: MySQL 用户名，默认 `root`
- `MYSQL_PASSWORD`: MySQL 密码，默认 `root`
- `MYSQL_DATABASE`: MySQL 数据库名，默认 `stock_agent`
- `MYSQL_CHARSET`: MySQL 字符集，默认 `utf8mb4`
- `MYSQL_PARSE_TIME`: 是否解析时间，默认 `true`
- `MYSQL_LOCATION`: MySQL 时区参数，默认 `UTC`
- `LEGACY_CHAT_FILE`: 旧版全局聊天文件路径，仅用于提示旧数据已废弃，默认 `data/chat.json`
- `AVATAR_UPLOAD_DIR`: 用户头像上传目录，默认 `uploads`

## 当前范围

- 前端提供登录/注册页和登录后的聊天页
- 后端提供注册、登录、登出、当前用户资料、上传头像/更新头像、会话管理和问答接口
- 基金查询支持通过聊天消息触发，也提供独立基金接口，便于后续网页端和飞书入口复用
- 所有聊天记录均按用户隔离
- 当前基金数据来自东方财富公开页面/接口组合，后端统一封装在 `internal/fund` 服务层
- 推荐与智能分析当前仍为占位内容，用于先验证真实数据接入、产品流程和接口契约

## 旧数据说明

旧版 `data/chat.json` 属于匿名全局原型数据。当前版本改为 MySQL 存储后，不再读取这份文件。建议：

- 如果旧数据没有继续保留的必要，可以直接删除
- 如果需要留档，可以自行备份后再清理
- 当前版本不会自动迁移旧文件中的匿名会话到某个默认用户

## 后续方向

- 接入大模型和 Eino
- 增加更稳定的基金数据源与板块数据源
- 从占位回复演进到真正的推荐与分析能力
- 复用同一套基金服务层接入飞书机器人等多入口
