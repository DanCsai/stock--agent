## ADDED Requirements

### Requirement: 聊天结果支持卡片化展示
系统 SHALL 在聊天回复之外展示基金结构化卡片。

#### Scenario: 返回基金查询结果
- **WHEN** 用户查询某只基金
- **THEN** 前端在聊天简短回复之外展示基金信息卡片

### Requirement: 聊天结果支持走势展示区域
系统 SHALL 在基金查询结果中展示走势区域。

#### Scenario: 显示基金走势
- **WHEN** 基金查询结果包含走势数据
- **THEN** 前端展示基金走势区域并支持时间范围切换

### Requirement: 系统预留推荐与分析占位区
系统 SHALL 为后续推荐与分析能力预留固定展示区域。

#### Scenario: 显示占位区域
- **WHEN** 用户查看基金查询结果
- **THEN** 页面同时显示推荐占位区和分析占位区

